// Package notification envia o relatório semanal e outros alertas por
// email, com o Excel em anexo.
package notification

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime"
	"net/smtp"
)

type EmailConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

type Attachment struct {
	Filename string
	Data     []byte
	MimeType string
}

type EmailSender interface {
	Send(to []string, subject, htmlBody string, attachments []Attachment) error
}

type smtpSender struct {
	cfg EmailConfig
}

func NewEmailSender(cfg EmailConfig) EmailSender {
	return &smtpSender{cfg: cfg}
}

// Send monta um email MIME multipart simples (corpo HTML + anexos
// codificados em base64) e envia via SMTP autenticado. Suficiente para
// relatórios periódicos; para volume alto de envio, considera um serviço
// dedicado (SES, SendGrid, etc.) mais tarde.
func (s *smtpSender) Send(to []string, subject, htmlBody string, attachments []Attachment) error {
	boundary := "AIP-BOUNDARY-42"

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "From: %s\r\n", s.cfg.From)
	fmt.Fprintf(&buf, "To: %s\r\n", joinAddresses(to))
	fmt.Fprintf(&buf, "Subject: %s\r\n", mime.QEncoding.Encode("UTF-8", subject))
	fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&buf, "Content-Type: multipart/mixed; boundary=%s\r\n\r\n", boundary)

	fmt.Fprintf(&buf, "--%s\r\n", boundary)
	fmt.Fprintf(&buf, "Content-Type: text/html; charset=UTF-8\r\n\r\n")
	buf.WriteString(htmlBody)
	buf.WriteString("\r\n")

	for _, att := range attachments {
		fmt.Fprintf(&buf, "--%s\r\n", boundary)
		fmt.Fprintf(&buf, "Content-Type: %s; name=%q\r\n", att.MimeType, att.Filename)
		fmt.Fprintf(&buf, "Content-Transfer-Encoding: base64\r\n")
		fmt.Fprintf(&buf, "Content-Disposition: attachment; filename=%q\r\n\r\n", att.Filename)
		buf.WriteString(base64.StdEncoding.EncodeToString(att.Data))
		buf.WriteString("\r\n")
	}
	fmt.Fprintf(&buf, "--%s--\r\n", boundary)

	auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	addr := s.cfg.Host + ":" + s.cfg.Port

	return smtp.SendMail(addr, auth, s.cfg.From, to, buf.Bytes())
}

func joinAddresses(addrs []string) string {
	out := ""
	for i, a := range addrs {
		if i > 0 {
			out += ", "
		}
		out += a
	}
	return out
}
