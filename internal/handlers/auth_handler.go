package handlers

import (
	"net/http"

	"github.com/fifawcp/api/internal/dtos"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/fifawcp/api/internal/infrastructure/middlewares"
	"github.com/fifawcp/api/internal/infrastructure/validator"
	"github.com/fifawcp/api/internal/packages/httputils"
	"github.com/fifawcp/api/internal/services"
	"github.com/go-chi/chi/v5"
)

type AuthHandler struct {
	authService services.AuthServiceInterface
	logger      logging.Logger
	validator   *validator.Validator
	cfg         *config.Config
}

func NewAuthHandler(
	authService services.AuthServiceInterface,
	logger logging.Logger,
	v *validator.Validator,
	cfg *config.Config,
) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		logger:      logger,
		validator:   v,
		cfg:         cfg,
	}
}

// RequestOtp godoc
//
//	@Summary		Request OTP
//	@Description	Sends a 6-digit OTP to the provided email address.
//	@Description	A cooldown is enforced between requests for the same identifier.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body	dtos.RequestOtpDto	true	"OTP request payload"
//	@Success		204
//	@Failure		400	{object}	httputils.ErrorResponse	"Invalid request body or validation error"
//	@Failure		401	{object}	httputils.ErrorResponse	"Invalid credentials"
//	@Failure		409	{object}	httputils.ErrorResponse	"User already exists"
//	@Failure		429	{object}	httputils.ErrorResponse	"Too many attempts or cooldown active"
//	@Failure		500	{object}	httputils.ErrorResponse	"Internal server error"
//	@Router			/auth/otp/request [post]
func (h *AuthHandler) RequestOtp(w http.ResponseWriter, r *http.Request) {
	var payload dtos.RequestOtpDto

	if err := httputils.ReadAndValidateJSON(w, r, &payload, h.validator); err != nil {
		return
	}

	if err := h.authService.RequestOtp(r.Context(), &payload); err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httputils.RespondWithData(w, http.StatusNoContent, nil)
}

// VerifyOtp godoc
//
//	@Summary		Verify OTP
//	@Description	Validates a 6-digit OTP for a given identifier and purpose (registration or login).
//	@Description	Does NOT create a session or return tokens — only verifies the OTP.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		dtos.VerifyOtpDto	true	"Verify OTP payload"
//	@Success		204
//	@Failure		400	{object}	httputils.ErrorResponse	"Invalid request body or validation error"
//	@Failure		401	{object}	httputils.ErrorResponse	"OTP invalid or expired"
//	@Failure		429	{object}	httputils.ErrorResponse	"Too many attempts"
//	@Failure		500	{object}	httputils.ErrorResponse	"Internal server error"
//	@Router			/auth/otp/verify [post]
func (h *AuthHandler) VerifyOtp(w http.ResponseWriter, r *http.Request) {
	var payload dtos.VerifyOtpDto

	if err := httputils.ReadAndValidateJSON(w, r, &payload, h.validator); err != nil {
		return
	}

	if err := h.authService.VerifyOTP(r.Context(), &payload); err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httputils.RespondWithData(w, http.StatusNoContent, nil)
}

// Authenticate godoc
//
//	@Summary		Exchange OTP for tokens
//	@Description	Verifies the OTP and issues a new access token + session.
//	@Description	The refresh token is NOT returned in the response body — it is set as an HttpOnly cookie (`refresh_token`, path `/api/auth`).
//	@Description	- For `registration`: include the `user` field in the request body.
//	@Description	- For `login`: omit the `user` field.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		dtos.AuthenticationInputDto						true	"Authentication payload"
//	@Success		200		{object}	httputils.Response{data=dtos.AuthenticationDto}	"Access token and user info. Refresh token set as HttpOnly cookie."
//	@Failure		400		{object}	httputils.ErrorResponse							"Invalid request body or validation error"
//	@Failure		401		{object}	httputils.ErrorResponse							"OTP invalid or expired, or invalid credentials"
//	@Failure		409		{object}	httputils.ErrorResponse							"User already exists or username already taken"
//	@Failure		429		{object}	httputils.ErrorResponse							"Too many OTP attempts or cooldown active"
//	@Failure		500		{object}	httputils.ErrorResponse							"Internal server error"
//	@Router			/auth/token [post]
func (h *AuthHandler) Authenticate(w http.ResponseWriter, r *http.Request) {
	var body dtos.AuthenticationInputDto

	if err := httputils.ReadAndValidateJSON(w, r, &body, h.validator); err != nil {
		return
	}

	requestInfo := middlewares.GetRequestInfo(r.Context())

	authenticationResponse, err := h.authService.Authenticate(r.Context(), &body, *requestInfo)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httputils.SetRefreshTokenCookie(
		w,
		authenticationResponse.Auth.RefreshToken,
		authenticationResponse.Auth.ExpiresAt,
		h.cfg.IsProd(),
	)
	httputils.RespondWithData(w, http.StatusOK, authenticationResponse)
}

