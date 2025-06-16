package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"email-harvester/backend/internal/models"
	"email-harvester/backend/internal/services"
)

// UserHandler handles user-related HTTP requests
type UserHandler struct {
	userService *services.UserService
	monitor     *monitoring.Monitor
}

// NewUserHandler creates a new user handler
func NewUserHandler(userService *services.UserService, monitor *monitoring.Monitor) *UserHandler {
	return &UserHandler{
		userService: userService,
		monitor:     monitor,
	}
}

// RegisterRoutes registers the user routes
func (h *UserHandler) RegisterRoutes(r chi.Router) {
	r.Group(func(r chi.Router) {
		// Public routes
		r.Post("/register", h.handleRegister)
		r.Post("/login", h.handleLogin)
	})

	r.Group(func(r chi.Router) {
		// Protected routes
		r.Use(jwtauth.Verifier(tokenAuth))
		r.Use(jwtauth.Authenticator)

		r.Get("/profile", h.handleGetProfile)
		r.Put("/profile", h.handleUpdateProfile)
		r.Put("/profile/password", h.handleChangePassword)
		r.Post("/profile/2fa/enable", h.handleEnable2FA)
		r.Post("/profile/2fa/disable", h.handleDisable2FA)
		r.Post("/profile/2fa/verify", h.handleVerify2FA)
		r.Put("/profile/preferences", h.handleUpdatePreferences)
		r.Delete("/profile", h.handleDeleteAccount)
	})
}

