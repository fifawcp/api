package handlers

import (
	"net/http"

	"github.com/fifawcp/api/internal/httpx"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/fifawcp/api/internal/infrastructure/totp"
	"github.com/go-chi/chi/v5"
)

type DebugHandler struct {
	cfg *config.Config
}

func NewDebugHandler(cfg *config.Config) *DebugHandler {
	return &DebugHandler{
		cfg: cfg,
	}
}

// RequestTotp godoc
//
//	@Summary		Get non-production OTP
//	@Description	Returns the current TOTP-style bypass OTP for a given identifier.
//	@Description	**This endpoint is NOT registered in production and will return 404.**
//	@Description	Intended for non-production environments only.
//	@Tags			debug
//	@Produce		json
//	@Param			identifier	path		string				true	"User identifier"
//	@Success		200			{object}	httpx.Response{}	"OTP and seconds until rotation"
//	@Router			/debug/totp/{identifier} [get]
func (h *DebugHandler) RequestTotp(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "identifier")

	httpx.RespondWithData(w, http.StatusOK, map[string]any{
		"otp":       totp.Generate(identifier, h.cfg.JWT.Secret),
		"expiresIn": totp.WindowExpiresIn(),
	})
}
