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
	case TypePasswordReset:
		return renderPasswordReset(email)
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

func renderPasswordReset(email Email) (mailer.Message, error) {
	resetURL, ok := stringField(email.TemplateData, "reset_url")
	if !ok {
		resetURL, ok = stringField(email.TemplateData, "reset_url_path")
	}
	if !ok {
		return mailer.Message{}, fmt.Errorf("%w: reset_url is required", ErrTemplateInvalid)
	}
	displayName, _ := stringField(email.TemplateData, "display_name")
	if displayName == "" {
		displayName = "there"
	}
	textBody := fmt.Sprintf("Hi %s,\n\nUse this link to reset your Team Task Tracker password:\n%s\n\nIf you did not request this reset, you can ignore this email.", displayName, resetURL)
	htmlBody := fmt.Sprintf("<p>Hi %s,</p><p>Use this link to reset your Team Task Tracker password:</p><p><a href=\"%s\">Reset password</a></p><p>If you did not request this reset, you can ignore this email.</p>", htmlEscape(displayName), htmlEscape(resetURL))
	return mailer.Message{
		To:       []string{email.RecipientEmail},
		Subject:  "Reset your Team Task Tracker password",
		TextBody: textBody,
		HTMLBody: htmlBody,
	}, nil
}

func htmlEscape(value string) string {
	value = strings.ReplaceAll(value, "&", "&amp;")
	value = strings.ReplaceAll(value, "<", "&lt;")
	value = strings.ReplaceAll(value, ">", "&gt;")
	value = strings.ReplaceAll(value, `"`, "&quot;")
	value = strings.ReplaceAll(value, `'`, "&#39;")
	return value
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
