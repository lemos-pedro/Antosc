// Package hizima implementa o adapter para a API ZMACS da Hizima
// (Anglobal_ZMACS-API-Integration-Document-v1_0-EN, v1.0.0, 2026-08-31).
//
// Propositadamente read-only: a API não expõe trancar/destrancar remoto.
// Essa função pertence à app mobile do técnico (BLE local). Este adapter
// serve apenas para consolidar estado, auditoria e work orders no TowerCore.
package hizima

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
)

const (
	defaultTimeout = 10 * time.Second
)

// Config contém os dados de acesso obtidos junto da Hizima.
// Security NUNCA deve ser versionado — carregar de variável de ambiente
// (ex.: HIZIMA_SECURITY), nunca hardcoded.
type Config struct {
	Host      string // ex.: "https://antosc.hizima.com"
	ClientID  string // ex.: "anglobal"
	Security  string // segredo — carregar de env
	Username  string // documentação mostra vazio nos exemplos; manter configurável
	Password  string
}

// Client implementa interfaces.AccessControlPort contra a API ZMACS.
type Client struct {
	cfg        Config
	httpClient *http.Client
}

// NewClient cria um client Hizima. httpClient pode ser nil para usar um default
// com timeout — nunca usar http.DefaultClient sem timeout em produção.
func NewClient(cfg Config, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}
	return &Client{cfg: cfg, httpClient: httpClient}
}

var _ interfaces.AccessControlPort = (*Client)(nil)

// generateToken implementa o esquema documentado:
// token = base64(SHA1(clientId+security+username+password+dayTimestamp+"hzm"))[:20]
// dayTimestamp = ms/(1000*60*60*24), portanto o token é estável durante o dia
// inteiro (UTC) e muda à meia-noite — não há necessidade de regenerar por pedido,
// mas regenerar por pedido é inofensivo e mais simples de manter correto.
func (c *Client) generateToken(now time.Time) string {
	dayTimestamp := now.UnixMilli() / (1000 * 60 * 60 * 24)
	raw := fmt.Sprintf("%s%s%s%s%d%s",
		c.cfg.ClientID, c.cfg.Security, c.cfg.Username, c.cfg.Password, dayTimestamp, "hzm")
	sum := sha1.Sum([]byte(raw))
	encoded := base64.StdEncoding.EncodeToString(sum[:])
	if len(encoded) > 20 {
		encoded = encoded[:20]
	}
	return encoded
}

func (c *Client) setAuthHeaders(req *http.Request) {
	token := c.generateToken(time.Now().UTC())
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("clientId", c.cfg.ClientID)
	req.Header.Set("username", c.cfg.Username)
	req.Header.Set("password", c.cfg.Password)
}

// apiError representa uma falha reportada pela API (envelope "failed" ou HTTP não-2xx).
type apiError struct {
	StatusCode int
	Message    string
}

func (e *apiError) Error() string {
	return fmt.Sprintf("hizima api error (http %d): %s", e.StatusCode, e.Message)
}

func (c *Client) doGet(ctx context.Context, path string, query url.Values) ([]byte, error) {
	full := c.cfg.Host + path
	if len(query) > 0 {
		full += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, full, nil)
	if err != nil {
		return nil, fmt.Errorf("hizima: build request: %w", err)
	}
	c.setAuthHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("hizima: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("hizima: read body: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, &apiError{StatusCode: 404, Message: "not existing"}
	}
	if resp.StatusCode != http.StatusOK {
		// tentar extrair a mensagem do envelope de falha documentado
		var fail failureEnvelope
		if jsonErr := json.Unmarshal(body, &fail); jsonErr == nil && fail.Message != "" {
			return nil, &apiError{StatusCode: resp.StatusCode, Message: fail.Message}
		}
		return nil, &apiError{StatusCode: resp.StatusCode, Message: string(body)}
	}
	return body, nil
}

func setPage(q url.Values, page interfaces.PageRequest) {
	q.Set("current", strconv.Itoa(page.Current))
	q.Set("size", strconv.Itoa(page.Size))
}

// GetLockStatus consulta GET /clientExchangeApi/lock/statusBySite.
func (c *Client) GetLockStatus(ctx context.Context, filter interfaces.LockStatusFilter, page interfaces.PageRequest) (interfaces.Page[domain.Lock], error) {
	q := url.Values{}
	setPage(q, page)
	if filter.StationNo != "" {
		q.Set("sno", filter.StationNo)
	}
	if filter.StationID != nil {
		q.Set("station", strconv.FormatInt(*filter.StationID, 10))
	}
	if filter.LockName != "" {
		q.Set("lockName", filter.LockName)
	}
	if filter.DeviceID != "" {
		q.Set("deviceId", filter.DeviceID)
	}
	if filter.OpenState != nil {
		q.Set("openState", strconv.Itoa(int(*filter.OpenState)))
	}

	body, err := c.doGet(ctx, "/clientExchangeApi/lock/statusBySite", q)
	if err != nil {
		return interfaces.Page[domain.Lock]{}, err
	}

	var env envelope[lockStatusDTO]
	if err := json.Unmarshal(body, &env); err != nil {
		return interfaces.Page[domain.Lock]{}, fmt.Errorf("hizima: decode lock status: %w", err)
	}

	records := make([]domain.Lock, 0, len(env.Data.Records))
	for _, r := range env.Data.Records {
		records = append(records, toDomainLock(r))
	}
	return interfaces.Page[domain.Lock]{
		Records: records,
		Total:   env.Data.Total,
		Size:    env.Data.Size,
		Current: env.Data.Current,
		Pages:   env.Data.Pages,
	}, nil
}

