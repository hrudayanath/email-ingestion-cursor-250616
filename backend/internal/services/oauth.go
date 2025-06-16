package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/confidential"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
	"go.uber.org/zap"

	"email-harvester/internal/config"
	"email-harvester/internal/monitoring"
	"email-harvester/internal/models"
	"email-harvester/internal/store"
)

// OAuthService handles OAuth authentication for email providers
type OAuthService struct {
	config  *config.Config
	monitor *monitoring.Monitor
	// Microsoft MSAL clients
	msalPublicClient     public.Client
	msalConfidentialApp  confidential.Client
	store  *store.MongoStore
	states map[string]string // In-memory state store for OAuth flow
}

// NewOAuthService creates a new OAuth service
func NewOAuthService(cfg *config.Config, monitor *monitoring.Monitor) (*OAuthService, error) {
	// Initialize Microsoft MSAL clients
	msalPublicClient, err := public.New(cfg.OAuth.Microsoft.ClientID,
		public.WithAuthority(cfg.OAuth.Microsoft.Authority))
	if err != nil {
		return nil, fmt.Errorf("failed to create MSAL public client: %w", err)
	}

	cred, err := confidential.NewCredFromSecret(cfg.OAuth.Microsoft.ClientSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to create MSAL credential: %w", err)
	}

	msalConfidentialApp, err := confidential.New(cfg.OAuth.Microsoft.ClientID, cred,
		confidential.WithAuthority(cfg.OAuth.Microsoft.Authority))
	if err != nil {
		return nil, fmt.Errorf("failed to create MSAL confidential client: %w", err)
	}

	return &OAuthService{
		config:              cfg,
		monitor:             monitor,
		msalPublicClient:    msalPublicClient,
		msalConfidentialApp: msalConfidentialApp,
		states:              make(map[string]string),
	}, nil
}

// SetStore sets the store for the OAuth service
func (s *OAuthService) SetStore(store *store.MongoStore) {
	s.store = store
}

// GetAuthURL generates an authorization URL for the specified provider
func (s *OAuthService) GetAuthURL(ctx context.Context, provider string, state string) (string, error) {
	ctx, span := s.monitor.WithSpan(ctx, "oauth.get_auth_url")
	defer span.End()

	s.monitor.LogDebug("Generating auth URL",
		zap.String("provider", provider),
		zap.String("state", state),
	)

	switch provider {
	case "google":
		return s.getGoogleAuthURL(state)
	case "microsoft":
		return s.getMicrosoftAuthURL(ctx, state)
	default:
		err := fmt.Errorf("unsupported provider: %s", provider)
		s.monitor.RecordError(span, err)
		return "", err
	}
}

// getGoogleAuthURL generates a Google OAuth authorization URL
func (s *OAuthService) getGoogleAuthURL(state string) (string, error) {
	params := url.Values{}
	params.Set("client_id", s.config.OAuth.Google.ClientID)
	params.Set("redirect_uri", s.config.OAuth.Google.RedirectURL)
	params.Set("response_type", "code")
	params.Set("scope", strings.Join(s.config.OAuth.Google.Scopes, " "))
	params.Set("access_type", "offline")
	params.Set("prompt", "consent")
	params.Set("state", state)

	return fmt.Sprintf("https://accounts.google.com/o/oauth2/v2/auth?%s", params.Encode()), nil
}

// getMicrosoftAuthURL generates a Microsoft OAuth authorization URL
func (s *OAuthService) getMicrosoftAuthURL(ctx context.Context, state string) (string, error) {
	authURL, err := s.msalPublicClient.CreateAuthCodeURL(ctx, s.config.OAuth.Microsoft.ClientID,
		s.config.OAuth.Microsoft.RedirectURL, s.config.OAuth.Microsoft.Scopes,
		public.WithState(state))
	if err != nil {
		return "", fmt.Errorf("failed to create Microsoft auth URL: %w", err)
	}
	return authURL, nil
}

// HandleCallback processes the OAuth callback and returns the tokens
func (s *OAuthService) HandleCallback(ctx context.Context, provider string, code string) (*models.OAuthTokens, error) {
	ctx, span := s.monitor.WithSpan(ctx, "oauth.handle_callback")
	defer span.End()

	s.monitor.LogDebug("Handling OAuth callback",
		zap.String("provider", provider),
	)

	switch provider {
	case "google":
		return s.handleGoogleCallback(ctx, code)
	case "microsoft":
		return s.handleMicrosoftCallback(ctx, code)
	default:
		err := fmt.Errorf("unsupported provider: %s", provider)
		s.monitor.RecordError(span, err)
		return nil, err
	}
}

