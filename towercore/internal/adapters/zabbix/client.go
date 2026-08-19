package zabbix

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client é um cliente mínimo para a API JSON-RPC do Zabbix.
// Segue o mesmo padrão dos outros adapters (nagios, neteco): construtor
// com credenciais + timeout, métodos de alto nível que escondem o
// protocolo JSON-RPC.
type Client struct {
	baseURL    string
	httpClient *http.Client

	// Preenche UM dos dois:
	authToken string // se já tiveres um API token gerado no Zabbix (recomendado)
	username  string
	password  string

	token string // token de sessão obtido via user.login, cacheado em memória
}

func NewClient(baseURL, username, password, authToken string, timeout time.Duration) *Client {
	return &Client{
		baseURL:    baseURL,
		username:   username,
		password:   password,
		authToken:  authToken,
		httpClient: &http.Client{Timeout: timeout},
	}
}

type rpcRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      int         `json:"id"`
	Auth    string      `json:"auth,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error,omitempty"`
	ID      int             `json:"id"`
}

// call faz um pedido JSON-RPC genérico. auth indica se deve anexar o
// token de sessão/API token no campo "auth" do pedido (a maioria dos
// métodos exige; user.login e apiinfo.version não).
func (c *Client) call(method string, params interface{}, auth bool, out interface{}) error {
	req := rpcRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      1,
	}

	if auth {
		if c.authToken != "" {
			req.Auth = c.authToken
		} else {
			req.Auth = c.token
		}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("zabbix: marshal request: %w", err)
	}

	httpReq, err := http.NewRequest(
		http.MethodPost,
		c.baseURL,
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("zabbix: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json-rpc")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("zabbix: request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("zabbix: HTTP status %s", resp.Status)
	}

	var rpcResp rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return fmt.Errorf("zabbix: decode response: %w", err)
	}

	if rpcResp.Error != nil {
		return fmt.Errorf(
			"zabbix: api error %d: %s (%s)",
			rpcResp.Error.Code,
			rpcResp.Error.Message,
			rpcResp.Error.Data,
		)
	}

	if out != nil {
		if err := json.Unmarshal(rpcResp.Result, out); err != nil {
			return fmt.Errorf("zabbix: unmarshal result: %w", err)
		}
	}

	return nil
}

// Login autentica via user.login e guarda o token de sessão em memória.
// Não é necessário se usares authToken (API token fixo gerado no Zabbix),
// que é o método recomendado — não expira por inatividade da mesma forma.
func (c *Client) Login() error {
	if c.authToken != "" {
		return nil // já autenticado via API token, nada a fazer
	}

	params := map[string]string{
		"username": c.username,
		"password": c.password,
	}

	var token string
	if err := c.call("user.login", params, false, &token); err != nil {
		return fmt.Errorf("zabbix: login failed: %w", err)
	}

	c.token = token
	return nil
}

// Host representa o subconjunto de campos de host.get que interessam
// para localizar os rádios de Benguela/Huambo.
type Host struct {
	HostID string `json:"hostid"`
	Host   string `json:"host"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Interfaces []HostInterface `json:"interfaces"`
}

type HostInterface struct {
	InterfaceID string `json:"interfaceid"`
	IP          string `json:"ip"`
	DNS         string `json:"dns"`
	Main        string `json:"main"`
	Type        string `json:"type"`
}

// GetHostsByNameFilter procura hosts cujo nome contenha algum dos termos
// dados (ex.: []string{"Benguela", "Huambo"}). Usa "search" (substring),
// não igualdade exata.
func (c *Client) GetHostsByNameFilter(nameSearch []string) ([]Host, error) {
	params := map[string]interface{}{
		"output": []string{"hostid", "host", "name", "status"},
		"selectInterfaces": []string{"interfaceid", "ip", "dns", "main", "type"},
		"search": map[string]interface{}{
			"name": nameSearch,
		},
		"searchByAny": true, // OR entre os termos, não AND
	}

	var hosts []Host
	if err := c.call("host.get", params, true, &hosts); err != nil {
		return nil, fmt.Errorf("zabbix: host.get failed: %w", err)
	}

	return hosts, nil
}

// Item representa o subconjunto de campos de item.get que interessam:
// key_ identifica a métrica (ex.: "rsl.rx[ifIndex]"), lastvalue é o
// valor mais recente já processado pelo Zabbix.
type Item struct {
	ItemID    string `json:"itemid"`
	HostID    string `json:"hostid"`
	Name      string `json:"name"`
	Key       string `json:"key_"`
	LastValue string `json:"lastvalue"`
	LastClock string `json:"lastclock"`
	Units     string `json:"units"`
}

// GetItemsForHost devolve todos os items (métricas) de um host, com o
// último valor já coletado pelo Zabbix — não faz polling SNMP novo,
// só lê o que o Zabbix já tem em cache/histórico.
func (c *Client) GetItemsForHost(hostID string) ([]Item, error) {
	params := map[string]interface{}{
		"output":  []string{"itemid", "hostid", "name", "key_", "lastvalue", "lastclock", "units"},
		"hostids": hostID,
	}

	var items []Item
	if err := c.call("item.get", params, true, &items); err != nil {
		return nil, fmt.Errorf("zabbix: item.get failed: %w", err)
	}

	return items, nil
}
