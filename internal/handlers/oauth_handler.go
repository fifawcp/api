package handlers

import (
	"net/http"

	"github.com/fifawcp/api/internal/httpctx"
	"github.com/fifawcp/api/internal/httpx"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/fifawcp/api/internal/services"
)

type OAuthHandler struct {
	logger       logging.Logger
	oauthService services.OAuthServiceInterface
	cfg          *config.Config
}

func NewOAuthHandler(
	oauthService services.OAuthServiceInterface,
	logger logging.Logger,
	cfg *config.Config,
) *OAuthHandler {
	return &OAuthHandler{
		oauthService: oauthService,
		logger:       logger,
		cfg:          cfg,
	}
}

// GoogleOAuth godoc
//
//	@Summary		Start Google OAuth login
//	@Description	Middleware validates `return_to` (allowlisted scheme://host). The handler stores `return_to` in Redis keyed by a random `state`, then responds with HTTP 307 to Google's authorization URL. After consent, Google sends the browser to `/api/oauth/google/callback` with `state` and `code`. The refresh cookie is set on that callback response, not here.
//	@Tags			oauth
//	@Produce		json
//	@Param			return_to	query	string	true	"Absolute URL to redirect after successful callback (allowlisted origin)"
//	@Success		307			"Redirect to Google (Temporary Redirect)"
//	@Failure		400			{object}	httpx.ErrorResponse	"Missing, invalid, or non-allowlisted return_to (middleware)"
//	@Failure		500			{object}	httpx.ErrorResponse	"Internal server error"
//	@Router			/oauth/google [get]
func (h *OAuthHandler) GoogleOAuth(w http.ResponseWriter, r *http.Request) {
	returnTo := httpctx.GetReturnTo(r.Context())

	url, err := h.oauthService.BeginGoogleLogin(r.Context(), returnTo)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// GoogleOAuthCallback godoc
//
//	@Summary		Google OAuth callback
//	@Description	Middleware rejects Google `error` query, missing `state`, or missing `code`. The handler exchanges `code` for tokens, verifies Google's `id_token`, resolves or creates the user, calls AuthService for session + refresh + access token, sets the HttpOnly refresh cookie, and redirects (302) to `return_to` loaded from Redis (not from the callback URL). On failure, JSON is returned instead of a redirect.
//	@Description	Domain errors are mapped in handleServiceError: unknown/expired `state` → 400; unverified Google email → 403; missing `id_token` after exchange → 502; other failures → 500.
//	@Tags			oauth
//	@Produce		json
//	@Param			state	query	string	true	"OAuth state from the login start redirect"
//	@Param			code	query	string	true	"Authorization code from Google"
//	@Success		302		"Refresh cookie set; redirect to return_to"
//	@Failure		400		{object}	httpx.ErrorResponse	"Invalid callback (middleware) or unknown/expired OAuth state"
//	@Failure		403		{object}	httpx.ErrorResponse	"Google email not verified"
//	@Failure		502		{object}	httpx.ErrorResponse	"Token exchange missing identity token"
//	@Failure		500		{object}	httpx.ErrorResponse	"Internal server error"
//	@Router			/oauth/google/callback [get]
func (h *OAuthHandler) GoogleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	state := httpctx.GetOAuthState(r.Context())
	code := httpctx.GetOAuthCode(r.Context())
	requestInfo := httpctx.GetRequestInfo(r.Context())

	authentication, returnTo, err := h.oauthService.CompleteGoogleLogin(r.Context(), state, code, *requestInfo)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	httpx.SetRefreshTokenCookie(
		w,
		authentication.Auth.RefreshToken,
		authentication.Auth.ExpiresAt,
		h.cfg.IsProd(),
	)
	http.Redirect(w, r, returnTo, http.StatusFound)
}
