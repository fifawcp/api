package middlewares

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/mssola/useragent"
)

// TODO: think in a better naming
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

func RequestInfoMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			ipAddress := getClientIP(r)
			userAgentStr := r.UserAgent()
			deviceInfo := parseUserAgent(userAgentStr)

			requestInfo := RequestInfo{
				IPAddress:  ipAddress,
				UserAgent:  userAgentStr,
				DeviceInfo: deviceInfo,
			}

			ctx = context.WithValue(ctx, RequestInfoContextKey, &requestInfo)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func getClientIP(r *http.Request) string {
	// Chi's middleware.RealIP has already processed X-Forwarded-For and X-Real-IP headers
	// and set r.RemoteAddr to the real client IP. We just need to strip the port.
	ip := r.RemoteAddr

	// Handle IPv6 addresses like [::1]:63514 or [2001:db8::1]:8080
	if len(ip) > 0 && ip[0] == '[' {
		if idx := strings.Index(ip, "]"); idx != -1 {
			return ip[1:idx]
		}
	}

	// Handle IPv4 like 127.0.0.1:63514
	if host, _, found := strings.Cut(ip, ":"); found {
		return host
	}

	return ip
}

func parseUserAgent(userAgent string) DeviceInfo {
	ua := useragent.New(userAgent)

	browser, _ := ua.Browser()
	platform := ua.Platform()
	os := ua.OS()
	model := ua.Model()
	displayName := generateDisplayName(browser, platform, model)

	return DeviceInfo{
		Browser:     browser,
		Platform:    platform,
		DeviceModel: model,
		DisplayName: displayName,
		OS:          os,
	}
}

func generateDisplayName(browser, platform, model string) string {
	// If we have a specific device model, use it
	if model != "" {
		if browser != "" {
			return fmt.Sprintf("%s on %s", browser, model)
		}
		return model
	}

	// If we have both browser and platform
	if browser != "" && platform != "" {
		return fmt.Sprintf("%s on %s", browser, platform)
	}

	// If we only have browser
	if browser != "" {
		return browser
	}

	// If we only have platform
	if platform != "" {
		return platform
	}

	return "Unknown Device"
}
