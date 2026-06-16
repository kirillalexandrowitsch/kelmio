package emailoutbox

import (
	"errors"
	"fmt"
	"strings"

	"team-task-tracker/backend/internal/mailer"
)

var ErrTemplateInvalid = errors.New("email template is invalid")

func Render(email Email) (mailer.Message, error) {
	switch email.EmailType {
	case TypeSystemTest:
		return renderSystemTest(email)
	default:
		return mailer.Message{}, fmt.Errorf("%w: unknown email type", ErrTemplateInvalid)
	}
}

func renderSystemTest(email Email) (mailer.Message, error) {
	subject, ok := stringField(email.TemplateData, "subject")
	if !ok {
		return mailer.Message{}, fmt.Errorf("%w: subject is required", ErrTemplateInvalid)
	}
	textBody, _ := stringField(email.TemplateData, "text_body")
	htmlBody, _ := stringField(email.TemplateData, "html_body")
	if strings.TrimSpace(textBody) == "" && strings.TrimSpace(htmlBody) == "" {
		return mailer.Message{}, fmt.Errorf("%w: text_body or html_body is required", ErrTemplateInvalid)
	}
	return mailer.Message{
		To:       []string{email.RecipientEmail},
		Subject:  subject,
		TextBody: textBody,
		HTMLBody: htmlBody,
	}, nil
}

func stringField(values map[string]any, key string) (string, bool) {
	raw, ok := values[key]
	if !ok {
		return "", false
	}
	value, ok := raw.(string)
	if !ok || strings.TrimSpace(value) == "" {
		return "", false
	}
	return strings.TrimSpace(value), true
}
