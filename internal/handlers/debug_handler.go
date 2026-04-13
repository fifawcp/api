package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ncondes/fifawcp/internal/infrastructure/config"
	"github.com/ncondes/fifawcp/internal/packages/httputils"
	"github.com/ncondes/fifawcp/internal/packages/totp"
)

type DebugHandler struct {
	cfg *config.Config
}

func NewDebugHandler(cfg *config.Config) *DebugHandler {
	return &DebugHandler{
		cfg: cfg,
	}
}

// GetOtp godoc
//
//	@Summary		Get non-production OTP
//	@Description	Returns the current TOTP-style bypass OTP for a given identifier.
//	@Description	**This endpoint is NOT registered in production and will return 404.**
//	@Description	Intended for non-production environments only.
//	@Tags			debug
//	@Produce		json
//	@Param			identifier	path		string					true	"User identifier"
//	@Success		200			{object}	httputils.Response{}	"OTP and seconds until rotation"
//	@Failure		400			{object}	httputils.ErrorResponse	"Missing identifier"
//	@Router			/debug/auth/otp/request/{identifier} [get]
func (h *DebugHandler) GetOtp(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "identifier")
	if identifier == "" {
		httputils.RespondWithError(w, http.StatusBadRequest, errors.New("identifier is required"))
		return
	}

	httputils.RespondWithData(w, http.StatusOK, map[string]any{
		"otp":       totp.Generate(identifier, h.cfg.JWT.Secret),
		"expiresIn": totp.WindowExpiresIn(),
	})
}
