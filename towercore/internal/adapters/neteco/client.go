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

	"towercore/internal/core/interfaces"
)

type Client struct {
	baseURL  string
	username string
	password string
	http     *http.Client
}

func NewClient(baseURL, username, password string) *Client {
	jar, _ := cookiejar.New(nil)
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	return &Client{
		baseURL:  baseURL,
		username: username,
		password: password,
		http:     &http.Client{Jar: jar, Transport: tr},
	}
}

func (c *Client) Login() error {
	form := url.Values{}
	form.Set("username", c.username)
	form.Set("value", c.password)
	form.Set("vcode", "")
	form.Set("isEncrypt", "false")

	req, err := http.NewRequest("POST", c.baseURL+"/login/authenticate.action", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.ReadAll(resp.Body)

	u, _ := url.Parse(c.baseURL)
	for _, ck := range c.http.Jar.Cookies(u) {
		if ck.Name == "JSESSIONID" {
			return nil
		}
	}
	return fmt.Errorf("login falhou: sem cookie de sessão")
}

func (c *Client) FetchBatteryStatus(siteDn string) (*interfaces.BatteryStatus, error) {
	reqURL := fmt.Sprintf("%s/rest/batteryManager/sohTreeRoaService/handleTreeClick?siteDn=%s",
		c.baseURL, url.QueryEscape(siteDn))
	resp, err := c.http.Get(reqURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("resposta inválida: %s", string(body))
	}
	return mapBatteryStatus(raw)
}