// GetLockEvents consulta GET /clientExchangeApi/lockOpenLog/page.
func (c *Client) GetLockEvents(ctx context.Context, filter interfaces.LockEventFilter, page interfaces.PageRequest) (interfaces.Page[domain.LockEvent], error) {
	q := url.Values{}
	setPage(q, page)
	if filter.TicketID != nil {
		q.Set("ticketId", strconv.FormatInt(*filter.TicketID, 10))
	}
	if filter.TicketUID != "" {
		q.Set("ticketUid", filter.TicketUID)
	}
	if filter.LockID != nil {
		q.Set("lockId", strconv.FormatInt(*filter.LockID, 10))
	}
	if filter.LockName != "" {
		q.Set("lockName", filter.LockName)
	}
	if filter.DeviceID != "" {
		q.Set("deviceId", filter.DeviceID)
	}
	if filter.StationID != nil {
		q.Set("station", strconv.FormatInt(*filter.StationID, 10))
	}
	if filter.StationName != "" {
		q.Set("stationName", filter.StationName)
	}
	if filter.StationNo != "" {
		q.Set("sno", filter.StationNo)
	}
	if filter.OperatorID != nil {
		q.Set("operator", strconv.FormatInt(*filter.OperatorID, 10))
	}
	if filter.OperatorName != "" {
		q.Set("operatorName", filter.OperatorName)
	}
	if filter.RealName != "" {
		q.Set("realName", filter.RealName)
	}
	if filter.EventType != nil {
		q.Set("operType", strconv.Itoa(int(*filter.EventType)))
	}
	if filter.StartTime != "" {
		q.Set("startTime", filter.StartTime)
	}
	if filter.EndTime != "" {
		q.Set("endTime", filter.EndTime)
	}

	body, err := c.doGet(ctx, "/clientExchangeApi/lockOpenLog/page", q)
	if err != nil {
		return interfaces.Page[domain.LockEvent]{}, err
	}

	var env envelope[lockEventDTO]
	if err := json.Unmarshal(body, &env); err != nil {
		return interfaces.Page[domain.LockEvent]{}, fmt.Errorf("hizima: decode lock events: %w", err)
	}

	records := make([]domain.LockEvent, 0, len(env.Data.Records))
	for _, r := range env.Data.Records {
		records = append(records, toDomainLockEvent(r))
	}
	return interfaces.Page[domain.LockEvent]{
		Records: records,
		Total:   env.Data.Total,
		Size:    env.Data.Size,
		Current: env.Data.Current,
		Pages:   env.Data.Pages,
	}, nil
}

// GetWorkOrders consulta GET /clientExchangeApi/ticket/statusPage.
func (c *Client) GetWorkOrders(ctx context.Context, filter interfaces.WorkOrderFilter, page interfaces.PageRequest) (interfaces.Page[domain.WorkOrder], error) {
	q := url.Values{}
	setPage(q, page)
	if filter.UID != "" {
		q.Set("uid", filter.UID)
	}
	if filter.StationID != nil {
		q.Set("station", strconv.FormatInt(*filter.StationID, 10))
	}
	if filter.StationName != "" {
		q.Set("stationName", filter.StationName)
	}
	if filter.AuthResult != nil {
		q.Set("authResult", strconv.Itoa(*filter.AuthResult))
	}
	if filter.ApplicantAcc != "" {
		q.Set("applyOperatorName", filter.ApplicantAcc)
	}
	if filter.StartTime != "" {
		q.Set("starttime", filter.StartTime)
	}
	if filter.EndTime != "" {
		q.Set("endtime", filter.EndTime)
	}

	body, err := c.doGet(ctx, "/clientExchangeApi/ticket/statusPage", q)
	if err != nil {
		return interfaces.Page[domain.WorkOrder]{}, err
	}

	var env envelope[workOrderDTO]
	if err := json.Unmarshal(body, &env); err != nil {
		return interfaces.Page[domain.WorkOrder]{}, fmt.Errorf("hizima: decode work orders: %w", err)
	}

	records := make([]domain.WorkOrder, 0, len(env.Data.Records))
	for _, r := range env.Data.Records {
		records = append(records, toDomainWorkOrder(r))
	}
	return interfaces.Page[domain.WorkOrder]{
		Records: records,
		Total:   env.Data.Total,
		Size:    env.Data.Size,
		Current: env.Data.Current,
		Pages:   env.Data.Pages,
	}, nil
}