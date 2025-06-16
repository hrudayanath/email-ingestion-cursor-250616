package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"email-harvester/internal/config"
	"email-harvester/internal/models"
	"email-harvester/internal/store"
)

// LLMService handles LLM operations for email analysis
type LLMService struct {
	store  *store.MongoStore
	config *config.OllamaConfig
	client *http.Client
}

// NewLLMService creates a new LLM service
func NewLLMService(cfg *config.OllamaConfig) *LLMService {
	return &LLMService{
		config: cfg,
		client: &http.Client{},
	}
}

// SetStore sets the store for the LLM service
func (s *LLMService) SetStore(store *store.MongoStore) {
	s.store = store
}

// SummarizeEmail generates a summary for a specific email
func (s *LLMService) SummarizeEmail(ctx context.Context, emailID primitive.ObjectID) (string, error) {
	// Get email
	email, err := s.store.GetEmail(ctx, emailID)
	if err != nil {
		return "", fmt.Errorf("failed to get email: %v", err)
	}
	if email == nil {
		return "", fmt.Errorf("email not found")
	}

	// Prepare prompt
	prompt := fmt.Sprintf(`Please summarize the following email in a concise and informative way:

Subject: %s
From: %s
To: %s

%s

Summary:`, email.Subject, email.From, strings.Join(email.To, ", "), email.Body)

	// Call Ollama API
	summary, err := s.callOllama(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate summary: %v", err)
	}

	// Update email with summary
	email.Summary = summary
	if err := s.store.UpdateEmail(ctx, email); err != nil {
		return "", fmt.Errorf("failed to update email: %v", err)
	}

	return summary, nil
}

// PerformNER performs Named Entity Recognition on a specific email
func (s *LLMService) PerformNER(ctx context.Context, emailID primitive.ObjectID) ([]models.NEREntity, error) {
	// Get email
	email, err := s.store.GetEmail(ctx, emailID)
	if err != nil {
		return nil, fmt.Errorf("failed to get email: %v", err)
	}
	if email == nil {
		return nil, fmt.Errorf("email not found")
	}

	// Prepare prompt
	prompt := fmt.Sprintf(`Please identify and extract named entities from the following email. For each entity, provide:
1. The entity text
2. The entity type (PERSON, ORGANIZATION, LOCATION, DATE, TIME, MONEY, PERCENT, etc.)
3. The start and end positions in the text
4. A confidence score between 0 and 1

Format the response as a JSON array of objects with the following structure:
[
  {
    "text": "entity text",
    "type": "entity type",
    "start_pos": start_position,
    "end_pos": end_position,
    "confidence": confidence_score
  }
]

Email:
%s

Entities:`, email.Body)

	// Call Ollama API
	response, err := s.callOllama(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to perform NER: %v", err)
	}

	// Parse response
	var entities []models.NEREntity
	if err := json.Unmarshal([]byte(response), &entities); err != nil {
		return nil, fmt.Errorf("failed to parse NER response: %v", err)
	}

	// Update email with entities
	email.Entities = entities
	if err := s.store.UpdateEmail(ctx, email); err != nil {
		return nil, fmt.Errorf("failed to update email: %v", err)
	}

	return entities, nil
}

// Helper functions

func (s *LLMService) callOllama(ctx context.Context, prompt string) (string, error) {
	// Prepare request
	reqBody := map[string]interface{}{
		"model":    s.config.Model,
		"prompt":   prompt,
		"stream":   false,
		"options": map[string]interface{}{
			"temperature": 0.7,
			"top_p":      0.9,
			"top_k":      40,
		},
	}

	reqBodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %v", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", s.config.APIURL+"/api/generate", bytes.NewBuffer(reqBodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Ollama API returned error: %s", resp.Status)
	}

	// Parse response
	var ollamaResp struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %v", err)
	}

	return ollamaResp.Response, nil
} 