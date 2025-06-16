package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"email-harvester/internal/models"
	"email-harvester/internal/services"
)

type Handler struct {
	emailService *services.EmailService
	oauthService *services.OAuthService
	llmService   *services.LLMService
}

func NewHandler(
	emailService *services.EmailService,
	oauthService *services.OAuthService,
	llmService *services.LLMService,
) *Handler {
	return &Handler{
		emailService: emailService,
		oauthService: oauthService,
		llmService:   llmService,
	}
}

func (h *Handler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api")
	{
		// Health check
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status": "healthy",
				"time":   c.GetTime("request_time"),
			})
		})

		// Account routes
		accounts := api.Group("/accounts")
		{
			accounts.POST("", h.AddAccount)
			accounts.GET("/callback", h.OAuthCallback)
			accounts.DELETE("/:account_id", h.DeleteAccount)
			accounts.GET("/:account_id/emails", h.FetchEmails)
		}

		// Email routes
		emails := api.Group("/emails")
		{
			emails.GET("", h.ListEmails)
			emails.GET("/:id", h.GetEmail)
			emails.POST("/:id/summarize", h.SummarizeEmail)
			emails.POST("/:id/ner", h.PerformNER)
		}
	}
}

// AddAccount handles the addition of a new email account
func (h *Handler) AddAccount(c *gin.Context) {
	var req models.AddAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authURL, err := h.oauthService.GetAuthURL(req.Provider, req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"auth_url": authURL})
}

// OAuthCallback handles the OAuth callback from email providers
func (h *Handler) OAuthCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	if code == "" || state == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing code or state"})
		return
	}

	account, err := h.oauthService.HandleCallback(c.Request.Context(), code, state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, account)
}

// DeleteAccount handles the deletion of an email account
func (h *Handler) DeleteAccount(c *gin.Context) {
	accountID := c.Param("account_id")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing account_id"})
		return
	}

	id, err := primitive.ObjectIDFromHex(accountID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account_id"})
		return
	}

	if err := h.emailService.DeleteAccount(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// FetchEmails fetches emails for a specific account
func (h *Handler) FetchEmails(c *gin.Context) {
	accountID := c.Param("account_id")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing account_id"})
		return
	}

	id, err := primitive.ObjectIDFromHex(accountID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account_id"})
		return
	}

	emails, err := h.emailService.FetchEmails(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, emails)
}

// ListEmails lists all emails with pagination
func (h *Handler) ListEmails(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	accountID := c.Query("account_id")

	var filter models.EmailFilter
	if accountID != "" {
		id, err := primitive.ObjectIDFromHex(accountID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account_id"})
			return
		}
		filter.AccountID = &id
	}

	emails, total, err := h.emailService.ListEmails(c.Request.Context(), filter, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"emails": emails,
		"total":  total,
		"page":   page,
		"limit":  limit,
	})
}

// GetEmail retrieves a specific email by ID
func (h *Handler) GetEmail(c *gin.Context) {
	emailID := c.Param("id")
	if emailID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing email id"})
		return
	}

	id, err := primitive.ObjectIDFromHex(emailID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email id"})
		return
	}

	email, err := h.emailService.GetEmail(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, email)
}

// SummarizeEmail generates a summary for a specific email
func (h *Handler) SummarizeEmail(c *gin.Context) {
	emailID := c.Param("id")
	if emailID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing email id"})
		return
	}

	id, err := primitive.ObjectIDFromHex(emailID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email id"})
		return
	}

	summary, err := h.llmService.SummarizeEmail(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"summary": summary})
}

// PerformNER performs Named Entity Recognition on a specific email
func (h *Handler) PerformNER(c *gin.Context) {
	emailID := c.Param("id")
	if emailID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing email id"})
		return
	}

	id, err := primitive.ObjectIDFromHex(emailID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email id"})
		return
	}

	entities, err := h.llmService.PerformNER(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"entities": entities})
} 