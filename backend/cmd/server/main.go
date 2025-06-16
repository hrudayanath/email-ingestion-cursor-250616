package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"email-harvester/internal/api"
	"email-harvester/internal/config"
	"email-harvester/internal/middleware"
	"email-harvester/internal/monitoring"
	"email-harvester/internal/services"
	"email-harvester/internal/store"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize monitoring
	monitor, err := monitoring.NewMonitor("email-harvester", cfg.Environment)
	if err != nil {
		fmt.Printf("Failed to initialize monitoring: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := monitor.Close(context.Background()); err != nil {
			fmt.Printf("Failed to close monitoring: %v\n", err)
		}
	}()

	// Create context that listens for shutdown signals
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Initialize store
	var s store.Store
	switch cfg.Store.Type {
	case "mongodb":
		s, err = store.NewMongoDBStore(ctx, cfg.MongoDB, monitor)
	case "cosmosdb":
		s, err = store.NewCosmosDBStore(ctx, cfg.CosmosDB, monitor)
	default:
		monitor.LogError("Invalid store type", nil, zap.String("store_type", cfg.Store.Type))
		os.Exit(1)
	}
	if err != nil {
		monitor.LogError("Failed to initialize store", err)
		os.Exit(1)
	}
	defer s.Close(ctx)

	// Run migrations
	if err := s.RunMigrations(ctx); err != nil {
		monitor.LogError("Failed to run migrations", err)
		os.Exit(1)
	}

	// Initialize services
	oauthService := services.NewOAuthService(cfg.OAuth, monitor)
	emailService := services.NewEmailService(s, oauthService, monitor)
	llmService := services.NewLLMService(cfg.Ollama, monitor)

	// Initialize API handlers
	handlers := api.NewHandlers(emailService, llmService, monitor)

	// Create router
	router := mux.NewRouter()

	// Add monitoring middleware
	router.Use(middleware.MonitoringMiddleware(monitor))
	router.Use(middleware.ErrorMiddleware(monitor))
	if cfg.Environment != "production" {
		router.Use(middleware.DebugMiddleware(monitor))
	}

	// Health check endpoint
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods(http.MethodGet)

	// Metrics endpoint
	router.Handle("/metrics", promhttp.Handler()).Methods(http.MethodGet)

	// API routes
	apiRouter := router.PathPrefix("/api/v1").Subrouter()

	// Account routes
	accountsRouter := apiRouter.PathPrefix("/accounts").Subrouter()
	accountsRouter.HandleFunc("", handlers.AddAccount).Methods(http.MethodPost)
	accountsRouter.HandleFunc("", handlers.ListAccounts).Methods(http.MethodGet)
	accountsRouter.HandleFunc("/{id}", handlers.DeleteAccount).Methods(http.MethodDelete)
	accountsRouter.HandleFunc("/{id}/auth", handlers.GetAuthURL).Methods(http.MethodGet)
	accountsRouter.HandleFunc("/{id}/auth/callback", handlers.HandleAuthCallback).Methods(http.MethodGet)
	accountsRouter.HandleFunc("/{id}/emails/fetch", handlers.FetchEmails).Methods(http.MethodPost)

	// Email routes
	emailsRouter := apiRouter.PathPrefix("/emails").Subrouter()
	emailsRouter.HandleFunc("", handlers.ListEmails).Methods(http.MethodGet)
	emailsRouter.HandleFunc("/{id}", handlers.GetEmail).Methods(http.MethodGet)
	emailsRouter.HandleFunc("/{id}/summarize", handlers.SummarizeEmail).Methods(http.MethodPost)
	emailsRouter.HandleFunc("/{id}/ner", handlers.PerformNER).Methods(http.MethodPost)

	// Create server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		monitor.LogInfo("Starting server",
			zap.Int("port", cfg.Port),
			zap.String("environment", cfg.Environment),
		)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			monitor.LogError("Server error", err)
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	monitor.LogInfo("Shutting down server...")
	if err := server.Shutdown(shutdownCtx); err != nil {
		monitor.LogError("Server shutdown error", err)
		os.Exit(1)
	}

	monitor.LogInfo("Server stopped gracefully")
} 