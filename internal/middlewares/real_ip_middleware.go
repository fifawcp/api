package middlewares

import (
	"net"
	"net/http"
	"strings"
)

func TrustedProxyRealIP(trustedCIDRs []string) func(http.Handler) http.Handler {
	nets := parseCIDRs(trustedCIDRs)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.RemoteAddr = resolveClientIP(r, nets)
			next.ServeHTTP(w, r)
		})
	}
}

func resolveClientIP(r *http.Request, trusted []*net.IPNet) string {
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

// forwardedChain returns the valid IPs from all X-Forwarded-For headers, left to right.
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

// stripPort returns the host portion of an address, tolerating bare IPs (v4 and v6),
// "host:port", and "[ipv6]:port".
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
