package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

// User represents a user in the system
type User struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Email          string            `bson:"email" json:"email"`
	Name           string            `bson:"name" json:"name"`
	PasswordHash   string            `bson:"passwordHash,omitempty" json:"-"`
	Picture        string            `bson:"picture" json:"picture"`
	Provider       string            `bson:"provider" json:"provider"` // "local", "google", or "microsoft"
	ProviderID     string            `bson:"providerId,omitempty" json:"providerId,omitempty"`
	RefreshToken   string            `bson:"refreshToken,omitempty" json:"-"`
	LastLogin      time.Time         `bson:"lastLogin" json:"lastLogin"`
	CreatedAt      time.Time         `bson:"createdAt" json:"createdAt"`
	UpdatedAt      time.Time         `bson:"updatedAt" json:"updatedAt"`
	EmailVerified  bool              `bson:"emailVerified" json:"emailVerified"`
	TwoFactorEnabled bool            `bson:"twoFactorEnabled" json:"twoFactorEnabled"`
	TwoFactorSecret string           `bson:"twoFactorSecret,omitempty" json:"-"`
	Preferences    UserPreferences   `bson:"preferences" json:"preferences"`
}

// UserPreferences stores user-specific settings
type UserPreferences struct {
	Theme           string `bson:"theme" json:"theme"` // "light" or "dark"
	EmailNotifications bool `bson:"emailNotifications" json:"emailNotifications"`
	Language        string `bson:"language" json:"language"`
}

// UpdateProfileRequest represents a request to update user profile
type UpdateProfileRequest struct {
	Name    string `json:"name" validate:"required,min=2,max=100"`
	Picture string `json:"picture,omitempty"`
}

// ChangePasswordRequest represents a request to change password
type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" validate:"required,min=8"`
	NewPassword     string `json:"newPassword" validate:"required,min=8"`
}

// ValidatePassword checks if the provided password matches the stored hash
func (u *User) ValidatePassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// SetPassword hashes and sets the user's password
func (u *User) SetPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	return nil
}

// BeforeCreate sets timestamps before creating a new user
func (u *User) BeforeCreate() {
	now := time.Now()
	u.CreatedAt = now
	u.UpdatedAt = now
	u.LastLogin = now
}

// BeforeUpdate sets the updated timestamp
func (u *User) BeforeUpdate() {
	u.UpdatedAt = time.Now()
}

// IsOAuthUser returns true if the user was created through OAuth
func (u *User) IsOAuthUser() bool {
	return u.Provider != "local"
}

// NewUser creates a new user with the given email and password
func NewUser(email, password, name string) (*User, error) {
	user := &User{
		Email:         email,
		Name:          name,
		Provider:      "local",
		EmailVerified: false,
		Preferences: UserPreferences{
			Theme:              "light",
			EmailNotifications: true,
			Language:          "en",
		},
	}

	if password != "" {
		if err := user.SetPassword(password); err != nil {
			return nil, err
		}
	}

	user.BeforeCreate()
	return user, nil
}

// NewOAuthUser creates a new user from OAuth provider data
func NewOAuthUser(email, name, picture, provider, providerID string) *User {
	user := &User{
		Email:         email,
		Name:          name,
		Picture:       picture,
		Provider:      provider,
		ProviderID:    providerID,
		EmailVerified: true, // OAuth users are considered verified
		Preferences: UserPreferences{
			Theme:              "light",
			EmailNotifications: true,
			Language:          "en",
		},
	}

	user.BeforeCreate()
	return user
} 