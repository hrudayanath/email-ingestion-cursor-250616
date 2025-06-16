package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"go.uber.org/zap"

	"email-harvester/internal/monitoring"
	"email-harvester/internal/models"
	"email-harvester/internal/services"
)

// OAuthHandler handles OAuth-related HTTP requests
type OAuthHandler struct {
	oauthService *services.OAuthService
	monitor      *monitoring.Monitor
}

// NewOAuthHandler creates a new OAuth handler
func NewOAuthHandler(oauthService *services.OAuthService, monitor *monitoring.Monitor) *OAuthHandler {
	return &OAuthHandler{
		oauthService: oauthService,
		monitor:      monitor,
	}
}

// RegisterRoutes registers the OAuth routes
func (h *OAuthHandler) RegisterRoutes(r chi.Router) {
	r.Route("/oauth", func(r chi.Router) {
		r.Get("/auth/{provider}", h.GetAuthURL)
		r.Get("/callback/{provider}", h.HandleCallback)
		r.Post("/refresh/{provider}", h.RefreshToken)
	})
}

// GetAuthURLRequest represents the request to get an OAuth authorization URL
type GetAuthURLRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// GetAuthURLResponse represents the response containing the OAuth authorization URL
type GetAuthURLResponse struct {
	URL string `json:"url"`
}

// GetAuthURL handles the request to get an OAuth authorization URL
func (h *OAuthHandler) GetAuthURL(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	provider := chi.URLParam(r, "provider")

	// Generate a random state
	state, err := generateRandomState()
	if err != nil {
		h.monitor.LogError("Failed to generate state", err)
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, ErrorResponse{Error: "Internal server error"})
		return
	}

	// Get the authorization URL
	authURL, err := h.oauthService.GetAuthURL(ctx, provider, state)
	if err != nil {
		h.monitor.LogError("Failed to get auth URL", err,
			zap.String("provider", provider))
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrorResponse{Error: "Invalid provider"})
		return
	}

	render.JSON(w, r, GetAuthURLResponse{URL: authURL})
}

// HandleCallbackRequest represents the request to handle an OAuth callback
type HandleCallbackRequest struct {
	Code  string `json:"code" validate:"required"`
	State string `json:"state" validate:"required"`
}

// HandleCallback handles the OAuth callback
func (h *OAuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	provider := chi.URLParam(r, "provider")

	// Parse the request
	var req HandleCallbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.monitor.LogError("Failed to decode request", err)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrorResponse{Error: "Invalid request"})
		return
	}

	// Handle the callback
	tokens, err := h.oauthService.HandleCallback(ctx, provider, req.Code)
	if err != nil {
		h.monitor.LogError("Failed to handle callback", err,
			zap.String("provider", provider))
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrorResponse{Error: "Invalid code"})
		return
	}

	// Get user info
	userInfo, err := h.oauthService.GetUserInfo(ctx, provider, tokens.AccessToken)
	if err != nil {
		h.monitor.LogError("Failed to get user info", err,
			zap.String("provider", provider))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, ErrorResponse{Error: "Failed to get user info"})
		return
	}

	// Create account
	account := &models.AccountCreate{
		Provider:     provider,
		Email:        userInfo.Email,
		Name:         userInfo.Name,
		Picture:      userInfo.Picture,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresAt:    tokens.ExpiresAt,
		TokenType:    tokens.TokenType,
	}

	// TODO: Save account to database
	// For now, just return the tokens and user info
	render.JSON(w, r, struct {
		Tokens    *models.OAuthTokens `json:"tokens"`
		UserInfo  *models.UserInfo    `json:"user_info"`
		Account   *models.AccountCreate `json:"account"`
	}{
		Tokens:   tokens,
		UserInfo: userInfo,
		Account:  account,
	})
}

// RefreshTokenRequest represents the request to refresh an OAuth token
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// RefreshToken handles the request to refresh an OAuth token
func (h *OAuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	provider := chi.URLParam(r, "provider")

	// Parse the request
	var req RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.monitor.LogError("Failed to decode request", err)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrorResponse{Error: "Invalid request"})
		return
	}

	// Refresh the token
	tokens, err := h.oauthService.RefreshToken(ctx, provider, req.RefreshToken)
	if err != nil {
		h.monitor.LogError("Failed to refresh token", err,
			zap.String("provider", provider))
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrorResponse{Error: "Invalid refresh token"})
		return
	}

	render.JSON(w, r, tokens)
}

// generateRandomState generates a random state string for OAuth
func generateRandomState() (string, error) {
	// TODO: Implement proper state generation
	// For now, just use a timestamp
	return time.Now().Format(time.RFC3339Nano), nil
} 