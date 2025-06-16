package models

import "time"

// OAuthTokens represents the OAuth tokens received from the provider
type OAuthTokens struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// UserInfo represents the user information received from the provider
type UserInfo struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture,omitempty"`
}

// Account represents a user's email account
type Account struct {
	ID           string    `json:"id" bson:"_id,omitempty"`
	Provider     string    `json:"provider" bson:"provider"` // "google" or "microsoft"
	Email        string    `json:"email" bson:"email"`
	Name         string    `json:"name" bson:"name"`
	Picture      string    `json:"picture,omitempty" bson:"picture,omitempty"`
	AccessToken  string    `json:"-" bson:"access_token"`  // Not exposed in JSON
	RefreshToken string    `json:"-" bson:"refresh_token"` // Not exposed in JSON
	ExpiresAt    time.Time `json:"-" bson:"expires_at"`    // Not exposed in JSON
	TokenType    string    `json:"-" bson:"token_type"`    // Not exposed in JSON
	CreatedAt    time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" bson:"updated_at"`
	LastSyncAt   time.Time `json:"last_sync_at" bson:"last_sync_at"`
	IsActive     bool      `json:"is_active" bson:"is_active"`
}

// AccountCreate represents the data needed to create a new account
type AccountCreate struct {
	Provider     string    `json:"provider" validate:"required,oneof=google microsoft"`
	Email        string    `json:"email" validate:"required,email"`
	Name         string    `json:"name" validate:"required"`
	Picture      string    `json:"picture,omitempty"`
	AccessToken  string    `json:"access_token" validate:"required"`
	RefreshToken string    `json:"refresh_token" validate:"required"`
	ExpiresAt    time.Time `json:"expires_at" validate:"required"`
	TokenType    string    `json:"token_type" validate:"required"`
}

// AccountUpdate represents the data needed to update an existing account
type AccountUpdate struct {
	Name         *string    `json:"name,omitempty"`
	Picture      *string    `json:"picture,omitempty"`
	AccessToken  *string    `json:"access_token,omitempty"`
	RefreshToken *string    `json:"refresh_token,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	TokenType    *string    `json:"token_type,omitempty"`
	LastSyncAt   *time.Time `json:"last_sync_at,omitempty"`
	IsActive     *bool      `json:"is_active,omitempty"`
}

// AccountResponse represents the account data returned in API responses
type AccountResponse struct {
	ID         string    `json:"id"`
	Provider   string    `json:"provider"`
	Email      string    `json:"email"`
	Name       string    `json:"name"`
	Picture    string    `json:"picture,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	LastSyncAt time.Time `json:"last_sync_at"`
	IsActive   bool      `json:"is_active"`
}

// ToResponse converts an Account to an AccountResponse
func (a *Account) ToResponse() *AccountResponse {
	return &AccountResponse{
		ID:         a.ID,
		Provider:   a.Provider,
		Email:      a.Email,
		Name:       a.Name,
		Picture:    a.Picture,
		CreatedAt:  a.CreatedAt,
		UpdatedAt:  a.UpdatedAt,
		LastSyncAt: a.LastSyncAt,
		IsActive:   a.IsActive,
	}
}

// Update applies the fields from AccountUpdate to the Account
func (a *Account) Update(update *AccountUpdate) {
	if update.Name != nil {
		a.Name = *update.Name
	}
	if update.Picture != nil {
		a.Picture = *update.Picture
	}
	if update.AccessToken != nil {
		a.AccessToken = *update.AccessToken
	}
	if update.RefreshToken != nil {
		a.RefreshToken = *update.RefreshToken
	}
	if update.ExpiresAt != nil {
		a.ExpiresAt = *update.ExpiresAt
	}
	if update.TokenType != nil {
		a.TokenType = *update.TokenType
	}
	if update.LastSyncAt != nil {
		a.LastSyncAt = *update.LastSyncAt
	}
	if update.IsActive != nil {
		a.IsActive = *update.IsActive
	}
	a.UpdatedAt = time.Now()
}

// FromCreate converts an AccountCreate to an Account
func FromCreate(create *AccountCreate) *Account {
	now := time.Now()
	return &Account{
		Provider:     create.Provider,
		Email:        create.Email,
		Name:         create.Name,
		Picture:      create.Picture,
		AccessToken:  create.AccessToken,
		RefreshToken: create.RefreshToken,
		ExpiresAt:    create.ExpiresAt,
		TokenType:    create.TokenType,
		CreatedAt:    now,
		UpdatedAt:    now,
		LastSyncAt:   now,
		IsActive:     true,
	}
} 