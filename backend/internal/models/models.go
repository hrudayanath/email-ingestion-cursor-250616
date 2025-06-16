package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AccountType represents the type of email account
type AccountType string

const (
	AccountTypeGmail   AccountType = "gmail"
	AccountTypeOutlook AccountType = "outlook"
)

// Account represents an email account
type Account struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Provider     string            `bson:"provider" json:"provider"` // "gmail" or "outlook"
	Email        string            `bson:"email" json:"email"`
	AccessToken  string            `bson:"access_token" json:"-"`
	RefreshToken string            `bson:"refresh_token" json:"-"`
	TokenExpiry  time.Time         `bson:"token_expiry" json:"-"`
	CreatedAt    time.Time         `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time         `bson:"updated_at" json:"updated_at"`
}

// Email represents an email message
type Email struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	AccountID   primitive.ObjectID `bson:"account_id" json:"account_id"`
	MessageID   string            `bson:"message_id" json:"message_id"`
	ThreadID    string            `bson:"thread_id" json:"thread_id"`
	From        string            `bson:"from" json:"from"`
	To          []string          `bson:"to" json:"to"`
	Cc          []string          `bson:"cc" json:"cc"`
	Bcc         []string          `bson:"bcc" json:"bcc"`
	Subject     string            `bson:"subject" json:"subject"`
	Body        string            `bson:"body" json:"body"`
	HTMLBody    string            `bson:"html_body" json:"html_body"`
	Summary     string            `bson:"summary,omitempty" json:"summary,omitempty"`
	Entities    []NEREntity       `bson:"entities,omitempty" json:"entities,omitempty"`
	Labels      []string          `bson:"labels" json:"labels"`
	Read        bool              `bson:"read" json:"read"`
	Starred     bool              `bson:"starred" json:"starred"`
	ReceivedAt  time.Time         `bson:"received_at" json:"received_at"`
	CreatedAt   time.Time         `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time         `bson:"updated_at" json:"updated_at"`
}

// NEREntity represents a named entity extracted from an email
type NEREntity struct {
	Text      string `bson:"text" json:"text"`
	Type      string `bson:"type" json:"type"` // e.g., "PERSON", "ORG", "LOC", etc.
	StartPos  int    `bson:"start_pos" json:"start_pos"`
	EndPos    int    `bson:"end_pos" json:"end_pos"`
	Confidence float64 `bson:"confidence" json:"confidence"`
}

// AddAccountRequest represents the request to add a new email account
type AddAccountRequest struct {
	Provider string `json:"provider" binding:"required,oneof=gmail outlook"`
	Email    string `json:"email" binding:"required,email"`
}

// EmailFilter represents the filter criteria for listing emails
type EmailFilter struct {
	AccountID *primitive.ObjectID `json:"account_id,omitempty"`
	From      *string            `json:"from,omitempty"`
	To        *string            `json:"to,omitempty"`
	Subject   *string            `json:"subject,omitempty"`
	Label     *string            `json:"label,omitempty"`
	Read      *bool              `json:"read,omitempty"`
	Starred   *bool              `json:"starred,omitempty"`
	StartDate *time.Time         `json:"start_date,omitempty"`
	EndDate   *time.Time         `json:"end_date,omitempty"`
}

// EmailListResponse represents the paginated response for listing emails
type EmailListResponse struct {
	Emails []Email `json:"emails"`
	Total  int64   `json:"total"`
	Page   int     `json:"page"`
	Limit  int     `json:"limit"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code"`
} 