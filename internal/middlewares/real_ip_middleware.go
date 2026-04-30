package middlewares

import (
	"net"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
)

func TrustedProxyRealIP(trustedCIDRs []string) func(http.Handler) http.Handler {
	nets := parseCIDRs(trustedCIDRs)

	return func(next http.Handler) http.Handler {
		realIP := middleware.RealIP(next)

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(nets) == 0 || isFromTrustedProxy(r.RemoteAddr, nets) {
				// Connection is from a known proxy — trust the forwarded headers.
				realIP.ServeHTTP(w, r)
			} else {
				// Unknown source — ignore forwarded headers, use the raw TCP IP.
				next.ServeHTTP(w, r)
			}
		})
	}
}

func parseCIDRs(cidrs []string) []*net.IPNet {
	var nets []*net.IPNet
	for _, cidr := range cidrs {
		cidr = strings.TrimSpace(cidr)
		if cidr == "" {
			continue
		}

		// Accept plain IPs without a mask by appending /32 or /128.
		if !strings.Contains(cidr, "/") {
			if strings.Contains(cidr, ":") {
				cidr += "/128"
			} else {
				cidr += "/32"
			}
		}

		if _, network, err := net.ParseCIDR(cidr); err == nil {
			nets = append(nets, network)
		}
	}

	return nets
}

func isFromTrustedProxy(remoteAddr string, nets []*net.IPNet) bool {
	// Strip port - remoteAddr may be "1.2.3.4:port" or "[::1]:port".
	host := remoteAddr
	if h, _, err := net.SplitHostPort(remoteAddr); err == nil {
		host = h
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}

	for _, network := range nets {
		if network.Contains(ip) {
			return true
		}
	}

	return false
}
