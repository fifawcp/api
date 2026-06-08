package middlewares_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fifawcp/api/internal/httpctx"
	"github.com/fifawcp/api/internal/middlewares"
)

// Runs the production middleware chain (TrustedProxyRealIP -> RequestInfo) and returns the
// IP that handlers ultimately observe via the request context.
func resolveIP(trustedCIDRs []string, remoteAddr string, headers map[string]string) string {
	var observed string
	terminal := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		if info := httpctx.GetRequestInfo(r.Context()); info != nil {
			observed = info.IPAddress
		}
	})

	chain := middlewares.TrustedProxyRealIP(trustedCIDRs)(middlewares.RequestInfo()(terminal))

	req := httptest.NewRequest(http.MethodPost, "/api/auth/token", nil)
	req.RemoteAddr = remoteAddr
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	chain.ServeHTTP(httptest.NewRecorder(), req)
	return observed
}

func TestResolveClientIP(t *testing.T) {
	const (
		realClient = "203.0.113.45"   // the actual end user
		bffEgress  = "44.220.117.213" // web BFF egress — the constant IP that used to leak into the DB
	)

	// Mirrors production: the API trusts its own Railway edge plus the BFF's egress range.
	prodTrusted := []string{"10.0.0.0/8", "44.220.117.0/24"}

	tests := []struct {
		name         string
		trustedCIDRs []string
		remoteAddr   string
		headers      map[string]string
		want         string
	}{
		{
			name:         "real client recovered from forwarded chain behind the BFF",
			trustedCIDRs: prodTrusted,
			remoteAddr:   "10.0.0.1:5000", // Railway edge
			headers:      map[string]string{"X-Forwarded-For": realClient + ", " + bffEgress},
			want:         realClient,
		},
		{
			name:         "X-Real-IP set to the BFF egress is ignored",
			trustedCIDRs: prodTrusted,
			remoteAddr:   "10.0.0.1:5000",
			headers: map[string]string{
				"X-Real-IP":       bffEgress,
				"X-Forwarded-For": realClient + ", " + bffEgress,
			},
			want: realClient,
		},
		{
			name:         "spoofed leftmost entry is not selected",
			trustedCIDRs: prodTrusted,
			remoteAddr:   "10.0.0.1:5000",
			headers:      map[string]string{"X-Forwarded-For": "6.6.6.6, " + realClient + ", " + bffEgress},
			want:         realClient,
		},
		{
			name:         "untrusted peer falls back to the raw connection IP",
			trustedCIDRs: prodTrusted,
			remoteAddr:   "8.8.8.8:1234",
			headers:      map[string]string{"X-Forwarded-For": "1.2.3.4"},
			want:         "8.8.8.8",
		},
		{
			name:         "no trust list takes the leftmost forwarded address",
			trustedCIDRs: nil,
			remoteAddr:   "10.0.0.1:5000",
			headers:      map[string]string{"X-Forwarded-For": realClient + ", " + bffEgress},
			want:         realClient,
		},
		{
			name:         "no forwarded header falls back to the peer",
			trustedCIDRs: prodTrusted,
			remoteAddr:   "10.0.0.1:5000",
			want:         "10.0.0.1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveIP(tc.trustedCIDRs, tc.remoteAddr, tc.headers)
			if got != tc.want {
				t.Fatalf("resolved client IP = %q, want %q", got, tc.want)
			}
		})
	}
}
