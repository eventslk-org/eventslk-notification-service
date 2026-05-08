package email

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log"
	"path/filepath"

	brevo "github.com/getbrevo/brevo-go/lib"
)

type EmailMessage struct {
	To           string
	Subject      string
	TemplateName string
	TemplateData interface{}
}

type EmailSender struct {
	client       *brevo.APIClient
	from         string
	templatePath string
}

func NewEmailSender(apiKey, from, templatePath string) *EmailSender {
	cfg := brevo.NewConfiguration()
	// Add API key header used by Brevo SDK
	cfg.AddDefaultHeader("api-key", apiKey)
	client := brevo.NewAPIClient(cfg)
	return &EmailSender{
		client:       client,
		from:         from,
		templatePath: templatePath,
	}
}

func (s *EmailSender) SendEmail(msg EmailMessage) error {
	tmplFile := filepath.Join(s.templatePath, msg.TemplateName)
	tmpl, err := template.ParseFiles(tmplFile)
	if err != nil {
		return fmt.Errorf("parse template %s: %w", msg.TemplateName, err)
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, msg.TemplateName, msg.TemplateData); err != nil {
		return fmt.Errorf("execute template %s: %w", msg.TemplateName, err)
	}

	// Build the Brevo transactional SMTP email request. Use sender name 'EventsLK' and the configured from address.
	sender := brevo.SendSmtpEmailSender{
		Name:  "EventsLK",
		Email: s.from,
	}

	to := []brevo.SendSmtpEmailTo{{Email: msg.To}}

	req := brevo.SendSmtpEmail{
		Sender:      &sender,
		To:          to,
		Subject:     msg.Subject,
		HtmlContent: buf.String(),
	}

	_, _, err = s.client.TransactionalEmailsApi.SendTransacEmail(context.Background(), req)
	if err != nil {
		log.Printf("[email] failed to send to %s (subject: %s): %v", msg.To, msg.Subject, err)
		return fmt.Errorf("send email via brevo: %w", err)
	}

	log.Printf("[email] sent %s to %s via Brevo", msg.TemplateName, msg.To)
	return nil
}
