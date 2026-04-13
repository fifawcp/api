package dtos

import (
	"net/mail"
	"strings"
	"time"

	"github.com/ncondes/fifa-world-cup-pickems/internal/domain"
)

type RequestOtpDto struct {
	Identifier string             `json:"identifier" validate:"required,max=255" example:"user@example.com"`
	Purpose    *domain.OTPPurpose `json:"purpose" validate:"required,oneof=registration login" example:"registration"`
}

func (dto *RequestOtpDto) Normalize() {
	dto.Identifier = strings.TrimSpace(dto.Identifier)

	if isEmail(dto.Identifier) {
		dto.Identifier = strings.ToLower(dto.Identifier)
	}
}

type AuthenticationInputDto struct {
	Identifier string            `json:"identifier" validate:"required,max=255" example:"user@example.com"`
	Purpose    domain.OTPPurpose `json:"purpose" validate:"required,oneof=registration login" example:"registration"`
	OTP        string            `json:"otp" validate:"required,min=6,max=6" example:"123456"`
	User       *CreateUserDto    `json:"user"`
}

func (dto *AuthenticationInputDto) Normalize() {
	dto.Identifier = strings.TrimSpace(dto.Identifier)

	if isEmail(dto.Identifier) {
		dto.Identifier = strings.ToLower(dto.Identifier)
	}
}

type AuthenticationDto struct {
	User *domain.User `json:"user"`
	Auth AuthData     `json:"auth"`
}

type AuthData struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type RefreshTokenDto struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

func isEmail(s string) bool {
	_, err := mail.ParseAddress(s)
	return err == nil
}
