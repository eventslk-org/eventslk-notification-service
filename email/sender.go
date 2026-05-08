package email

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"path/filepath"

	"gopkg.in/gomail.v2"
)

type EmailMessage struct {
	To           string
	Subject      string
	TemplateName string
	TemplateData interface{}
}

type EmailSender struct {
	dialer       *gomail.Dialer
	from         string
	templatePath string
}

func NewEmailSender(host string, port int, username, password, from, templatePath string) *EmailSender {
	d := gomail.NewDialer(host, port, username, password)
	return &EmailSender{
		dialer:       d,
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

	m := gomail.NewMessage()
	m.SetHeader("From", s.from)
	m.SetHeader("To", msg.To)
	m.SetHeader("Subject", msg.Subject)
	m.SetBody("text/html", buf.String())

	if err := s.dialer.DialAndSend(m); err != nil {
		log.Printf("[email] failed to send to %s (subject: %s): %v", msg.To, msg.Subject, err)
		return fmt.Errorf("send email: %w", err)
	}

	log.Printf("[email] sent %s to %s", msg.TemplateName, msg.To)
	return nil
}
