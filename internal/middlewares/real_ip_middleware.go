package middlewares

import (
	"crypto/subtle"
	"net"
	"net/http"
	"strings"
)

const (
	clientIPHeader      = "X-Client-IP"
	forwardSecretHeader = "X-IP-Forward-Secret"
)

func TrustedProxyRealIP(trustedCIDRs []string, forwardSecret string) func(http.Handler) http.Handler {
	nets := parseCIDRs(trustedCIDRs)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.RemoteAddr = resolveClientIP(r, nets, forwardSecret)
			next.ServeHTTP(w, r)
		})
	}
}

func resolveClientIP(r *http.Request, trusted []*net.IPNet, forwardSecret string) string {
	if ip := clientIPFromTrustedHeader(r, forwardSecret); ip != "" {
		return ip
	}

	peer := stripPort(r.RemoteAddr)
	forwarded := forwardedChain(r)

	// No trust list: trust every proxy and take the original (leftmost) forwarded address.
	if len(trusted) == 0 {
		if len(forwarded) > 0 {
			return forwarded[0]
		}
		return peer
	}

	// The immediate peer must itself be a trusted proxy; otherwise the forwarded headers
	// are attacker-controlled and the raw connection address is the only safe value.
	if !ipInNets(peer, trusted) {
		return peer
	}

	// Walk right-to-left, skipping trusted proxies; the first untrusted address is the client.
	for i := len(forwarded) - 1; i >= 0; i-- {
		if !ipInNets(forwarded[i], trusted) {
			return forwarded[i]
		}
	}

	// Entire chain is trusted (or empty) — the closest trusted hop is the best we have.
	return peer
}

func clientIPFromTrustedHeader(r *http.Request, forwardSecret string) string {
	if forwardSecret == "" {
		return ""
	}

	provided := r.Header.Get(forwardSecretHeader)
	if subtle.ConstantTimeCompare([]byte(provided), []byte(forwardSecret)) != 1 {
		return ""
	}

	ip := strings.TrimSpace(r.Header.Get(clientIPHeader))
	if net.ParseIP(ip) == nil {
		return ""
	}

	return ip
}

func forwardedChain(r *http.Request) []string {
	var ips []string
	for _, header := range r.Header.Values("X-Forwarded-For") {
		for _, part := range strings.Split(header, ",") {
			if ip := strings.TrimSpace(part); net.ParseIP(ip) != nil {
				ips = append(ips, ip)
			}
		}
	}
	return ips
}

func ipInNets(ipStr string, nets []*net.IPNet) bool {
	ip := net.ParseIP(ipStr)
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

func stripPort(addr string) string {
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}
	return addr
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
