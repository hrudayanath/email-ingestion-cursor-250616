package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"email-harvester/internal/models"
)

// MongoStore implements the store interface using MongoDB
type MongoStore struct {
	db *mongo.Database
}

// NewMongoStore creates a new MongoDB store
func NewMongoStore(db *mongo.Database) *MongoStore {
	return &MongoStore{db: db}
}

// CreateAccount creates a new email account
func (s *MongoStore) CreateAccount(ctx context.Context, account *models.Account) error {
	account.CreatedAt = time.Now()
	account.UpdatedAt = time.Now()

	result, err := s.db.Collection("accounts").InsertOne(ctx, account)
	if err != nil {
		return err
	}

	account.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// GetAccount retrieves an account by ID
func (s *MongoStore) GetAccount(ctx context.Context, id primitive.ObjectID) (*models.Account, error) {
	var account models.Account
	err := s.db.Collection("accounts").FindOne(ctx, bson.M{"_id": id}).Decode(&account)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

// GetAccountByEmail retrieves an account by email
func (s *MongoStore) GetAccountByEmail(ctx context.Context, email string) (*models.Account, error) {
	var account models.Account
	err := s.db.Collection("accounts").FindOne(ctx, bson.M{"email": email}).Decode(&account)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

// UpdateAccount updates an existing account
func (s *MongoStore) UpdateAccount(ctx context.Context, account *models.Account) error {
	account.UpdatedAt = time.Now()

	update := bson.M{
		"$set": bson.M{
			"access_token":  account.AccessToken,
			"refresh_token": account.RefreshToken,
			"token_expiry":  account.TokenExpiry,
			"updated_at":    account.UpdatedAt,
		},
	}

	_, err := s.db.Collection("accounts").UpdateOne(
		ctx,
		bson.M{"_id": account.ID},
		update,
	)
	return err
}

// DeleteAccount deletes an account by ID
func (s *MongoStore) DeleteAccount(ctx context.Context, id primitive.ObjectID) error {
	_, err := s.db.Collection("accounts").DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// ListAccounts lists all accounts with pagination
func (s *MongoStore) ListAccounts(ctx context.Context, page, limit int) ([]models.Account, int64, error) {
	skip := (page - 1) * limit

	// Get total count
	total, err := s.db.Collection("accounts").CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, 0, err
	}

	// Find accounts
	opts := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(limit)).
		SetSort(bson.M{"created_at": -1})

	cursor, err := s.db.Collection("accounts").Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var accounts []models.Account
	if err := cursor.All(ctx, &accounts); err != nil {
		return nil, 0, err
	}

	return accounts, total, nil
}

// CreateEmail creates a new email
func (s *MongoStore) CreateEmail(ctx context.Context, email *models.Email) error {
	email.CreatedAt = time.Now()
	email.UpdatedAt = time.Now()

	result, err := s.db.Collection("emails").InsertOne(ctx, email)
	if err != nil {
		return err
	}

	email.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// GetEmail retrieves an email by ID
func (s *MongoStore) GetEmail(ctx context.Context, id primitive.ObjectID) (*models.Email, error) {
	var email models.Email
	err := s.db.Collection("emails").FindOne(ctx, bson.M{"_id": id}).Decode(&email)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &email, nil
}

// GetEmailByMessageID retrieves an email by message ID
func (s *MongoStore) GetEmailByMessageID(ctx context.Context, accountID primitive.ObjectID, messageID string) (*models.Email, error) {
	var email models.Email
	err := s.db.Collection("emails").FindOne(ctx, bson.M{
		"account_id": accountID,
		"message_id": messageID,
	}).Decode(&email)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &email, nil
}

// UpdateEmail updates an existing email
func (s *MongoStore) UpdateEmail(ctx context.Context, email *models.Email) error {
	email.UpdatedAt = time.Now()

	update := bson.M{
		"$set": bson.M{
			"summary":    email.Summary,
			"entities":   email.Entities,
			"labels":     email.Labels,
			"read":       email.Read,
			"starred":    email.Starred,
			"updated_at": email.UpdatedAt,
		},
	}

	_, err := s.db.Collection("emails").UpdateOne(
		ctx,
		bson.M{"_id": email.ID},
		update,
	)
	return err
}

// DeleteEmail deletes an email by ID
func (s *MongoStore) DeleteEmail(ctx context.Context, id primitive.ObjectID) error {
	_, err := s.db.Collection("emails").DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// ListEmails lists emails with filtering and pagination
func (s *MongoStore) ListEmails(ctx context.Context, filter models.EmailFilter, page, limit int) ([]models.Email, int64, error) {
	skip := (page - 1) * limit

	// Build filter
	mongoFilter := bson.M{}
	if filter.AccountID != nil {
		mongoFilter["account_id"] = *filter.AccountID
	}
	if filter.From != nil {
		mongoFilter["from"] = bson.M{"$regex": *filter.From, "$options": "i"}
	}
	if filter.To != nil {
		mongoFilter["to"] = bson.M{"$regex": *filter.To, "$options": "i"}
	}
	if filter.Subject != nil {
		mongoFilter["subject"] = bson.M{"$regex": *filter.Subject, "$options": "i"}
	}
	if filter.Label != nil {
		mongoFilter["labels"] = *filter.Label
	}
	if filter.Read != nil {
		mongoFilter["read"] = *filter.Read
	}
	if filter.Starred != nil {
		mongoFilter["starred"] = *filter.Starred
	}
	if filter.StartDate != nil || filter.EndDate != nil {
		dateFilter := bson.M{}
		if filter.StartDate != nil {
			dateFilter["$gte"] = *filter.StartDate
		}
		if filter.EndDate != nil {
			dateFilter["$lte"] = *filter.EndDate
		}
		mongoFilter["received_at"] = dateFilter
	}

	// Get total count
	total, err := s.db.Collection("emails").CountDocuments(ctx, mongoFilter)
	if err != nil {
		return nil, 0, err
	}

	// Find emails
	opts := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(limit)).
		SetSort(bson.M{"received_at": -1})

	cursor, err := s.db.Collection("emails").Find(ctx, mongoFilter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var emails []models.Email
	if err := cursor.All(ctx, &emails); err != nil {
		return nil, 0, err
	}

	return emails, total, nil
}

// DeleteAccountEmails deletes all emails for an account
func (s *MongoStore) DeleteAccountEmails(ctx context.Context, accountID primitive.ObjectID) error {
	_, err := s.db.Collection("emails").DeleteMany(ctx, bson.M{"account_id": accountID})
	return err
} 