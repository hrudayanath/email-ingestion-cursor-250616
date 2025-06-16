package store

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"email-harvester/internal/models"
)

// CosmosStore implements the Store interface using Azure Cosmos DB
type CosmosStore struct {
	client     *azcosmos.Client
	database   *azcosmos.Database
	accounts   *azcosmos.Container
	emails     *azcosmos.Container
}

// NewCosmosStore creates a new Cosmos DB store instance
func NewCosmosStore(endpoint, key, databaseName string) (*CosmosStore, error) {
	cred, err := azcosmos.NewKeyCredential(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create credential: %w", err)
	}

	client, err := azcosmos.NewClientWithKey(endpoint, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	database, err := client.NewDatabase(databaseName)
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	// Create containers if they don't exist
	accounts, err := createContainerIfNotExists(database, "accounts", "/email")
	if err != nil {
		return nil, fmt.Errorf("failed to create accounts container: %w", err)
	}

	emails, err := createContainerIfNotExists(database, "emails", "/accountId")
	if err != nil {
		return nil, fmt.Errorf("failed to create emails container: %w", err)
	}

	return &CosmosStore{
		client:   client,
		database: database,
		accounts: accounts,
		emails:   emails,
	}, nil
}

func createContainerIfNotExists(db *azcosmos.Database, id string, partitionKey string) (*azcosmos.Container, error) {
	container, err := db.NewContainer(id)
	if err != nil {
		return nil, err
	}

	// Check if container exists
	_, err = container.Read(context.Background(), nil)
	if err == nil {
		return container, nil
	}

	// Create container if it doesn't exist
	properties := azcosmos.ContainerProperties{
		ID: id,
		PartitionKeyDefinition: azcosmos.PartitionKeyDefinition{
			Paths: []string{partitionKey},
		},
		IndexingPolicy: &azcosmos.IndexingPolicy{
			Automatic: true,
			IndexingMode: azcosmos.IndexingModeConsistent,
			IncludedPaths: []azcosmos.IncludedPath{
				{Path: "/*"},
			},
		},
	}

	_, err = db.CreateContainer(context.Background(), properties, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	return container, nil
}

// Account operations
func (s *CosmosStore) CreateAccount(ctx context.Context, account *models.Account) error {
	if account.ID.IsZero() {
		account.ID = primitive.NewObjectID()
	}
	account.CreatedAt = time.Now()
	account.UpdatedAt = account.CreatedAt

	_, err := s.accounts.CreateItem(ctx, azcosmos.NewPartitionKeyString(account.Email), account, nil)
	return err
}

func (s *CosmosStore) GetAccount(ctx context.Context, id primitive.ObjectID) (*models.Account, error) {
	var account models.Account
	_, err := s.accounts.ReadItem(ctx, azcosmos.NewPartitionKeyString(id.Hex()), id.Hex(), &account)
	if err != nil {
		return nil, err
	}
	return &account, nil
}

func (s *CosmosStore) GetAccountByEmail(ctx context.Context, email string) (*models.Account, error) {
	query := fmt.Sprintf("SELECT * FROM c WHERE c.email = @email")
	parameters := []azcosmos.QueryParameter{
		{Name: "@email", Value: email},
	}

	options := azcosmos.QueryOptions{
		QueryParameters: parameters,
	}

	pager := s.accounts.NewQueryItemsPager(query, azcosmos.NewPartitionKeyString(email), &options)
	var accounts []models.Account
	for pager.More() {
		response, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		var batch []models.Account
		err = response.Unmarshal(&batch)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, batch...)
	}

	if len(accounts) == 0 {
		return nil, fmt.Errorf("account not found")
	}
	return &accounts[0], nil
}

func (s *CosmosStore) UpdateAccount(ctx context.Context, account *models.Account) error {
	account.UpdatedAt = time.Now()
	_, err := s.accounts.ReplaceItem(ctx, azcosmos.NewPartitionKeyString(account.Email), account.ID.Hex(), account, nil)
	return err
}

func (s *CosmosStore) DeleteAccount(ctx context.Context, id primitive.ObjectID) error {
	account, err := s.GetAccount(ctx, id)
	if err != nil {
		return err
	}
	_, err = s.accounts.DeleteItem(ctx, azcosmos.NewPartitionKeyString(account.Email), id.Hex(), nil)
	return err
}

func (s *CosmosStore) ListAccounts(ctx context.Context, page, limit int) ([]models.Account, int64, error) {
	query := "SELECT * FROM c ORDER BY c.createdAt DESC OFFSET @offset LIMIT @limit"
	parameters := []azcosmos.QueryParameter{
		{Name: "@offset", Value: (page - 1) * limit},
		{Name: "@limit", Value: limit},
	}

	options := azcosmos.QueryOptions{
		QueryParameters: parameters,
	}

	pager := s.accounts.NewQueryItemsPager(query, nil, &options)
	var accounts []models.Account
	for pager.More() {
		response, err := pager.NextPage(ctx)
		if err != nil {
			return nil, 0, err
		}
		var batch []models.Account
		err = response.Unmarshal(&batch)
		if err != nil {
			return nil, 0, err
		}
		accounts = append(accounts, batch...)
	}

	// Get total count
	countQuery := "SELECT VALUE COUNT(1) FROM c"
	countPager := s.accounts.NewQueryItemsPager(countQuery, nil, nil)
	var total int64
	if countPager.More() {
		response, err := countPager.NextPage(ctx)
		if err != nil {
			return nil, 0, err
		}
		err = response.Unmarshal(&total)
		if err != nil {
			return nil, 0, err
		}
	}

	return accounts, total, nil
}

// Email operations
func (s *CosmosStore) CreateEmail(ctx context.Context, email *models.Email) error {
	if email.ID.IsZero() {
		email.ID = primitive.NewObjectID()
	}
	email.CreatedAt = time.Now()
	email.UpdatedAt = email.CreatedAt

	_, err := s.emails.CreateItem(ctx, azcosmos.NewPartitionKeyString(email.AccountID.Hex()), email, nil)
	return err
}

func (s *CosmosStore) GetEmail(ctx context.Context, id primitive.ObjectID) (*models.Email, error) {
	var email models.Email
	_, err := s.emails.ReadItem(ctx, azcosmos.NewPartitionKeyString(id.Hex()), id.Hex(), &email)
	if err != nil {
		return nil, err
	}
	return &email, nil
}

func (s *CosmosStore) GetEmailByMessageID(ctx context.Context, accountID primitive.ObjectID, messageID string) (*models.Email, error) {
	query := "SELECT * FROM c WHERE c.accountId = @accountId AND c.messageId = @messageId"
	parameters := []azcosmos.QueryParameter{
		{Name: "@accountId", Value: accountID.Hex()},
		{Name: "@messageId", Value: messageID},
	}

	options := azcosmos.QueryOptions{
		QueryParameters: parameters,
	}

	pager := s.emails.NewQueryItemsPager(query, azcosmos.NewPartitionKeyString(accountID.Hex()), &options)
	var emails []models.Email
	for pager.More() {
		response, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		var batch []models.Email
		err = response.Unmarshal(&batch)
		if err != nil {
			return nil, err
		}
		emails = append(emails, batch...)
	}

	if len(emails) == 0 {
		return nil, fmt.Errorf("email not found")
	}
	return &emails[0], nil
}

func (s *CosmosStore) UpdateEmail(ctx context.Context, email *models.Email) error {
	email.UpdatedAt = time.Now()
	_, err := s.emails.ReplaceItem(ctx, azcosmos.NewPartitionKeyString(email.AccountID.Hex()), email.ID.Hex(), email, nil)
	return err
}

func (s *CosmosStore) DeleteEmail(ctx context.Context, id primitive.ObjectID) error {
	email, err := s.GetEmail(ctx, id)
	if err != nil {
		return err
	}
	_, err = s.emails.DeleteItem(ctx, azcosmos.NewPartitionKeyString(email.AccountID.Hex()), id.Hex(), nil)
	return err
}

func (s *CosmosStore) ListEmails(ctx context.Context, filter models.EmailFilter, page, limit int) ([]models.Email, int64, error) {
	query := "SELECT * FROM c WHERE 1=1"
	parameters := []azcosmos.QueryParameter{}

	if !filter.AccountID.IsZero() {
		query += " AND c.accountId = @accountId"
		parameters = append(parameters, azcosmos.QueryParameter{Name: "@accountId", Value: filter.AccountID.Hex()})
	}
	if filter.From != "" {
		query += " AND c.from = @from"
		parameters = append(parameters, azcosmos.QueryParameter{Name: "@from", Value: filter.From})
	}
	if filter.To != "" {
		query += " AND c.to = @to"
		parameters = append(parameters, azcosmos.QueryParameter{Name: "@to", Value: filter.To})
	}
	if filter.Subject != "" {
		query += " AND CONTAINS(c.subject, @subject)"
		parameters = append(parameters, azcosmos.QueryParameter{Name: "@subject", Value: filter.Subject})
	}
	if filter.StartDate != nil {
		query += " AND c.date >= @startDate"
		parameters = append(parameters, azcosmos.QueryParameter{Name: "@startDate", Value: filter.StartDate})
	}
	if filter.EndDate != nil {
		query += " AND c.date <= @endDate"
		parameters = append(parameters, azcosmos.QueryParameter{Name: "@endDate", Value: filter.EndDate})
	}

	query += " ORDER BY c.date DESC OFFSET @offset LIMIT @limit"
	parameters = append(parameters,
		azcosmos.QueryParameter{Name: "@offset", Value: (page - 1) * limit},
		azcosmos.QueryParameter{Name: "@limit", Value: limit},
	)

	options := azcosmos.QueryOptions{
		QueryParameters: parameters,
	}

	partitionKey := azcosmos.NewPartitionKeyString(filter.AccountID.Hex())
	if filter.AccountID.IsZero() {
		partitionKey = nil
	}

	pager := s.emails.NewQueryItemsPager(query, partitionKey, &options)
	var emails []models.Email
	for pager.More() {
		response, err := pager.NextPage(ctx)
		if err != nil {
			return nil, 0, err
		}
		var batch []models.Email
		err = response.Unmarshal(&batch)
		if err != nil {
			return nil, 0, err
		}
		emails = append(emails, batch...)
	}

	// Get total count
	countQuery := "SELECT VALUE COUNT(1) FROM c WHERE 1=1"
	if !filter.AccountID.IsZero() {
		countQuery += " AND c.accountId = @accountId"
	}
	countPager := s.emails.NewQueryItemsPager(countQuery, partitionKey, &options)
	var total int64
	if countPager.More() {
		response, err := countPager.NextPage(ctx)
		if err != nil {
			return nil, 0, err
		}
		err = response.Unmarshal(&total)
		if err != nil {
			return nil, 0, err
		}
	}

	return emails, total, nil
}

func (s *CosmosStore) DeleteAccountEmails(ctx context.Context, accountID primitive.ObjectID) error {
	query := "SELECT c.id FROM c WHERE c.accountId = @accountId"
	parameters := []azcosmos.QueryParameter{
		{Name: "@accountId", Value: accountID.Hex()},
	}

	options := azcosmos.QueryOptions{
		QueryParameters: parameters,
	}

	pager := s.emails.NewQueryItemsPager(query, azcosmos.NewPartitionKeyString(accountID.Hex()), &options)
	for pager.More() {
		response, err := pager.NextPage(ctx)
		if err != nil {
			return err
		}
		var emails []struct{ ID string }
		err = response.Unmarshal(&emails)
		if err != nil {
			return err
		}
		for _, email := range emails {
			_, err = s.emails.DeleteItem(ctx, azcosmos.NewPartitionKeyString(accountID.Hex()), email.ID, nil)
			if err != nil {
				return err
			}
		}
	}
	return nil
} 