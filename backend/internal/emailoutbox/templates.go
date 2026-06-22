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
	case TypeTeamInvite:
		return renderTeamInvite(email)
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

func renderTeamInvite(email Email) (mailer.Message, error) {
	inviteURL, ok := stringField(email.TemplateData, "invite_url")
	if !ok {
		inviteURL, ok = stringField(email.TemplateData, "invite_url_path")
	}
	if !ok {
		return mailer.Message{}, fmt.Errorf("%w: invite_url is required", ErrTemplateInvalid)
	}
	workspaceName, _ := stringField(email.TemplateData, "workspace_name")
	if workspaceName == "" {
		workspaceName = "Kelmio"
	}
	inviterDisplayName, _ := stringField(email.TemplateData, "inviter_display_name")
	if inviterDisplayName == "" {
		inviterDisplayName = "An administrator"
	}
	role, _ := stringField(email.TemplateData, "role")
	if role == "" {
		role = "member"
	}

	subject := fmt.Sprintf("You're invited to %s", workspaceName)
	textBody := fmt.Sprintf("%s invited you to join %s as %s.\n\nAccept your invite:\n%s\n\nIf you did not expect this invite, you can ignore this email.", inviterDisplayName, workspaceName, role, inviteURL)
	htmlBody := fmt.Sprintf("<p>%s invited you to join <strong>%s</strong> as %s.</p><p><a href=\"%s\">Accept invite</a></p><p>If you did not expect this invite, you can ignore this email.</p>", htmlEscape(inviterDisplayName), htmlEscape(workspaceName), htmlEscape(role), htmlEscape(inviteURL))
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
	textBody := fmt.Sprintf("Hi %s,\n\nUse this link to reset your Kelmio password:\n%s\n\nIf you did not request this reset, you can ignore this email.", displayName, resetURL)
	htmlBody := fmt.Sprintf("<p>Hi %s,</p><p>Use this link to reset your Kelmio password:</p><p><a href=\"%s\">Reset password</a></p><p>If you did not request this reset, you can ignore this email.</p>", htmlEscape(displayName), htmlEscape(resetURL))
	return mailer.Message{
		To:       []string{email.RecipientEmail},
		Subject:  "Reset your Kelmio password",
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
