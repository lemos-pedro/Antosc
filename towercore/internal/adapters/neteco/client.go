package neteco

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"

	"towercore/internal/core/interfaces"
)

// Client é o cliente HTTP para a API interna (não-NBI) do Huawei NetEco.
// Usa autenticação baseada em sessão (JSESSIONID via cookie), não a
// OpenAPI NBI oficial — ver nota em runbook/definições sobre preferência
// futura pela NBI sancionada.
type Client struct {
	baseURL  string
	username string
	password string
	http     *http.Client

	mu       sync.Mutex
	loggedIn bool
}

func NewClient(baseURL, username, password string, tlsInsecureSkipVerify bool, timeout time.Duration) *Client {
	jar, _ := cookiejar.New(nil)
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: tlsInsecureSkipVerify}}
	return &Client{
		baseURL:  strings.TrimRight(baseURL, "/"),
		username: username,
		password: password,
		http:     &http.Client{Jar: jar, Transport: tr, Timeout: timeout},
	}
}

// Login autentica explicitamente. Chamado no arranque do serviço; depois
// disso a reautenticação é automática e transparente via doGet.
func (c *Client) Login() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.doLogin()
}

// doLogin executa o POST de autenticação e confirma a sessão pelo cookie
// JSESSIONID. Deve ser chamado sempre com c.mu já detido.
func (c *Client) doLogin() error {
	form := url.Values{}
	form.Set("username", c.username)
	form.Set("value", c.password)
	form.Set("vcode", "")
	form.Set("isEncrypt", "false")

	req, err := http.NewRequest(
		"POST",
		c.baseURL+"/login/authenticate.action",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return fmt.Errorf("neteco: montar pedido de login: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		c.loggedIn = false
		return fmt.Errorf("neteco: pedido de login falhou: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		c.loggedIn = false
		return fmt.Errorf(
			"neteco: login devolveu status %d (body=%s)",
			resp.StatusCode,
			truncate(body, 300),
		)
	}

	u, _ := url.Parse(c.baseURL)
	for _, ck := range c.http.Jar.Cookies(u) {
		if ck.Name == "JSESSIONID" {
			c.loggedIn = true
			return nil
		}
	}

	c.loggedIn = false
	return fmt.Errorf(
		"neteco: login sem cookie de sessão (body=%s)",
		truncate(body, 300),
	)
}

// isSessionInvalid deteta duas formas de sessão expirada observadas no
// NetEco: página de login HTML, ou envelope JSON de erro com
// errorCode=SM_SESSIONID_INVALID (ex.: {"error":"Please login first.",
// "errorCode":"SM_SESSIONID_INVALID"}). Confirmado empiricamente em
// 2026-07-24 que este segundo formato é o mais comum e não era detetado
// antes, causando falsos "campo 'data' ausente" em vez de reautenticação.
func isSessionInvalid(body []byte) bool {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return false
	}

	if strings.HasPrefix(trimmed, "<") {
		return true
	}

	var errResp struct {
		ErrorCode string `json:"errorCode"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil {
		if errResp.ErrorCode == "SM_SESSIONID_INVALID" {
			return true
		}
	}

	return false
}

// doGet executa um GET autenticado, reautenticando uma vez se a sessão
// tiver expirado. Não distingue "sessão expirada" de "resposta JSON válida
// mas sem os campos esperados" — essa distinção é feita pelo chamador
// (ver mapBatteryStatus/mapSiteCounterInfo), porque um payload JSON sem
// 'data' não é necessariamente um problema de sessão.
func (c *Client) doGet(reqURL string) ([]byte, error) {
	c.mu.Lock()
	if !c.loggedIn {
		if err := c.doLogin(); err != nil {
			c.mu.Unlock()
			return nil, fmt.Errorf("neteco: reautenticação inicial falhou: %w", err)
		}
	}
	c.mu.Unlock()

	body, retry, err := c.attemptGet(reqURL)
	if err != nil {
		return nil, err
	}
	if !retry {
		return body, nil
	}

	c.mu.Lock()
	loginErr := c.doLogin()
	c.mu.Unlock()
	if loginErr != nil {
		return nil, fmt.Errorf("neteco: reautenticação após sessão expirada falhou: %w", loginErr)
	}

	body, _, err = c.attemptGet(reqURL)
	if err != nil {
		return nil, fmt.Errorf("neteco: pedido após reautenticação falhou: %w", err)
	}
	return body, nil
}

func (c *Client) attemptGet(reqURL string) (body []byte, needsRetry bool, err error) {
	resp, err := c.http.Get(reqURL)
	if err != nil {
		return nil, false, fmt.Errorf("neteco: GET %s: %w", reqURL, err)
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, false, fmt.Errorf("neteco: ler corpo da resposta: %w", err)
	}

	// Verificar sessão inválida ANTES do status code: o NetEco devolve
	// SM_SESSIONID_INVALID com HTTP 203 (não 200), por isso a checagem
	// de status não pode "engolir" este caso antes de olharmos o body.
	if isSessionInvalid(body) {
		return nil, true, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf(
			"neteco: status %d em %s (body=%s)",
			resp.StatusCode,
			reqURL,
			truncate(body, 300),
		)
	}

	return body, false, nil
}

// FetchBatteryStatus consulta SOC/SOH/tempo de backup da bateria de um
// site pelo seu NEID (ex.: "NE=33554471").
//
// NOTA: sites sem a licença isSocLicense do NetEco (ou com siteDn em
// formato não reconhecido pelo endpoint) devolvem JSON 200 válido mas
// sem a chave "data" — não é um erro de sessão, é resposta legítima do
// NetEco. Confirmar licença/formato antes de assumir bug de código.
func (c *Client) FetchBatteryStatus(siteDn string) (*interfaces.BatteryStatus, error) {
	reqURL := fmt.Sprintf(
		"%s/rest/batteryManager/sohTreeRoaService/handleTreeClick?siteDn=%s",
		c.baseURL,
		url.QueryEscape(siteDn),
	)

	body, err := c.doGet(reqURL)
	if err != nil {
		return nil, err
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("neteco: resposta de bateria não é JSON válido (raw=%s)", truncate(body, 300))
	}

	status, err := mapBatteryStatus(raw)
	if err != nil {
		return nil, fmt.Errorf("%w (raw=%s)", err, truncate(body, 300))
	}
	return status, nil
}

// FetchSiteCounterInfo consulta contadores de energia (tensão/corrente
// DC, retificador) de um site pelo seu NEID.
func (c *Client) FetchSiteCounterInfo(siteDn string) (*interfaces.SiteCounterInfo, error) {
	params := fmt.Sprintf(`{"siteDn":"%s","jobs":"siteCounterInfo"}`, siteDn)
	reqURL := fmt.Sprintf(
		"%s/rest/dashboard/moduleHub/analyseJobs?params=%s",
		c.baseURL,
		url.QueryEscape(params),
	)

	body, err := c.doGet(reqURL)
	if err != nil {
		return nil, err
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("neteco: resposta de energia não é JSON válido (raw=%s)", truncate(body, 300))
	}

	info, err := mapSiteCounterInfo(raw)
	if err != nil {
		return nil, fmt.Errorf("%w (raw=%s)", err, truncate(body, 300))
	}

	return &interfaces.SiteCounterInfo{
		DCOutputVoltage:  info.DCOutputVoltage,
		DCLoadCurrent:    info.DCLoadCurrent,
		RectifierCurrent: info.RectifierCurrent,
	}, nil
}

func truncate(b []byte, n int) string {
	s := string(b)
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}