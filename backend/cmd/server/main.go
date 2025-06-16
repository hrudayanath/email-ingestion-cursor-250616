package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"go.uber.org/zap"

	"email-harvester/internal/config"
	"email-harvester/internal/handlers"
	"email-harvester/internal/middleware/middleware"
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
	monitor, err := monitoring.NewMonitor(cfg.Monitoring)
	if err != nil {
		fmt.Printf("Failed to initialize monitoring: %v\n", err)
		os.Exit(1)
	}
	defer monitor.Shutdown()

	// Initialize store
	var store store.Store
	switch cfg.Store.Type {
	case "mongodb":
		store, err = store.NewMongoStore(cfg.MongoDB, monitor)
	case "cosmos":
		store, err = store.NewCosmosStore(cfg.CosmosDB, monitor)
	default:
		err = fmt.Errorf("unsupported store type: %s", cfg.Store.Type)
	}
	if err != nil {
		monitor.LogFatal("Failed to initialize store", err)
	}
	defer store.Close()

	// Initialize services
	oauthService, err := services.NewOAuthService(cfg, monitor)
	if err != nil {
		monitor.LogFatal("Failed to initialize OAuth service", err)
	}

	emailService := services.NewEmailService(store, monitor)
	llmService := services.NewLLMService(cfg.Ollama, monitor)

	// Initialize handlers
	oauthHandler := handlers.NewOAuthHandler(oauthService, monitor)
	emailHandler := handlers.NewEmailHandler(emailService, monitor)
	accountHandler := handlers.NewAccountHandler(store, monitor)

	// Create router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(middleware.CleanPath)
	r.Use(middleware.GetHead)
	r.Use(middleware.NoCache)
	r.Use(middleware.StripSlashes)
	r.Use(middleware.WithValue("monitor", monitor))
	r.Use(middleware.WithValue("config", cfg))

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Monitoring middleware
	r.Use(middleware.MonitoringMiddleware(monitor))

	// Routes
	r.Route("/api", func(r chi.Router) {
		// OAuth routes
		oauthHandler.RegisterRoutes(r)

		// Account routes
		accountHandler.RegisterRoutes(r)

		// Email routes
		emailHandler.RegisterRoutes(r)

		// Health check
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			render.JSON(w, r, map[string]string{"status": "ok"})
		})
	})

	// Create server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server
	go func() {
		monitor.LogInfo("Starting server",
			zap.Int("port", cfg.Server.Port),
			zap.String("env", cfg.Server.Env),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			monitor.LogFatal("Failed to start server", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Graceful shutdown
	monitor.LogInfo("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		monitor.LogError("Server forced to shutdown", err)
	}

	monitor.LogInfo("Server stopped")
} 