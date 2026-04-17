package mailer

import (
	"bytes"
	"context"
	"embed"
	"html/template"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/resend/resend-go/v3"
)

type ResendMailer struct {
	client *resend.Client
	cfg    *config.Config
}

// Use Go compiler directive that bundles the templates into the binary
// This ensures the templates are always available and can be deployed as a single binary

//
//go:embed templates/*.html
var templateFS embed.FS

func NewResendMailer(cfg *config.Config) *ResendMailer {
	return &ResendMailer{
		client: resend.NewClient(cfg.Mailer.APIKey),
		cfg:    cfg,
	}
}

func (m *ResendMailer) SendOTPEmail(
	ctx context.Context,
	to, otp string,
	purpose domain.OTPPurpose,
) error {
	if !m.cfg.IsProd() {
		return nil
	}

	subject := "⚽ Your FIFA World Cup Pickems login code"
	if purpose == domain.OTPPurposeRegistration {
		subject = "⚽ Confirm your email — You're almost in!"
	}

	html, err := m.renderTemplate("templates/otp.html", map[string]string{
		"OTP":     otp,
		"Purpose": string(purpose),
	})
	if err != nil {
		return err
	}

	_, err = m.client.Emails.Send(&resend.SendEmailRequest{
		From:    m.cfg.Mailer.FromAddress,
		To:      []string{to},
		Subject: subject,
		Html:    html,
	})

	return err
}

func (m *ResendMailer) SendWelcomeEmail(
	ctx context.Context,
	to, firstName string,
) error {
	if !m.cfg.IsProd() {
		return nil
	}

	html, err := m.renderTemplate("templates/welcome.html", map[string]string{
		"FirstName": firstName,
	})
	if err != nil {
		return err
	}

	_, err = m.client.Emails.Send(&resend.SendEmailRequest{
		From:    m.cfg.Mailer.FromAddress,
		To:      []string{to},
		Subject: "Welcome to FIFA World Cup Pickems",
		Html:    html,
	})

	return err
}

func (m *ResendMailer) renderTemplate(name string, data any) (string, error) {
	// Parse the template from the embedded filesystem
	tmpl, err := template.ParseFS(templateFS, name)
	if err != nil {
		return "", err
	}

	// Execute the template with the provided data
	var buf bytes.Buffer

	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
