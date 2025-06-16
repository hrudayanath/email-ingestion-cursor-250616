package store

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"email-harvester/internal/models"
)

// Store defines the interface for data storage operations
type Store interface {
	// Account operations
	CreateAccount(ctx context.Context, account *models.Account) error
	GetAccount(ctx context.Context, id primitive.ObjectID) (*models.Account, error)
	GetAccountByEmail(ctx context.Context, email string) (*models.Account, error)
	UpdateAccount(ctx context.Context, account *models.Account) error
	DeleteAccount(ctx context.Context, id primitive.ObjectID) error
	ListAccounts(ctx context.Context, page, limit int) ([]models.Account, int64, error)

	// Email operations
	CreateEmail(ctx context.Context, email *models.Email) error
	GetEmail(ctx context.Context, id primitive.ObjectID) (*models.Email, error)
	GetEmailByMessageID(ctx context.Context, accountID primitive.ObjectID, messageID string) (*models.Email, error)
	UpdateEmail(ctx context.Context, email *models.Email) error
	DeleteEmail(ctx context.Context, id primitive.ObjectID) error
	ListEmails(ctx context.Context, filter models.EmailFilter, page, limit int) ([]models.Email, int64, error)
	DeleteAccountEmails(ctx context.Context, accountID primitive.ObjectID) error
}

// StoreType represents the type of store to use
type StoreType string

const (
	StoreTypeMongoDB    StoreType = "mongodb"
	StoreTypeCosmosDB   StoreType = "cosmosdb"
)

// StoreConfig holds configuration for the store
type StoreConfig struct {
	Type StoreType
	// MongoDB specific config
	MongoURI      string
	MongoDatabase string
	// Cosmos DB specific config
	CosmosEndpoint string
	CosmosKey      string
	CosmosDatabase string
}

// NewStore creates a new store instance based on the configuration
func NewStore(cfg StoreConfig) (Store, error) {
	switch cfg.Type {
	case StoreTypeMongoDB:
		return NewMongoStore(cfg.MongoURI, cfg.MongoDatabase)
	case StoreTypeCosmosDB:
		return NewCosmosStore(cfg.CosmosEndpoint, cfg.CosmosKey, cfg.CosmosDatabase)
	default:
		return nil, fmt.Errorf("unsupported store type: %s", cfg.Type)
	}
} 