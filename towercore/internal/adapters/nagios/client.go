package nagios

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"towercore/internal/core/interfaces"
)

// Bitmask confirmado na fonte do Nagios Core:
// github.com/NagiosEnterprises/nagioscore/blob/master/cgi/statusjson.c
const (
	bitHostPending     = 1
	bitHostUp          = 2
	bitHostDown        = 4
	bitHostUnreachable = 8
)

type Client struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
}

func NewClient(baseURL, username, password string, timeout time.Duration) *Client {
	return &Client{
		baseURL:    baseURL,
		username:   username,
		password:   password,
		httpClient: &http.Client{Timeout: timeout},
	}
}

type statusJSONResponse struct {
	Result struct {
		TypeCode int    `json:"type_code"`
		Message  string `json:"message"`
	} `json:"result"`
	Data struct {
		Host struct {
			Name                 string `json:"name"`
			PluginOutput         string `json:"plugin_output"`
			Status               int    `json:"status"`
			LastCheck            int64  `json:"last_check"`
			LastStateChange      int64  `json:"last_state_change"`
			NotificationsEnabled bool   `json:"notifications_enabled"`
			ChecksEnabled        bool   `json:"checks_enabled"`
		} `json:"host"`
	} `json:"data"`
}

func (c *Client) FetchHostStatus(ctx context.Context, hostname string) (interfaces.HostStatus, error) {
	endpoint := fmt.Sprintf("%s/nagios/cgi-bin/statusjson.cgi", c.baseURL)
	q := url.Values{}
	q.Set("query", "host")
	q.Set("hostname", hostname)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+q.Encode(), nil)
	if err != nil {
		return interfaces.HostStatus{}, err
	}
	req.SetBasicAuth(c.username, c.password)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return interfaces.HostStatus{}, fmt.Errorf("nagios request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return interfaces.HostStatus{}, fmt.Errorf("nagios returned status %d for host %s", resp.StatusCode, hostname)
	}

	var parsed statusJSONResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return interfaces.HostStatus{}, fmt.Errorf("failed to decode nagios response: %w", err)
	}
	if parsed.Result.TypeCode != 0 {
		return interfaces.HostStatus{}, fmt.Errorf("nagios query failed: %s", parsed.Result.Message)
	}

	h := parsed.Data.Host
	return interfaces.HostStatus{
		Hostname:     h.Name,
		State:        mapHostState(h.Status),
		PluginOutput: h.PluginOutput,
		// Nagios statusjson.cgi devolve epoch em SEGUNDOS, não ms.
		// Usar UnixMilli aqui dava datas perto de 1970.
		LastCheck:            time.Unix(h.LastCheck, 0),
		LastStateChange:      time.Unix(h.LastStateChange, 0),
		NotificationsEnabled: h.NotificationsEnabled,
		ChecksEnabled:        h.ChecksEnabled,
	}, nil
}

func mapHostState(bitmask int) interfaces.HostState {
	switch bitmask {
	case bitHostUp:
		return interfaces.HostStateUp
	case bitHostDown:
		return interfaces.HostStateDown
	case bitHostUnreachable:
		return interfaces.HostStateUnreachable
	case bitHostPending:
		return interfaces.HostStatePending
	default:
		return interfaces.HostStateUnknown
	}
}