// handleRegister handles user registration
func (h *UserHandler) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required,min=8"`
		Name     string `json:"name" validate:"required,min=2,max=100"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.userService.Create(r.Context(), req.Email, req.Password, req.Name)
	if err != nil {
		switch err {
		case services.ErrEmailExists:
			http.Error(w, "Email already exists", http.StatusConflict)
		default:
			h.monitor.LogError("Failed to create user", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Generate JWT token
	token, _, err := tokenAuth.Encode(map[string]interface{}{
		"user_id": user.ID.Hex(),
		"email":   user.Email,
	})
	if err != nil {
		h.monitor.LogError("Failed to generate token", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"token": token,
		"user":  user,
	})
}

// handleLogin handles user login
func (h *UserHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required"`
		OTPCode  string `json:"otpCode,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.userService.GetByEmail(r.Context(), req.Email)
	if err != nil {
		if err == services.ErrUserNotFound {
			http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		} else {
			h.monitor.LogError("Failed to get user", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	if !user.ValidatePassword(req.Password) {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	// If 2FA is enabled, verify OTP code
	if user.TwoFactorEnabled {
		if req.OTPCode == "" {
			http.Error(w, "2FA code required", http.StatusUnauthorized)
			return
		}

		if err := h.userService.Verify2FA(r.Context(), user.ID, req.OTPCode); err != nil {
			if err == services.ErrInvalidOTP {
				http.Error(w, "Invalid 2FA code", http.StatusUnauthorized)
			} else {
				h.monitor.LogError("Failed to verify 2FA", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
			return
		}
	}

	// Update last login
	if err := h.userService.UpdateLastLogin(r.Context(), user.ID); err != nil {
		h.monitor.LogError("Failed to update last login", err)
	}

	// Generate JWT token
	token, _, err := tokenAuth.Encode(map[string]interface{}{
		"user_id": user.ID.Hex(),
		"email":   user.Email,
	})
	if err != nil {
		h.monitor.LogError("Failed to generate token", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"token": token,
		"user":  user,
	})
}

// handleGetProfile handles getting the user's profile
func (h *UserHandler) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	_, claims, _ := jwtauth.FromContext(r.Context())
	userID, _ := primitive.ObjectIDFromHex(claims["user_id"].(string))

	user, err := h.userService.GetByID(r.Context(), userID)
	if err != nil {
		if err == services.ErrUserNotFound {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			h.monitor.LogError("Failed to get user", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	json.NewEncoder(w).Encode(user)
}

// handleUpdateProfile handles updating the user's profile
func (h *UserHandler) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	_, claims, _ := jwtauth.FromContext(r.Context())
	userID, _ := primitive.ObjectIDFromHex(claims["user_id"].(string))

	var req models.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.userService.UpdateProfile(r.Context(), userID, req)
	if err != nil {
		if err == services.ErrUserNotFound {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			h.monitor.LogError("Failed to update profile", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	json.NewEncoder(w).Encode(user)
}

// handleChangePassword handles changing the user's password
func (h *UserHandler) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	_, claims, _ := jwtauth.FromContext(r.Context())
	userID, _ := primitive.ObjectIDFromHex(claims["user_id"].(string))

	var req models.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.userService.ChangePassword(r.Context(), userID, req); err != nil {
		switch err {
		case services.ErrUserNotFound:
			http.Error(w, "User not found", http.StatusNotFound)
		case services.ErrInvalidPassword:
			http.Error(w, "Invalid current password", http.StatusUnauthorized)
		default:
			h.monitor.LogError("Failed to change password", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleEnable2FA handles enabling 2FA for a user
func (h *UserHandler) handleEnable2FA(w http.ResponseWriter, r *http.Request) {
	_, claims, _ := jwtauth.FromContext(r.Context())
	userID, _ := primitive.ObjectIDFromHex(claims["user_id"].(string))

	secret, err := h.userService.Enable2FA(r.Context(), userID)
	if err != nil {
		switch err {
		case services.ErrUserNotFound:
			http.Error(w, "User not found", http.StatusNotFound)
		case services.ErrTwoFactorEnabled:
			http.Error(w, "2FA is already enabled", http.StatusBadRequest)
		default:
			h.monitor.LogError("Failed to enable 2FA", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"secret": secret,
	})
}

// handleDisable2FA handles disabling 2FA for a user
func (h *UserHandler) handleDisable2FA(w http.ResponseWriter, r *http.Request) {
	_, claims, _ := jwtauth.FromContext(r.Context())
	userID, _ := primitive.ObjectIDFromHex(claims["user_id"].(string))

	var req struct {
		Code string `json:"code" validate:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.userService.Disable2FA(r.Context(), userID, req.Code); err != nil {
		switch err {
		case services.ErrUserNotFound:
			http.Error(w, "User not found", http.StatusNotFound)
		case services.ErrTwoFactorDisabled:
			http.Error(w, "2FA is not enabled", http.StatusBadRequest)
		case services.ErrInvalidOTP:
			http.Error(w, "Invalid 2FA code", http.StatusUnauthorized)
		default:
			h.monitor.LogError("Failed to disable 2FA", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleVerify2FA handles verifying a 2FA code
func (h *UserHandler) handleVerify2FA(w http.ResponseWriter, r *http.Request) {
	_, claims, _ := jwtauth.FromContext(r.Context())
	userID, _ := primitive.ObjectIDFromHex(claims["user_id"].(string))

	var req struct {
		Code string `json:"code" validate:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.userService.Verify2FA(r.Context(), userID, req.Code); err != nil {
		switch err {
		case services.ErrUserNotFound:
			http.Error(w, "User not found", http.StatusNotFound)
		case services.ErrTwoFactorDisabled:
			http.Error(w, "2FA is not enabled", http.StatusBadRequest)
		case services.ErrInvalidOTP:
			http.Error(w, "Invalid 2FA code", http.StatusUnauthorized)
		default:
			h.monitor.LogError("Failed to verify 2FA", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleUpdatePreferences handles updating user preferences
func (h *UserHandler) handleUpdatePreferences(w http.ResponseWriter, r *http.Request) {
	_, claims, _ := jwtauth.FromContext(r.Context())
	userID, _ := primitive.ObjectIDFromHex(claims["user_id"].(string))

	var prefs models.UserPreferences
	if err := json.NewDecoder(r.Body).Decode(&prefs); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.userService.UpdatePreferences(r.Context(), userID, prefs); err != nil {
		if err == services.ErrUserNotFound {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			h.monitor.LogError("Failed to update preferences", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleDeleteAccount handles deleting a user's account
func (h *UserHandler) handleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	_, claims, _ := jwtauth.FromContext(r.Context())
	userID, _ := primitive.ObjectIDFromHex(claims["user_id"].(string))

	var req struct {
		Password string `json:"password,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.userService.DeleteAccount(r.Context(), userID, req.Password); err != nil {
		switch err {
		case services.ErrUserNotFound:
			http.Error(w, "User not found", http.StatusNotFound)
		case services.ErrInvalidPassword:
			http.Error(w, "Invalid password", http.StatusUnauthorized)
		default:
			h.monitor.LogError("Failed to delete account", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
} 