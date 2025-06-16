package migrations

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
)

// Migration represents a database migration
type Migration struct {
	Version   int
	Name      string
	CreatedAt time.Time
}

// MigrationStore defines the interface for migration operations
type MigrationStore interface {
	// GetMigrations returns all applied migrations
	GetMigrations(ctx context.Context) ([]Migration, error)
	// CreateMigration records a new migration
	CreateMigration(ctx context.Context, migration Migration) error
}

// MongoDBMigrationStore implements MigrationStore for MongoDB
type MongoDBMigrationStore struct {
	collection *mongo.Collection
}

// NewMongoDBMigrationStore creates a new MongoDB migration store
func NewMongoDBMigrationStore(db *mongo.Database) *MongoDBMigrationStore {
	return &MongoDBMigrationStore{
		collection: db.Collection("migrations"),
	}
}

// GetMigrations returns all applied migrations from MongoDB
func (s *MongoDBMigrationStore) GetMigrations(ctx context.Context) ([]Migration, error) {
	cursor, err := s.collection.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "version", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to get migrations: %w", err)
	}
	defer cursor.Close(ctx)

	var migrations []Migration
	if err := cursor.All(ctx, &migrations); err != nil {
		return nil, fmt.Errorf("failed to decode migrations: %w", err)
	}
	return migrations, nil
}

// CreateMigration records a new migration in MongoDB
func (s *MongoDBMigrationStore) CreateMigration(ctx context.Context, migration Migration) error {
	_, err := s.collection.InsertOne(ctx, migration)
	if err != nil {
		return fmt.Errorf("failed to create migration: %w", err)
	}
	return nil
}

// CosmosDBMigrationStore implements MigrationStore for Cosmos DB
type CosmosDBMigrationStore struct {
	container *azcosmos.Container
}

// NewCosmosDBMigrationStore creates a new Cosmos DB migration store
func NewCosmosDBMigrationStore(container *azcosmos.Container) *CosmosDBMigrationStore {
	return &CosmosDBMigrationStore{
		container: container,
	}
}

// GetMigrations returns all applied migrations from Cosmos DB
func (s *CosmosDBMigrationStore) GetMigrations(ctx context.Context) ([]Migration, error) {
	query := "SELECT * FROM c ORDER BY c.version ASC"
	pager := s.container.NewQueryItemsPager(query, nil, nil)

	var migrations []Migration
	for pager.More() {
		response, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get migrations: %w", err)
		}
		var batch []Migration
		if err := response.Unmarshal(&batch); err != nil {
			return nil, fmt.Errorf("failed to decode migrations: %w", err)
		}
		migrations = append(migrations, batch...)
	}
	return migrations, nil
}

// CreateMigration records a new migration in Cosmos DB
func (s *CosmosDBMigrationStore) CreateMigration(ctx context.Context, migration Migration) error {
	_, err := s.container.CreateItem(ctx, azcosmos.NewPartitionKeyString(fmt.Sprintf("%d", migration.Version)), migration, nil)
	if err != nil {
		return fmt.Errorf("failed to create migration: %w", err)
	}
	return nil
}

// Migrator handles database migrations
type Migrator struct {
	store     MigrationStore
	migrations []Migration
}

// NewMigrator creates a new migrator
func NewMigrator(store MigrationStore) *Migrator {
	return &Migrator{
		store: store,
		migrations: []Migration{
			{
				Version:   1,
				Name:      "initial_schema",
				CreatedAt: time.Now(),
			},
		},
	}
}

// Run executes all pending migrations
func (m *Migrator) Run(ctx context.Context) error {
	// Get applied migrations
	applied, err := m.store.GetMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Create a map of applied versions for quick lookup
	appliedVersions := make(map[int]bool)
	for _, migration := range applied {
		appliedVersions[migration.Version] = true
	}

	// Run pending migrations
	for _, migration := range m.migrations {
		if !appliedVersions[migration.Version] {
			if err := m.runMigration(ctx, migration); err != nil {
				return fmt.Errorf("failed to run migration %d: %w", migration.Version, err)
			}
		}
	}

	return nil
}

// runMigration executes a single migration
func (m *Migrator) runMigration(ctx context.Context, migration Migration) error {
	// Record migration before running to ensure idempotency
	if err := m.store.CreateMigration(ctx, migration); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return nil
} 