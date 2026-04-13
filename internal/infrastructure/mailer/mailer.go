package mailer

import (
	"context"

	"github.com/ncondes/fifawcp/internal/domain"
)

type Mailer interface {
	SendOTPEmail(ctx context.Context, to, otp string, purpose domain.OTPPurpose) error
	SendWelcomeEmail(ctx context.Context, to, firstName string) error
}
