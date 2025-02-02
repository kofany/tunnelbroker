package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// Config reprezentuje konfigurację autoryzacji
type Config struct {
	IssuerURL     string
	ClientID      string
	ClientSecret  string
	RedirectURL   string
	AllowedGroups []string
}

// Authenticator obsługuje autoryzację
type Authenticator struct {
	provider     *oidc.Provider
	oauth2Config oauth2.Config
	verifier     *oidc.IDTokenVerifier
	config       Config
}

// NewAuthenticator tworzy nowy autentykator
func NewAuthenticator(config Config) (*Authenticator, error) {
	provider, err := oidc.NewProvider(context.Background(), config.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %v", err)
	}

	oauth2Config := oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "groups"},
	}

	return &Authenticator{
		provider:     provider,
		oauth2Config: oauth2Config,
		verifier:     provider.Verifier(&oidc.Config{ClientID: config.ClientID}),
		config:       config,
	}, nil
}

// Middleware sprawdza token JWT i autoryzuje dostęp
func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "No authorization header", http.StatusUnauthorized)
			return
		}

		bearerToken := strings.TrimPrefix(authHeader, "Bearer ")
		if bearerToken == authHeader {
			http.Error(w, "Invalid authorization header", http.StatusUnauthorized)
			return
		}

		token, err := a.verifier.Verify(r.Context(), bearerToken)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		var claims struct {
			Groups []string `json:"groups"`
			Email  string   `json:"email"`
		}
		if err := token.Claims(&claims); err != nil {
			http.Error(w, "Failed to parse claims", http.StatusInternalServerError)
			return
		}

		// Sprawdź uprawnienia grupy
		if !a.hasAllowedGroup(claims.Groups) {
			http.Error(w, "Insufficient permissions", http.StatusForbidden)
			return
		}

		// Dodaj informacje o użytkowniku do kontekstu
		ctx := context.WithValue(r.Context(), "user_email", claims.Email)
		ctx = context.WithValue(ctx, "user_groups", claims.Groups)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// hasAllowedGroup sprawdza czy użytkownik należy do dozwolonej grupy
func (a *Authenticator) hasAllowedGroup(userGroups []string) bool {
	for _, allowedGroup := range a.config.AllowedGroups {
		for _, userGroup := range userGroups {
			if allowedGroup == userGroup {
				return true
			}
		}
	}
	return false
}

// LoginHandler obsługuje proces logowania
func (a *Authenticator) LoginHandler(w http.ResponseWriter, r *http.Request) {
	state := generateRandomState()
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_state",
		Value:    state,
		MaxAge:   int(time.Hour.Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, a.oauth2Config.AuthCodeURL(state), http.StatusTemporaryRedirect)
}

// CallbackHandler obsługuje callback po autoryzacji
func (a *Authenticator) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	state, err := r.Cookie("auth_state")
	if err != nil {
		http.Error(w, "State cookie not found", http.StatusBadRequest)
		return
	}
	if r.URL.Query().Get("state") != state.Value {
		http.Error(w, "State did not match", http.StatusBadRequest)
		return
	}

	oauth2Token, err := a.oauth2Config.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
		return
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "No ID token found", http.StatusInternalServerError)
		return
	}

	// Weryfikacja tokenu
	idToken, err := a.verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		http.Error(w, "Failed to verify ID token", http.StatusInternalServerError)
		return
	}

	var claims struct {
		Email  string   `json:"email"`
		Groups []string `json:"groups"`
	}
	if err := idToken.Claims(&claims); err != nil {
		http.Error(w, "Failed to parse claims", http.StatusInternalServerError)
		return
	}

	// Sprawdź uprawnienia
	if !a.hasAllowedGroup(claims.Groups) {
		http.Error(w, "Insufficient permissions", http.StatusForbidden)
		return
	}

	// Zwróć token dostępu
	response := struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}{
		AccessToken: oauth2Token.AccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int(oauth2Token.Expiry.Sub(time.Now()).Seconds()),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// generateRandomState generuje losowy stan dla OAuth2
func generateRandomState() string {
	// W rzeczywistej implementacji należy użyć bezpiecznego generatora
	return fmt.Sprintf("%d", time.Now().UnixNano())
} 