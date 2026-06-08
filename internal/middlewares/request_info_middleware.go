package middlewares

import (
	"context"
	"fmt"
	"net/http"

	"github.com/fifawcp/api/internal/dtos"
	"github.com/fifawcp/api/internal/httpctx"
	"github.com/mssola/useragent"
)

func RequestInfo() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			ipAddress := getClientIP(r)
			userAgentStr := r.UserAgent()
			deviceInfo := parseUserAgent(userAgentStr)

			requestInfo := dtos.RequestInfo{
				IPAddress:  ipAddress,
				UserAgent:  userAgentStr,
				DeviceInfo: deviceInfo,
			}

			ctx = context.WithValue(ctx, httpctx.RequestInfoContextKey, &requestInfo)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func getClientIP(r *http.Request) string {
	return stripPort(r.RemoteAddr)
}

func parseUserAgent(userAgent string) dtos.DeviceInfo {
	ua := useragent.New(userAgent)

	browser, _ := ua.Browser()
	platform := ua.Platform()
	os := ua.OS()
	model := ua.Model()
	displayName := generateDisplayName(browser, platform, model)

	return dtos.DeviceInfo{
		Browser:     browser,
		Platform:    platform,
		DeviceModel: model,
		DisplayName: displayName,
		OS:          os,
	}
}

func generateDisplayName(browser, platform, model string) string {
	if model != "" {
		if browser != "" {
			return fmt.Sprintf("%s on %s", browser, model)
		}
		return model
	}

	if browser != "" && platform != "" {
		return fmt.Sprintf("%s on %s", browser, platform)
	}

	if browser != "" {
		return browser
	}

	if platform != "" {
		return platform
	}

	return "Unknown Device"
}
