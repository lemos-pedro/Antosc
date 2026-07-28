package notification

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// resendSender implementa EmailSender usando a API do Resend
// (https://resend.com/docs/api-reference/emails/send-email), em vez de SMTP
// direto. Mais simples de configurar (só precisa de uma API key) e com
// plano gratuito de 3000 emails/mês, suficiente para relatórios periódicos.
type resendSender struct {
	apiKey string
	from   string
	http   *http.Client
}

func NewResendSender(apiKey, from string) EmailSender {
	return &resendSender{
		apiKey: apiKey,
		from:   from,
		http:   &http.Client{Timeout: 20 * time.Second},
	}
}

type resendAttachment struct {
	Filename string `json:"filename"`
	Content  string `json:"content"` // base64
}

type resendRequest struct {
	From        string              `json:"from"`
	To          []string            `json:"to"`
	Subject     string              `json:"subject"`
	HTML        string              `json:"html"`
	Attachments []resendAttachment  `json:"attachments,omitempty"`
}

func (s *resendSender) Send(to []string, subject, htmlBody string, attachments []Attachment) error {
	req := resendRequest{
		From:    s.from,
		To:      to,
		Subject: subject,
		HTML:    htmlBody,
	}
	for _, a := range attachments {
		req.Attachments = append(req.Attachments, resendAttachment{
			Filename: a.Filename,
			Content:  base64.StdEncoding.EncodeToString(a.Data),
		})
	}

	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequest(http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Authorization", "Bearer "+s.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.http.Do(httpReq)
	if err != nil {
		return fmt.Errorf("resend indisponível: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resend devolveu status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