// RefreshToken godoc
//
//	@Summary		Rotate refresh token
//	@Description	Reads the `refresh_token` HttpOnly cookie, validates it, atomically replaces it with a new one, and returns a new access token.
//	@Description	The old token is invalidated immediately — replaying it results in a 401.
//	@Description	The new refresh token is set as an HttpOnly cookie. Must be called with `credentials: 'include'` from the frontend.
//	@Tags			auth
//	@Produce		json
//	@Success		200	{object}	httputils.Response{data=dtos.AuthData}	"New access token. New refresh token set as HttpOnly cookie."
//	@Failure		401	{object}	httputils.ErrorResponse					"Missing cookie, or refresh token invalid or expired"
//	@Failure		500	{object}	httputils.ErrorResponse					"Internal server error"
//	@Router			/auth/token/refresh [post]
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := httputils.GetRefreshTokenFromCookie(r)
	if err != nil {
		httputils.RespondWithError(w, http.StatusUnauthorized, err)
		return
	}

	authResponse, err := h.authService.RefreshToken(r.Context(), refreshToken)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httputils.SetRefreshTokenCookie(
		w,
		authResponse.RefreshToken,
		authResponse.ExpiresAt,
		h.cfg.IsProd(),
	)
	httputils.RespondWithData(w, http.StatusOK, authResponse)
}

// Logout godoc
//
//	@Summary		Logout current device
//	@Description	Reads the `refresh_token` cookie, deletes the associated session and its refresh token, then clears the cookie.
//	@Description	Only the session tied to this specific token is affected.
//	@Tags			auth
//	@Produce		json
//	@Success		204
//	@Failure		401	{object}	httputils.ErrorResponse	"Missing cookie, or refresh token invalid or expired"
//	@Failure		500	{object}	httputils.ErrorResponse	"Internal server error"
//	@Router			/auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := httputils.GetRefreshTokenFromCookie(r)
	if err != nil {
		httputils.RespondWithError(w, http.StatusUnauthorized, err)
		return
	}

	if err := h.authService.Logout(r.Context(), refreshToken); err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httputils.ClearRefreshTokenCookie(w, h.cfg.IsProd())
	httputils.RespondWithData(w, http.StatusNoContent, nil)
}

// LogoutAll godoc
//
//	@Summary		Logout all devices
//	@Description	Reads the `refresh_token` cookie, identifies the user, deletes ALL of their sessions and refresh tokens, then clears the cookie.
//	@Description	Every device the user is logged into is logged out simultaneously.
//	@Tags			auth
//	@Produce		json
//	@Success		204
//	@Failure		401	{object}	httputils.ErrorResponse	"Missing cookie, or refresh token invalid or expired"
//	@Failure		500	{object}	httputils.ErrorResponse	"Internal server error"
//	@Router			/auth/logout/all [post]
func (h *AuthHandler) LogoutAll(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := httputils.GetRefreshTokenFromCookie(r)
	if err != nil {
		httputils.RespondWithError(w, http.StatusUnauthorized, err)
		return
	}

	if err := h.authService.LogoutAll(r.Context(), refreshToken); err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httputils.ClearRefreshTokenCookie(w, h.cfg.IsProd())
	httputils.RespondWithData(w, http.StatusNoContent, nil)
}

// GetSessions godoc
//
//	@Summary		List active sessions
//	@Description	Returns all active sessions for the user associated with the `refresh_token` cookie.
//	@Description	Each session includes device info, IP address, and usage timestamps.
//	@Tags			auth
//	@Produce		json
//	@Success		200	{object}	httputils.Response{data=[]domain.Session}	"List of active sessions"
//	@Failure		401	{object}	httputils.ErrorResponse						"Missing cookie, or refresh token invalid or expired"
//	@Failure		500	{object}	httputils.ErrorResponse						"Internal server error"
//	@Router			/auth/sessions [get]
func (h *AuthHandler) GetSessions(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := httputils.GetRefreshTokenFromCookie(r)
	if err != nil {
		httputils.RespondWithError(w, http.StatusUnauthorized, err)
		return
	}

	sessions, err := h.authService.GetSessions(r.Context(), refreshToken)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httputils.RespondWithData(w, http.StatusOK, sessions)
}

// DeleteSession godoc
//
//	@Summary		Delete a session
//	@Description	Deletes a specific session by ID. Ownership is enforced — the session must belong to the authenticated user.
//	@Description	Returns 404 if the session does not exist or belongs to a different user.
//	@Tags			auth
//	@Produce		json
//	@Param			id	path	string	true	"Session ID (UUID)"
//	@Success		204
//	@Failure		401	{object}	httputils.ErrorResponse	"Missing or invalid Bearer token"
//	@Failure		404	{object}	httputils.ErrorResponse	"Session not found or not owned by authenticated user"
//	@Failure		500	{object}	httputils.ErrorResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/auth/sessions/{id} [delete]
func (h *AuthHandler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	user := middlewares.GetAuthenticatedUser(r.Context())

	if err := h.authService.DeleteSession(
		r.Context(),
		sessionID,
		user.ID,
	); err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httputils.RespondWithData(w, http.StatusNoContent, nil)
}
