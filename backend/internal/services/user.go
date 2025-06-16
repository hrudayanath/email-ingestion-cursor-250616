package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"

	"email-harvester/backend/internal/models"
	"email-harvester/backend/internal/monitoring"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrInvalidPassword   = errors.New("invalid password")
	ErrEmailExists       = errors.New("email already exists")
	ErrInvalidOTP        = errors.New("invalid OTP code")
	ErrTwoFactorDisabled = errors.New("2FA is not enabled")
	ErrTwoFactorEnabled  = errors.New("2FA is already enabled")
)

// UserService handles user-related operations
type UserService struct {
	db        *mongo.Database
	monitor   *monitoring.Monitor
	usersColl *mongo.Collection
}

// NewUserService creates a new user service
func NewUserService(db *mongo.Database, monitor *monitoring.Monitor) *UserService {
	return &UserService{
		db:        db,
		monitor:   monitor,
		usersColl: db.Collection("users"),
	}
}

// GetByID retrieves a user by their ID
func (s *UserService) GetByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	var user models.User
	err := s.usersColl.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetByEmail retrieves a user by their email
func (s *UserService) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := s.usersColl.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// Create creates a new user
func (s *UserService) Create(ctx context.Context, email, password, name string) (*models.User, error) {
	// Check if user already exists
	_, err := s.GetByEmail(ctx, email)
	if err == nil {
		return nil, ErrEmailExists
	} else if err != ErrUserNotFound {
		return nil, err
	}

	user, err := models.NewUser(email, password, name)
	if err != nil {
		return nil, err
	}

	_, err = s.usersColl.InsertOne(ctx, user)
	if err != nil {
		return nil, err
	}

	s.monitor.RecordMetric("user_created", 1, map[string]string{
		"provider": user.Provider,
	})

	return user, nil
}

// UpdateProfile updates a user's profile information
func (s *UserService) UpdateProfile(ctx context.Context, id primitive.ObjectID, req models.UpdateProfileRequest) (*models.User, error) {
	update := bson.M{
		"$set": bson.M{
			"name":      req.Name,
			"picture":   req.Picture,
			"updatedAt": time.Now(),
		},
	}

	result := s.usersColl.FindOneAndUpdate(
		ctx,
		bson.M{"_id": id},
		update,
		mongo.ReturnDocument(mongo.After),
	)

	var user models.User
	if err := result.Decode(&user); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	s.monitor.RecordMetric("profile_updated", 1, nil)
	return &user, nil
}

// ChangePassword changes a user's password
func (s *UserService) ChangePassword(ctx context.Context, id primitive.ObjectID, req models.ChangePasswordRequest) error {
	user, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if !user.ValidatePassword(req.CurrentPassword) {
		return ErrInvalidPassword
	}

	if err := user.SetPassword(req.NewPassword); err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"passwordHash": user.PasswordHash,
			"updatedAt":    time.Now(),
		},
	}

	result := s.usersColl.UpdateOne(ctx, bson.M{"_id": id}, update)
	if result.Err != nil {
		return result.Err
	}

	s.monitor.RecordMetric("password_changed", 1, nil)
	return nil
}

// Enable2FA enables two-factor authentication for a user
func (s *UserService) Enable2FA(ctx context.Context, id primitive.ObjectID) (string, error) {
	user, err := s.GetByID(ctx, id)
	if err != nil {
		return "", err
	}

	if user.TwoFactorEnabled {
		return "", ErrTwoFactorEnabled
	}

	// Generate a new TOTP secret
	secret, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Email Harvester",
		AccountName: user.Email,
	})
	if err != nil {
		return "", err
	}

	update := bson.M{
		"$set": bson.M{
			"twoFactorSecret":   secret.Secret(),
			"twoFactorEnabled": true,
			"updatedAt":        time.Now(),
		},
	}

	result := s.usersColl.UpdateOne(ctx, bson.M{"_id": id}, update)
	if result.Err != nil {
		return "", result.Err
	}

	s.monitor.RecordMetric("2fa_enabled", 1, nil)
	return secret.Secret(), nil
}

// Disable2FA disables two-factor authentication for a user
func (s *UserService) Disable2FA(ctx context.Context, id primitive.ObjectID, code string) error {
	user, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if !user.TwoFactorEnabled {
		return ErrTwoFactorDisabled
	}

	// Verify the TOTP code
	if !totp.Validate(code, user.TwoFactorSecret) {
		return ErrInvalidOTP
	}

	update := bson.M{
		"$set": bson.M{
			"twoFactorEnabled": false,
			"twoFactorSecret": "",
			"updatedAt":       time.Now(),
		},
	}

	result := s.usersColl.UpdateOne(ctx, bson.M{"_id": id}, update)
	if result.Err != nil {
		return result.Err
	}

	s.monitor.RecordMetric("2fa_disabled", 1, nil)
	return nil
}

// Verify2FA verifies a 2FA code for a user
func (s *UserService) Verify2FA(ctx context.Context, id primitive.ObjectID, code string) error {
	user, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if !user.TwoFactorEnabled {
		return ErrTwoFactorDisabled
	}

	if !totp.Validate(code, user.TwoFactorSecret) {
		return ErrInvalidOTP
	}

	return nil
}

// UpdatePreferences updates a user's preferences
func (s *UserService) UpdatePreferences(ctx context.Context, id primitive.ObjectID, prefs models.UserPreferences) error {
	update := bson.M{
		"$set": bson.M{
			"preferences": prefs,
			"updatedAt":  time.Now(),
		},
	}

	result := s.usersColl.UpdateOne(ctx, bson.M{"_id": id}, update)
	if result.Err != nil {
		return result.Err
	}

	s.monitor.RecordMetric("preferences_updated", 1, nil)
	return nil
}

// DeleteAccount deletes a user's account
func (s *UserService) DeleteAccount(ctx context.Context, id primitive.ObjectID, password string) error {
	user, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if !user.IsOAuthUser() && !user.ValidatePassword(password) {
		return ErrInvalidPassword
	}

	result := s.usersColl.DeleteOne(ctx, bson.M{"_id": id})
	if result.Err != nil {
		return result.Err
	}

	s.monitor.RecordMetric("account_deleted", 1, map[string]string{
		"provider": user.Provider,
	})
	return nil
}

// UpdateLastLogin updates the user's last login timestamp
func (s *UserService) UpdateLastLogin(ctx context.Context, id primitive.ObjectID) error {
	update := bson.M{
		"$set": bson.M{
			"lastLogin": time.Now(),
		},
	}

	result := s.usersColl.UpdateOne(ctx, bson.M{"_id": id}, update)
	if result.Err != nil {
		return result.Err
	}

	s.monitor.RecordMetric("user_login", 1, nil)
	return nil
} 