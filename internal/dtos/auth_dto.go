package dtos

import (
	"net/mail"
	"strings"
	"time"

	"github.com/fifawcp/api/internal/domain"
)

type RequestOtpDto struct {
	Identifier string             `json:"identifier" validate:"required,max=255" example:"john.doe@example.com"`
	Purpose    *domain.OTPPurpose `json:"purpose" validate:"required,oneof=registration login" example:"registration" swaggertype:"string" enums:"registration,login"`
}

func (dto *RequestOtpDto) Normalize() {
	dto.Identifier = strings.TrimSpace(dto.Identifier)

	if isEmail(dto.Identifier) {
		dto.Identifier = strings.ToLower(dto.Identifier)
	}
}

type AuthenticationInputDto struct {
	Identifier string            `json:"identifier" validate:"required,max=255" example:"john.doe@example.com"`
	Purpose    domain.OTPPurpose `json:"purpose" validate:"required,oneof=registration login" example:"registration" swaggertype:"string" enums:"registration,login"`
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
	AccessToken  string    `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIzZjAxZmFmNiJ9.signature"`
	RefreshToken string    `json:"-"` // Set in cookie, not returned in API response
	ExpiresAt    time.Time `json:"expires_at" example:"2026-04-20T02:18:46Z"`
}

type RequestInfo struct {
	IPAddress  string
	UserAgent  string
	DeviceInfo DeviceInfo
}

type DeviceInfo struct {
	Browser     string `json:"browser"`
	Platform    string `json:"platform"`
	DeviceModel string `json:"device_model,omitempty"`
	DisplayName string `json:"display_name"`
	OS          string `json:"os"`
}

func isEmail(s string) bool {
	_, err := mail.ParseAddress(s)
	return err == nil
}
