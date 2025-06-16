package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/email-harvester/internal/models"
	"github.com/email-harvester/internal/store"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/microsoft"

	"email-harvester/internal/config"
)

// EmailService handles email operations for different providers
type EmailService struct {
	store        *store.MongoDBStore
	oauthService *OAuthService
	config       *config.OAuthConfig
}

// NewEmailService creates a new email service instance
func NewEmailService(store *store.MongoDBStore, oauthService *OAuthService) *EmailService {
	return &EmailService{
		store:        store,
		oauthService: oauthService,
		config:       oauthService.config,
	}
}

// FetchEmails fetches emails from the specified account and stores them in MongoDB
func (s *EmailService) FetchEmails(ctx context.Context, accountID string) error {
	// Get account from store
	objID, err := primitive.ObjectIDFromHex(accountID)
	if err != nil {
		return fmt.Errorf("invalid account ID: %v", err)
	}

	account, err := s.store.GetAccount(ctx, objID)
	if err != nil {
		return fmt.Errorf("failed to get account: %v", err)
	}

	// Get fresh token
	token := &oauth2.Token{
		AccessToken:  account.AccessToken,
		RefreshToken: account.RefreshToken,
		Expiry:       account.TokenExpiry,
	}

	token, err = s.oauthService.RefreshToken(ctx, account.Type, token)
	if err != nil {
		return fmt.Errorf("failed to refresh token: %v", err)
	}

	// Update account tokens
	if err := s.store.UpdateAccountTokens(ctx, account.ID, token.AccessToken, token.RefreshToken, token.Expiry); err != nil {
		return fmt.Errorf("failed to update account tokens: %v", err)
	}

	// Fetch emails based on account type
	switch account.Type {
	case models.AccountTypeGmail:
		return s.fetchGmailEmails(ctx, account, token)
	case models.AccountTypeOutlook:
		return s.fetchOutlookEmails(ctx, account, token)
	default:
		return fmt.Errorf("unsupported account type: %s", account.Type)
	}
}

// fetchGmailEmails fetches emails from Gmail
func (s *EmailService) fetchGmailEmails(ctx context.Context, account *models.Account, token *oauth2.Token) error {
	client := s.oauthService.getClient(ctx, account.Type, token)
	gmailService, err := gmail.New(client)
	if err != nil {
		return fmt.Errorf("failed to create Gmail service: %v", err)
	}

	// Get list of messages
	messages, err := gmailService.Users.Messages.List("me").Q("in:inbox").Do()
	if err != nil {
		return fmt.Errorf("failed to list messages: %v", err)
	}

	for _, msg := range messages.Messages {
		// Check if email already exists
		if _, err := s.store.GetEmailByMessageID(ctx, account.ID, msg.Id); err == nil {
			continue // Email already exists
		}

		// Get full message
		message, err := gmailService.Users.Messages.Get("me", msg.Id).Format("full").Do()
		if err != nil {
			return fmt.Errorf("failed to get message %s: %v", msg.Id, err)
		}

		// Parse message
		email, err := s.parseGmailMessage(message)
		if err != nil {
			return fmt.Errorf("failed to parse message %s: %v", msg.Id, err)
		}

		email.AccountID = account.ID
		email.MessageID = msg.Id

		// Store in MongoDB
		if err := s.store.CreateEmail(ctx, email); err != nil {
			return fmt.Errorf("failed to store message %s: %v", msg.Id, err)
		}
	}

	return nil
}

// fetchOutlookEmails fetches emails from Outlook
func (s *EmailService) fetchOutlookEmails(ctx context.Context, account *models.Account, token *oauth2.Token) error {
	client := s.oauthService.getClient(ctx, account.Type, token)

	// Get messages from Outlook Graph API
	resp, err := client.Get("https://graph.microsoft.com/v1.0/me/messages?$top=50&$select=id,subject,from,toRecipients,ccRecipients,bccRecipients,receivedDateTime,body")
	if err != nil {
		return fmt.Errorf("failed to get messages: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		Value []struct {
			ID               string    `json:"id"`
			Subject          string    `json:"subject"`
			From             struct{ EmailAddress struct{ Address string } } `json:"from"`
			ToRecipients     []struct{ EmailAddress struct{ Address string } } `json:"toRecipients"`
			CcRecipients     []struct{ EmailAddress struct{ Address string } } `json:"ccRecipients"`
			BccRecipients    []struct{ EmailAddress struct{ Address string } } `json:"bccRecipients"`
			ReceivedDateTime time.Time `json:"receivedDateTime"`
			Body             struct {
				Content     string `json:"content"`
				ContentType string `json:"contentType"`
			} `json:"body"`
		} `json:"value"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	for _, msg := range result.Value {
		// Check if email already exists
		if _, err := s.store.GetEmailByMessageID(ctx, account.ID, msg.ID); err == nil {
			continue // Email already exists
		}

		// Convert to our email model
		email := &models.Email{
			AccountID:  account.ID,
			MessageID:  msg.ID,
			From:       msg.From.EmailAddress.Address,
			Subject:    msg.Subject,
			ReceivedAt: msg.ReceivedDateTime,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		// Add recipients
		for _, to := range msg.ToRecipients {
			email.To = append(email.To, to.EmailAddress.Address)
		}
		for _, cc := range msg.CcRecipients {
			email.Cc = append(email.Cc, cc.EmailAddress.Address)
		}
		for _, bcc := range msg.BccRecipients {
			email.Bcc = append(email.Bcc, bcc.EmailAddress.Address)
		}

		// Set body based on content type
		if msg.Body.ContentType == "html" {
			email.HTMLBody = msg.Body.Content
		} else {
			email.Body = msg.Body.Content
		}

		// Store in MongoDB
		if err := s.store.CreateEmail(ctx, email); err != nil {
			return fmt.Errorf("failed to store message %s: %v", msg.ID, err)
		}
	}

	return nil
}

// parseGmailMessage parses a Gmail message into our email model
func (s *EmailService) parseGmailMessage(msg *gmail.Message) (*models.Email, error) {
	email := &models.Email{
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Parse headers
	headers := make(map[string]string)
	for _, header := range msg.Payload.Headers {
		headers[strings.ToLower(header.Name)] = header.Value
	}

	email.Subject = headers["subject"]
	email.From = headers["from"]
	email.To = strings.Split(headers["to"], ",")
	if cc := headers["cc"]; cc != "" {
		email.Cc = strings.Split(cc, ",")
	}
	if bcc := headers["bcc"]; bcc != "" {
		email.Bcc = strings.Split(bcc, ",")
	}

	// Parse received date
	if date := headers["date"]; date != "" {
		if t, err := time.Parse(time.RFC1123Z, date); err == nil {
			email.ReceivedAt = t
		}
	}

	// Parse body
	if err := s.parseGmailBody(msg.Payload, email); err != nil {
		return nil, err
	}

	return email, nil
}

// parseGmailBody parses the body of a Gmail message
func (s *EmailService) parseGmailBody(part *gmail.MessagePart, email *models.Email) error {
	if part.MimeType == "text/plain" {
		data, err := base64.URLEncoding.DecodeString(part.Body.Data)
		if err != nil {
			return err
		}
		email.Body = string(data)
	} else if part.MimeType == "text/html" {
		data, err := base64.URLEncoding.DecodeString(part.Body.Data)
		if err != nil {
			return err
		}
		email.HTMLBody = string(data)
	}

	// Process nested parts
	for _, p := range part.Parts {
		if err := s.parseGmailBody(p, email); err != nil {
			return err
		}
	}

	return nil
} 