package api

import (
	"net"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/time/rate"
)

type APIKeyConfig struct {
	Key        string
	AllowedIPs []string
	RateLimit  int
}

// APIKeyMiddleware obsługuje autoryzację kluczami API
type APIKeyMiddleware struct {
	keys       map[string]APIKeyConfig
	limiters   map[string]*rate.Limiter
	mu         sync.RWMutex
	allowedIPs map[string][]net.IPNet
}

// NewAPIKeyMiddleware tworzy nową instancję middleware dla kluczy API
func NewAPIKeyMiddleware(keys map[string]APIKeyConfig) (*APIKeyMiddleware, error) {
	m := &APIKeyMiddleware{
		keys:       keys,
		limiters:   make(map[string]*rate.Limiter),
		allowedIPs: make(map[string][]net.IPNet),
	}

	// Parsuj dozwolone adresy IP
	for key, config := range keys {
		var nets []net.IPNet
		for _, ipStr := range config.AllowedIPs {
			_, ipNet, err := net.ParseCIDR(ipStr)
			if err != nil {
				// Jeśli to nie CIDR, spróbuj jako pojedynczy adres IP
				ip := net.ParseIP(ipStr)
				if ip == nil {
					return nil, err
				}
				var mask net.IPMask
				if strings.Contains(ipStr, ":") {
					mask = net.CIDRMask(128, 128) // IPv6
				} else {
					mask = net.CIDRMask(32, 32) // IPv4
				}
				ipNet = &net.IPNet{IP: ip, Mask: mask}
			}
			nets = append(nets, *ipNet)
		}
		m.allowedIPs[key] = nets
	}

	return m, nil
}

// Middleware implementuje http.Handler dla autoryzacji kluczami API
func (m *APIKeyMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-API-Key")
		if key == "" {
			http.Error(w, "Missing API key", http.StatusUnauthorized)
			return
		}

		// Sprawdź czy klucz istnieje
		config, exists := m.keys[key]
		if !exists {
			http.Error(w, "Invalid API key", http.StatusUnauthorized)
			return
		}

		// Sprawdź adres IP
		if len(config.AllowedIPs) > 0 {
			clientIP := net.ParseIP(strings.Split(r.RemoteAddr, ":")[0])
			if clientIP == nil {
				http.Error(w, "Invalid client IP", http.StatusBadRequest)
				return
			}

			allowed := false
			for _, ipNet := range m.allowedIPs[key] {
				if ipNet.Contains(clientIP) {
					allowed = true
					break
				}
			}
			if !allowed {
				http.Error(w, "IP not allowed", http.StatusForbidden)
				return
			}
		}

		// Sprawdź rate limit
		m.mu.Lock()
		limiter, exists := m.limiters[key]
		if !exists {
			limiter = rate.NewLimiter(rate.Limit(config.RateLimit)/60, config.RateLimit)
			m.limiters[key] = limiter
		}
		m.mu.Unlock()

		if !limiter.Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
