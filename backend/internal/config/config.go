package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"email-harvester/internal/store"
)

// Config holds all configuration for the application
type Config struct {
	// Server configuration
	Port        int
	Environment string

	// Store configuration
	Store struct {
		Type string // "mongodb" or "cosmosdb"
	}
	MongoDB struct {
		URI      string
		Database string
	}
	CosmosDB struct {
		Endpoint string
		Key      string
		Database string
	}

	// OAuth configuration
	OAuth struct {
		Google struct {
			ClientID     string
			ClientSecret string
			RedirectURL  string
		}
		Outlook struct {
			ClientID     string
			ClientSecret string
			RedirectURL  string
		}
	}

	// Ollama configuration
	Ollama struct {
		APIURL string
		Model  string
	}

	// Monitoring configuration
	Monitoring struct {
		Enabled     bool
		ServiceName string
		Environment string
		OTLP struct {
			Endpoint string
			Insecure bool
		}
		Prometheus struct {
			Enabled bool
			Path    string
		}
		Logging struct {
			Level      string
			Format     string // "json" or "console"
			OutputPath string
		}
	}
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{}

	// Server configuration
	cfg.Port = getIntEnv("PORT", 8080)
	cfg.Environment = getEnv("ENV", "development")

	// Store configuration
	cfg.Store.Type = getEnv("STORE_TYPE", "mongodb")
	if cfg.Store.Type != "mongodb" && cfg.Store.Type != "cosmosdb" {
		return nil, fmt.Errorf("invalid store type: %s", cfg.Store.Type)
	}

	// MongoDB configuration
	cfg.MongoDB.URI = getEnv("MONGODB_URI", "mongodb://localhost:27017")
	cfg.MongoDB.Database = getEnv("MONGODB_DB", "email_harvester")

	// Cosmos DB configuration
	cfg.CosmosDB.Endpoint = getEnv("COSMOS_ENDPOINT", "")
	cfg.CosmosDB.Key = getEnv("COSMOS_KEY", "")
	cfg.CosmosDB.Database = getEnv("COSMOS_DB", "email_harvester")

	// OAuth configuration
	cfg.OAuth.Google.ClientID = getEnv("GOOGLE_CLIENT_ID", "")
	cfg.OAuth.Google.ClientSecret = getEnv("GOOGLE_CLIENT_SECRET", "")
	cfg.OAuth.Google.RedirectURL = getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/api/v1/accounts/auth/callback")

	cfg.OAuth.Outlook.ClientID = getEnv("OUTLOOK_CLIENT_ID", "")
	cfg.OAuth.Outlook.ClientSecret = getEnv("OUTLOOK_CLIENT_SECRET", "")
	cfg.OAuth.Outlook.RedirectURL = getEnv("OUTLOOK_REDIRECT_URL", "http://localhost:8080/api/v1/accounts/auth/callback")

	// Ollama configuration
	cfg.Ollama.APIURL = getEnv("OLLAMA_API_URL", "http://localhost:11434")
	cfg.Ollama.Model = getEnv("OLLAMA_MODEL", "llama2")

	// Monitoring configuration
	cfg.Monitoring.Enabled = getBoolEnv("MONITORING_ENABLED", true)
	cfg.Monitoring.ServiceName = getEnv("SERVICE_NAME", "email-harvester")
	cfg.Monitoring.Environment = getEnv("ENV", "development")

	cfg.Monitoring.OTLP.Endpoint = getEnv("OTLP_ENDPOINT", "localhost:4317")
	cfg.Monitoring.OTLP.Insecure = getBoolEnv("OTLP_INSECURE", true)

	cfg.Monitoring.Prometheus.Enabled = getBoolEnv("PROMETHEUS_ENABLED", true)
	cfg.Monitoring.Prometheus.Path = getEnv("PROMETHEUS_PATH", "/metrics")

	cfg.Monitoring.Logging.Level = getEnv("LOG_LEVEL", "info")
	cfg.Monitoring.Logging.Format = getEnv("LOG_FORMAT", "console")
	cfg.Monitoring.Logging.OutputPath = getEnv("LOG_OUTPUT_PATH", "stdout")

	// Validate required fields
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validate validates the configuration
func (c *Config) validate() error {
	// Validate store configuration
	switch c.Store.Type {
	case "mongodb":
		if c.MongoDB.URI == "" {
			return fmt.Errorf("MONGODB_URI is required for MongoDB store")
		}
	case "cosmosdb":
		if c.CosmosDB.Endpoint == "" {
			return fmt.Errorf("COSMOS_ENDPOINT is required for Cosmos DB store")
		}
		if c.CosmosDB.Key == "" {
			return fmt.Errorf("COSMOS_KEY is required for Cosmos DB store")
		}
	}

	// Validate OAuth configuration
	if c.OAuth.Google.ClientID == "" {
		return fmt.Errorf("GOOGLE_CLIENT_ID is required")
	}
	if c.OAuth.Google.ClientSecret == "" {
		return fmt.Errorf("GOOGLE_CLIENT_SECRET is required")
	}
	if c.OAuth.Outlook.ClientID == "" {
		return fmt.Errorf("OUTLOOK_CLIENT_ID is required")
	}
	if c.OAuth.Outlook.ClientSecret == "" {
		return fmt.Errorf("OUTLOOK_CLIENT_SECRET is required")
	}

	// Validate monitoring configuration
	if c.Monitoring.Enabled {
		if c.Monitoring.ServiceName == "" {
			return fmt.Errorf("SERVICE_NAME is required when monitoring is enabled")
		}
		if c.Monitoring.Environment == "" {
			return fmt.Errorf("ENV is required when monitoring is enabled")
		}
	}

	return nil
}

// Helper functions for environment variables
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
} 