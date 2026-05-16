package email

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log"

	brevo "github.com/getbrevo/brevo-go/lib"
)

// ErrTemplateFailed marks errors that are unrecoverable — retrying the same
// Kafka message won't help (bad template syntax, missing data fields, or
// malformed JSON payload).
var ErrTemplateFailed = errors.New("template failed")

type EmailMessage struct {
	To           string
	Subject      string
	TemplateName string
	TemplateData interface{}
}

type EmailSender struct {
	client    *brevo.APIClient
	apiKey    string
	from      string
	templates fs.FS
}

// NewEmailSender creates an EmailSender that renders templates from the
// provided fs.FS (typically an embed.FS sub-tree) and sends via Brevo REST API.
//
// The api-key header is supplied per-request through the context
// (see ContextAPIKey usage in SendEmail). Do NOT also set it via
// cfg.AddDefaultHeader("api-key", ...) — the SDK's prepareRequest calls
// Header.Add for both, producing a duplicated `api-key: <k>, <k>` header
// that Brevo rejects with 401 Unauthorized.
func NewEmailSender(apiKey, from string, templates fs.FS) *EmailSender {
	cfg := brevo.NewConfiguration()
	client := brevo.NewAPIClient(cfg)
	return &EmailSender{
		client:    client,
		apiKey:    apiKey,
		from:      from,
		templates: templates,
	}
}

func (s *EmailSender) SendEmail(msg EmailMessage) error {
	tmpl, err := template.ParseFS(s.templates, msg.TemplateName)
	if err != nil {
		return fmt.Errorf("%w: parse %s: %v", ErrTemplateFailed, msg.TemplateName, err)
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, msg.TemplateName, msg.TemplateData); err != nil {
		return fmt.Errorf("%w: execute %s: %v", ErrTemplateFailed, msg.TemplateName, err)
	}

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

	// Pass the API key via context — this is the authentication mechanism the
	// brevo-go SDK actually reads in SendTransacEmail (ctx.Value(ContextAPIKey)).
	// DefaultHeader is a fallback but the SDK method only populates
	// localVarHeaderParams["api-key"] from the context, not from DefaultHeader.
	ctx := context.WithValue(context.Background(), brevo.ContextAPIKey, brevo.APIKey{
		Key: s.apiKey,
	})

	log.Printf("[email] sending %s to %s (subject: %s)", msg.TemplateName, msg.To, msg.Subject)

	result, httpResp, err := s.client.TransactionalEmailsApi.SendTransacEmail(ctx, req)
	if err != nil {
		log.Printf("[email] send failed to %s: %v", msg.To, err)
		return fmt.Errorf("brevo send: %w", err)
	}

	statusCode := 0
	if httpResp != nil {
		statusCode = httpResp.StatusCode
	}

	if result.MessageId == "" {
		log.Printf("[email] WARNING — Brevo returned http=%d but messageId is empty for %s; email was NOT queued. Check Brevo dashboard → Transactional → Logs.", statusCode, msg.To)
	} else {
		log.Printf("[email] sent %s to %s | http=%d | brevo messageId=%s", msg.TemplateName, msg.To, statusCode, result.MessageId)
	}

	return nil
}

// SendRaw sends a plain HTML email body without a template file.
// Used by the debug test endpoint.
func (s *EmailSender) SendRaw(to, subject, htmlBody string) (messageId string, httpStatus int, err error) {
	sender := brevo.SendSmtpEmailSender{Name: "EventsLK", Email: s.from}
	req := brevo.SendSmtpEmail{
		Sender:      &sender,
		To:          []brevo.SendSmtpEmailTo{{Email: to}},
		Subject:     subject,
		HtmlContent: htmlBody,
	}

	ctx := context.WithValue(context.Background(), brevo.ContextAPIKey, brevo.APIKey{
		Key: s.apiKey,
	})

	result, httpResp, apiErr := s.client.TransactionalEmailsApi.SendTransacEmail(ctx, req)
	if httpResp != nil {
		httpStatus = httpResp.StatusCode
	}
	if apiErr != nil {
		return "", httpStatus, apiErr
	}
	return result.MessageId, httpStatus, nil
}