// handleGoogleCallback processes the Google OAuth callback
func (s *OAuthService) handleGoogleCallback(ctx context.Context, code string) (*models.OAuthTokens, error) {
	params := url.Values{}
	params.Set("client_id", s.config.OAuth.Google.ClientID)
	params.Set("client_secret", s.config.OAuth.Google.ClientSecret)
	params.Set("code", code)
	params.Set("grant_type", "authorization_code")
	params.Set("redirect_uri", s.config.OAuth.Google.RedirectURL)

	resp, err := http.PostForm("https://oauth2.googleapis.com/token", params)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status: %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	return &models.OAuthTokens{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
		TokenType:    tokenResp.TokenType,
	}, nil
}

// handleMicrosoftCallback processes the Microsoft OAuth callback
func (s *OAuthService) handleMicrosoftCallback(ctx context.Context, code string) (*models.OAuthTokens, error) {
	result, err := s.msalPublicClient.AcquireTokenByAuthCode(ctx, code,
		s.config.OAuth.Microsoft.RedirectURL, s.config.OAuth.Microsoft.Scopes)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire token: %w", err)
	}

	return &models.OAuthTokens{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    result.ExpiresOn,
		TokenType:    "Bearer",
	}, nil
}

// RefreshToken refreshes the OAuth tokens for the specified provider
func (s *OAuthService) RefreshToken(ctx context.Context, provider string, refreshToken string) (*models.OAuthTokens, error) {
	ctx, span := s.monitor.WithSpan(ctx, "oauth.refresh_token")
	defer span.End()

	s.monitor.LogDebug("Refreshing OAuth token",
		zap.String("provider", provider),
	)

	switch provider {
	case "google":
		return s.refreshGoogleToken(ctx, refreshToken)
	case "microsoft":
		return s.refreshMicrosoftToken(ctx, refreshToken)
	default:
		err := fmt.Errorf("unsupported provider: %s", provider)
		s.monitor.RecordError(span, err)
		return nil, err
	}
}

// refreshGoogleToken refreshes a Google OAuth token
func (s *OAuthService) refreshGoogleToken(ctx context.Context, refreshToken string) (*models.OAuthTokens, error) {
	params := url.Values{}
	params.Set("client_id", s.config.OAuth.Google.ClientID)
	params.Set("client_secret", s.config.OAuth.Google.ClientSecret)
	params.Set("refresh_token", refreshToken)
	params.Set("grant_type", "refresh_token")

	resp, err := http.PostForm("https://oauth2.googleapis.com/token", params)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed with status: %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	return &models.OAuthTokens{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: refreshToken, // Google returns the same refresh token
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
		TokenType:    tokenResp.TokenType,
	}, nil
}

// refreshMicrosoftToken refreshes a Microsoft OAuth token
func (s *OAuthService) refreshMicrosoftToken(ctx context.Context, refreshToken string) (*models.OAuthTokens, error) {
	result, err := s.msalConfidentialApp.AcquireTokenByRefreshToken(ctx, refreshToken,
		s.config.OAuth.Microsoft.Scopes)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	return &models.OAuthTokens{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    result.ExpiresOn,
		TokenType:    "Bearer",
	}, nil
}

// GetUserInfo retrieves user information from the provider
func (s *OAuthService) GetUserInfo(ctx context.Context, provider string, accessToken string) (*models.UserInfo, error) {
	ctx, span := s.monitor.WithSpan(ctx, "oauth.get_user_info")
	defer span.End()

	s.monitor.LogDebug("Getting user info",
		zap.String("provider", provider),
	)

	switch provider {
	case "google":
		return s.getGoogleUserInfo(ctx, accessToken)
	case "microsoft":
		return s.getMicrosoftUserInfo(ctx, accessToken)
	default:
		err := fmt.Errorf("unsupported provider: %s", provider)
		s.monitor.RecordError(span, err)
		return nil, err
	}
}

// getGoogleUserInfo retrieves user information from Google
func (s *OAuthService) getGoogleUserInfo(ctx context.Context, accessToken string) (*models.UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user info with status: %d", resp.StatusCode)
	}

	var userInfo struct {
		ID            string `json:"id"`
		Email         string `json:"email"`
		VerifiedEmail bool   `json:"verified_email"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &models.UserInfo{
		ID:      userInfo.ID,
		Email:   userInfo.Email,
		Name:    userInfo.Name,
		Picture: userInfo.Picture,
	}, nil
}

// getMicrosoftUserInfo retrieves user information from Microsoft
func (s *OAuthService) getMicrosoftUserInfo(ctx context.Context, accessToken string) (*models.UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://graph.microsoft.com/v1.0/me", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user info with status: %d", resp.StatusCode)
	}

	var userInfo struct {
		ID                string `json:"id"`
		UserPrincipalName string `json:"userPrincipalName"`
		DisplayName       string `json:"displayName"`
		Mail              string `json:"mail"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &models.UserInfo{
		ID:      userInfo.ID,
		Email:   userInfo.Mail,
		Name:    userInfo.DisplayName,
		Picture: "", // Microsoft Graph API doesn't provide profile picture by default
	}, nil
} 