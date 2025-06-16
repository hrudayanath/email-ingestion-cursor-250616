package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/microsoft"

	"email-harvester/internal/config"
	"email-harvester/internal/models"
	"email-harvester/internal/store"
)

// OAuthService handles OAuth authentication for email providers
type OAuthService struct {
	config *config.OAuthConfig
	store  *store.MongoStore
	states map[string]string // In-memory state store for OAuth flow
}

// NewOAuthService creates a new OAuth service
func NewOAuthService(cfg *config.OAuthConfig) *OAuthService {
	return &OAuthService{
		config: cfg,
		states: make(map[string]string),
	}
}

// SetStore sets the store for the OAuth service
func (s *OAuthService) SetStore(store *store.MongoStore) {
	s.store = store
}

// GetAuthURL generates an OAuth authorization URL for the specified provider
func (s *OAuthService) GetAuthURL(provider, email string) (string, error) {
	var oauthConfig *oauth2.Config
	switch provider {
	case "gmail":
		oauthConfig = &oauth2.Config{
			ClientID:     s.config.Google.ClientID,
			ClientSecret: s.config.Google.ClientSecret,
			RedirectURL:  s.config.Google.RedirectURL,
			Scopes: []string{
				"https://www.googleapis.com/auth/gmail.readonly",
				"https://www.googleapis.com/auth/userinfo.email",
			},
			Endpoint: google.Endpoint,
		}
	case "outlook":
		oauthConfig = &oauth2.Config{
			ClientID:     s.config.Outlook.ClientID,
			ClientSecret: s.config.Outlook.ClientSecret,
			RedirectURL:  s.config.Outlook.RedirectURL,
			Scopes: []string{
				"https://graph.microsoft.com/Mail.Read",
				"https://graph.microsoft.com/User.Read",
			},
			Endpoint: microsoft.AzureADEndpoint("common"),
		}
	default:
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}

	// Generate random state
	state := make([]byte, 32)
	if _, err := rand.Read(state); err != nil {
		return "", fmt.Errorf("failed to generate state: %v", err)
	}
	stateStr := base64.URLEncoding.EncodeToString(state)

	// Store state with email for verification
	s.states[stateStr] = email

	// Generate authorization URL
	authURL := oauthConfig.AuthCodeURL(stateStr, oauth2.AccessTypeOffline)
	return authURL, nil
}

// HandleCallback processes the OAuth callback and creates/updates the account
func (s *OAuthService) HandleCallback(ctx context.Context, code, state string) (*models.Account, error) {
	// Verify state
	email, ok := s.states[state]
	if !ok {
		return nil, fmt.Errorf("invalid state")
	}
	delete(s.states, state) // Clean up used state

	// Get provider from email domain
	var provider string
	switch {
	case email[len(email)-10:] == "@gmail.com":
		provider = "gmail"
	case email[len(email)-13:] == "@outlook.com" || email[len(email)-12:] == "@hotmail.com":
		provider = "outlook"
	default:
		return nil, fmt.Errorf("unsupported email domain")
	}

	// Get OAuth config
	var oauthConfig *oauth2.Config
	switch provider {
	case "gmail":
		oauthConfig = &oauth2.Config{
			ClientID:     s.config.Google.ClientID,
			ClientSecret: s.config.Google.ClientSecret,
			RedirectURL:  s.config.Google.RedirectURL,
			Scopes: []string{
				"https://www.googleapis.com/auth/gmail.readonly",
				"https://www.googleapis.com/auth/userinfo.email",
			},
			Endpoint: google.Endpoint,
		}
	case "outlook":
		oauthConfig = &oauth2.Config{
			ClientID:     s.config.Outlook.ClientID,
			ClientSecret: s.config.Outlook.ClientSecret,
			RedirectURL:  s.config.Outlook.RedirectURL,
			Scopes: []string{
				"https://graph.microsoft.com/Mail.Read",
				"https://graph.microsoft.com/User.Read",
			},
			Endpoint: microsoft.AzureADEndpoint("common"),
		}
	}

	// Exchange code for token
	token, err := oauthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %v", err)
	}

	// Get user email from provider
	userEmail, err := s.getUserEmail(ctx, provider, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get user email: %v", err)
	}

	// Verify email matches
	if userEmail != email {
		return nil, fmt.Errorf("email mismatch: expected %s, got %s", email, userEmail)
	}

	// Create or update account
	account, err := s.store.GetAccountByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %v", err)
	}

	if account == nil {
		// Create new account
		account = &models.Account{
			Provider:     provider,
			Email:        email,
			AccessToken:  token.AccessToken,
			RefreshToken: token.RefreshToken,
			TokenExpiry:  token.Expiry,
		}
		if err := s.store.CreateAccount(ctx, account); err != nil {
			return nil, fmt.Errorf("failed to create account: %v", err)
		}
	} else {
		// Update existing account
		account.AccessToken = token.AccessToken
		account.RefreshToken = token.RefreshToken
		account.TokenExpiry = token.Expiry
		if err := s.store.UpdateAccount(ctx, account); err != nil {
			return nil, fmt.Errorf("failed to update account: %v", err)
		}
	}

	return account, nil
}

// RefreshToken refreshes the access token for an account
func (s *OAuthService) RefreshToken(ctx context.Context, account *models.Account) error {
	var oauthConfig *oauth2.Config
	switch account.Provider {
	case "gmail":
		oauthConfig = &oauth2.Config{
			ClientID:     s.config.Google.ClientID,
			ClientSecret: s.config.Google.ClientSecret,
			RedirectURL:  s.config.Google.RedirectURL,
			Scopes: []string{
				"https://www.googleapis.com/auth/gmail.readonly",
				"https://www.googleapis.com/auth/userinfo.email",
			},
			Endpoint: google.Endpoint,
		}
	case "outlook":
		oauthConfig = &oauth2.Config{
			ClientID:     s.config.Outlook.ClientID,
			ClientSecret: s.config.Outlook.ClientSecret,
			RedirectURL:  s.config.Outlook.RedirectURL,
			Scopes: []string{
				"https://graph.microsoft.com/Mail.Read",
				"https://graph.microsoft.com/User.Read",
			},
			Endpoint: microsoft.AzureADEndpoint("common"),
		}
	default:
		return fmt.Errorf("unsupported provider: %s", account.Provider)
	}

	token := &oauth2.Token{
		AccessToken:  account.AccessToken,
		RefreshToken: account.RefreshToken,
		Expiry:       account.TokenExpiry,
	}

	newToken, err := oauthConfig.TokenSource(ctx, token).Token()
	if err != nil {
		return fmt.Errorf("failed to refresh token: %v", err)
	}

	account.AccessToken = newToken.AccessToken
	account.RefreshToken = newToken.RefreshToken
	account.TokenExpiry = newToken.Expiry

	if err := s.store.UpdateAccount(ctx, account); err != nil {
		return fmt.Errorf("failed to update account: %v", err)
	}

	return nil
}

// getUserEmail retrieves the user's email from the provider
func (s *OAuthService) getUserEmail(ctx context.Context, provider string, token *oauth2.Token) (string, error) {
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

	switch provider {
	case "gmail":
		resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("failed to get user info: %s", resp.Status)
		}

		var userInfo struct {
			Email string `json:"email"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
			return "", err
		}
		return userInfo.Email, nil

	case "outlook":
		resp, err := client.Get("https://graph.microsoft.com/v1.0/me")
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("failed to get user info: %s", resp.Status)
		}

		var userInfo struct {
			Mail string `json:"mail"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
			return "", err
		}
		return userInfo.Mail, nil

	default:
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}
} 