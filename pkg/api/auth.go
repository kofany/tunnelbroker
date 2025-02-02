package api

import (
	"log"
	"os"

	"github.com/kofany/tunnelbroker/internal/auth"
)

// NewAuthenticator tworzy nową instancję autentykacji
func NewAuthenticator() (*auth.Authenticator, error) {
	// Sprawdź czy wszystkie wymagane zmienne środowiskowe są ustawione
	if os.Getenv("OIDC_ISSUER_URL") == "" ||
		os.Getenv("OIDC_CLIENT_ID") == "" ||
		os.Getenv("OIDC_CLIENT_SECRET") == "" ||
		os.Getenv("OIDC_REDIRECT_URL") == "" {
		log.Println("Warning: Missing OIDC configuration")
		return nil, nil
	}

	authConfig := auth.Config{
		IssuerURL:     os.Getenv("OIDC_ISSUER_URL"),
		ClientID:      os.Getenv("OIDC_CLIENT_ID"),
		ClientSecret:  os.Getenv("OIDC_CLIENT_SECRET"),
		RedirectURL:   os.Getenv("OIDC_REDIRECT_URL"),
		AllowedGroups: []string{"admin", "tunnelbroker-users"},
	}

	return auth.NewAuthenticator(authConfig)
}
