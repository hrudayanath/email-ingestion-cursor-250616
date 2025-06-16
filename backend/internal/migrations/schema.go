package migrations

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
)

// SchemaMigrator handles database schema migrations
type SchemaMigrator struct {
	store MigrationStore
}

// NewSchemaMigrator creates a new schema migrator
func NewSchemaMigrator(store MigrationStore) *SchemaMigrator {
	return &SchemaMigrator{
		store: store,
	}
}

// RunMongoDBSchema runs MongoDB schema migrations
func (m *SchemaMigrator) RunMongoDBSchema(ctx context.Context, db *mongo.Database) error {
	// Create accounts collection with indexes
	accountsCollection := db.Collection("accounts")
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "email", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "createdAt", Value: -1},
			},
		},
		{
			Keys: bson.D{
				{Key: "updatedAt", Value: -1},
			},
		},
	}

	if _, err := accountsCollection.Indexes().CreateMany(ctx, indexes); err != nil {
		return fmt.Errorf("failed to create accounts indexes: %w", err)
	}

	// Create emails collection with indexes
	emailsCollection := db.Collection("emails")
	indexes = []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "accountId", Value: 1},
				{Key: "messageId", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "accountId", Value: 1},
				{Key: "date", Value: -1},
			},
		},
		{
			Keys: bson.D{
				{Key: "from", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "to", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "subject", Value: "text"},
			},
		},
		{
			Keys: bson.D{
				{Key: "createdAt", Value: -1},
			},
		},
		{
			Keys: bson.D{
				{Key: "updatedAt", Value: -1},
			},
		},
	}

	if _, err := emailsCollection.Indexes().CreateMany(ctx, indexes); err != nil {
		return fmt.Errorf("failed to create emails indexes: %w", err)
	}

	// Create migrations collection with indexes
	migrationsCollection := db.Collection("migrations")
	indexes = []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "version", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
	}

	if _, err := migrationsCollection.Indexes().CreateMany(ctx, indexes); err != nil {
		return fmt.Errorf("failed to create migrations indexes: %w", err)
	}

	return nil
}

// RunCosmosDBSchema runs Cosmos DB schema migrations
func (m *SchemaMigrator) RunCosmosDBSchema(ctx context.Context, database *azcosmos.Database) error {
	// Create accounts container
	accountsProperties := azcosmos.ContainerProperties{
		ID: "accounts",
		PartitionKeyDefinition: azcosmos.PartitionKeyDefinition{
			Paths: []string{"/email"},
		},
		IndexingPolicy: &azcosmos.IndexingPolicy{
			Automatic: true,
			IndexingMode: azcosmos.IndexingModeConsistent,
			IncludedPaths: []azcosmos.IncludedPath{
				{Path: "/email/?"},
				{Path: "/createdAt/?"},
				{Path: "/updatedAt/?"},
			},
			ExcludedPaths: []azcosmos.ExcludedPath{
				{Path: "/*"},
			},
		},
	}

	if _, err := database.CreateContainer(ctx, accountsProperties, nil); err != nil {
		// Ignore if container already exists
		if !isContainerExistsError(err) {
			return fmt.Errorf("failed to create accounts container: %w", err)
		}
	}

	// Create emails container
	emailsProperties := azcosmos.ContainerProperties{
		ID: "emails",
		PartitionKeyDefinition: azcosmos.PartitionKeyDefinition{
			Paths: []string{"/accountId"},
		},
		IndexingPolicy: &azcosmos.IndexingPolicy{
			Automatic: true,
			IndexingMode: azcosmos.IndexingModeConsistent,
			IncludedPaths: []azcosmos.IncludedPath{
				{Path: "/accountId/?"},
				{Path: "/messageId/?"},
				{Path: "/date/?"},
				{Path: "/from/?"},
				{Path: "/to/?"},
				{Path: "/subject/?"},
				{Path: "/createdAt/?"},
				{Path: "/updatedAt/?"},
			},
			ExcludedPaths: []azcosmos.ExcludedPath{
				{Path: "/*"},
			},
		},
		UniqueKeyPolicy: &azcosmos.UniqueKeyPolicy{
			UniqueKeys: []azcosmos.UniqueKey{
				{
					Paths: []string{"/accountId", "/messageId"},
				},
			},
		},
	}

	if _, err := database.CreateContainer(ctx, emailsProperties, nil); err != nil {
		// Ignore if container already exists
		if !isContainerExistsError(err) {
			return fmt.Errorf("failed to create emails container: %w", err)
		}
	}

	// Create migrations container
	migrationsProperties := azcosmos.ContainerProperties{
		ID: "migrations",
		PartitionKeyDefinition: azcosmos.PartitionKeyDefinition{
			Paths: []string{"/version"},
		},
		IndexingPolicy: &azcosmos.IndexingPolicy{
			Automatic: true,
			IndexingMode: azcosmos.IndexingModeConsistent,
			IncludedPaths: []azcosmos.IncludedPath{
				{Path: "/version/?"},
				{Path: "/createdAt/?"},
			},
			ExcludedPaths: []azcosmos.ExcludedPath{
				{Path: "/*"},
			},
		},
	}

	if _, err := database.CreateContainer(ctx, migrationsProperties, nil); err != nil {
		// Ignore if container already exists
		if !isContainerExistsError(err) {
			return fmt.Errorf("failed to create migrations container: %w", err)
		}
	}

	return nil
}

// isContainerExistsError checks if the error is due to container already existing
func isContainerExistsError(err error) bool {
	if err == nil {
		return false
	}
	// Cosmos DB returns a specific error code when container already exists
	return err.Error() == "Container already exists"
} 