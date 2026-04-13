package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ncondes/fifa-world-cup-pickems/internal/dtos"
	"github.com/ncondes/fifa-world-cup-pickems/internal/infrastructure/logging"
	"github.com/ncondes/fifa-world-cup-pickems/internal/infrastructure/middlewares"
	"github.com/ncondes/fifa-world-cup-pickems/internal/infrastructure/validator"
	"github.com/ncondes/fifa-world-cup-pickems/internal/services"
)

type AuthHandler struct {
	authService services.AuthServiceInterface
	logger      logging.Logger
	validator   *validator.Validator
}

func NewAuthHandler(
	authService services.AuthServiceInterface,
	logger logging.Logger,
	v *validator.Validator,
) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		logger:      logger,
		validator:   v,
	}
}

func (h *AuthHandler) RequestOtp(w http.ResponseWriter, r *http.Request) {
	var payload dtos.RequestOtpDto

	if err := readAndValidateJSON(w, r, &payload, h.validator); err != nil {
		return
	}

	if err := h.authService.RequestOtp(r.Context(), &payload); err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	respondWithData(w, http.StatusNoContent, nil)
}

func (h *AuthHandler) Authenticate(w http.ResponseWriter, r *http.Request) {
	var body dtos.AuthenticationInputDto

	if err := readAndValidateJSON(w, r, &body, h.validator); err != nil {
		return
	}

	requestInfo := middlewares.GetRequestInfo(r.Context())

	if requestInfo == nil {
		respondWithError(w, http.StatusInternalServerError, errInternalServerError)
		return
	}

	authenticationResponse, err := h.authService.Authenticate(r.Context(), &body, requestInfo)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	respondWithData(w, http.StatusOK, authenticationResponse)
}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var body dtos.RefreshTokenDto

	if err := readAndValidateJSON(w, r, &body, h.validator); err != nil {
		return
	}

	authResponse, err := h.authService.RefreshToken(r.Context(), &body)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	respondWithData(w, http.StatusOK, authResponse)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var body dtos.RefreshTokenDto

	if err := readAndValidateJSON(w, r, &body, h.validator); err != nil {
		return
	}

	if err := h.authService.Logout(r.Context(), body.RefreshToken); err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	respondWithData(w, http.StatusNoContent, nil)
}

func (h *AuthHandler) LogoutAll(w http.ResponseWriter, r *http.Request) {
	var body dtos.RefreshTokenDto

	if err := readAndValidateJSON(w, r, &body, h.validator); err != nil {
		return
	}

	if err := h.authService.LogoutAll(r.Context(), body.RefreshToken); err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	respondWithData(w, http.StatusNoContent, nil)
}

func (h *AuthHandler) GetSessions(w http.ResponseWriter, r *http.Request) {
	var body dtos.RefreshTokenDto

	if err := readAndValidateJSON(w, r, &body, h.validator); err != nil {
		return
	}

	sessions, err := h.authService.GetSessions(r.Context(), body.RefreshToken)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	respondWithData(w, http.StatusOK, sessions)
}

func (h *AuthHandler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")

	if err := h.authService.DeleteSession(r.Context(), sessionID); err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	respondWithData(w, http.StatusNoContent, nil)
}
