package monitoring

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics
type Metrics struct {
	// HTTP metrics
	HTTPRequestsTotal    *prometheus.CounterVec
	HTTPRequestDuration  *prometheus.HistogramVec
	HTTPRequestsInFlight *prometheus.GaugeVec

	// Store metrics
	StoreOperationsTotal    *prometheus.CounterVec
	StoreOperationDuration  *prometheus.HistogramVec
	StoreConnectionsActive  *prometheus.GaugeVec
	StoreErrorsTotal        *prometheus.CounterVec

	// Email metrics
	EmailFetchTotal        *prometheus.CounterVec
	EmailFetchDuration     *prometheus.HistogramVec
	EmailProcessTotal      *prometheus.CounterVec
	EmailProcessDuration   *prometheus.HistogramVec
	EmailErrorsTotal       *prometheus.CounterVec

	// OAuth metrics
	OAuthRequestsTotal     *prometheus.CounterVec
	OAuthRequestDuration   *prometheus.HistogramVec
	OAuthErrorsTotal       *prometheus.CounterVec

	// LLM metrics
	LLMRequestsTotal       *prometheus.CounterVec
	LLMRequestDuration     *prometheus.HistogramVec
	LLMErrorsTotal         *prometheus.CounterVec
	LLMTokensTotal         *prometheus.CounterVec
}

// Monitor holds all monitoring components
type Monitor struct {
	Logger  *zap.Logger
	Metrics *Metrics
	Tracer  trace.Tracer
}

// NewMonitor creates a new monitor instance
func NewMonitor(serviceName, environment string) (*Monitor, error) {
	// Initialize logger
	logger, err := newLogger(environment)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	// Initialize metrics
	metrics := newMetrics(serviceName)

	// Initialize tracer
	tracer, err := newTracer(serviceName, environment)
	if err != nil {
		return nil, fmt.Errorf("failed to create tracer: %w", err)
	}

	return &Monitor{
		Logger:  logger,
		Metrics: metrics,
		Tracer:  tracer,
	}, nil
}

// newLogger creates a new structured logger
func newLogger(environment string) (*zap.Logger, error) {
	var config zap.Config

	if environment == "production" {
		config = zap.NewProductionConfig()
	} else {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeDuration = zapcore.StringDurationEncoder

	return config.Build()
}

// newMetrics creates all Prometheus metrics
func newMetrics(serviceName string) *Metrics {
	commonLabels := prometheus.Labels{"service": serviceName}

	return &Metrics{
		// HTTP metrics
		HTTPRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "http_requests_total",
				Help:        "Total number of HTTP requests",
				ConstLabels: commonLabels,
			},
			[]string{"method", "path", "status"},
		),
		HTTPRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:        "http_request_duration_seconds",
				Help:        "HTTP request duration in seconds",
				ConstLabels: commonLabels,
				Buckets:     prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		HTTPRequestsInFlight: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "http_requests_in_flight",
				Help:        "Current number of HTTP requests being served",
				ConstLabels: commonLabels,
			},
			[]string{"method", "path"},
		),

		// Store metrics
		StoreOperationsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "store_operations_total",
				Help:        "Total number of store operations",
				ConstLabels: commonLabels,
			},
			[]string{"operation", "collection", "status"},
		),
		StoreOperationDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:        "store_operation_duration_seconds",
				Help:        "Store operation duration in seconds",
				ConstLabels: commonLabels,
				Buckets:     prometheus.DefBuckets,
			},
			[]string{"operation", "collection"},
		),
		StoreConnectionsActive: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "store_connections_active",
				Help:        "Number of active store connections",
				ConstLabels: commonLabels,
			},
			[]string{"store_type"},
		),
		StoreErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "store_errors_total",
				Help:        "Total number of store errors",
				ConstLabels: commonLabels,
			},
			[]string{"operation", "collection", "error_type"},
		),

		// Email metrics
		EmailFetchTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "email_fetch_total",
				Help:        "Total number of email fetches",
				ConstLabels: commonLabels,
			},
			[]string{"provider", "status"},
		),
		EmailFetchDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:        "email_fetch_duration_seconds",
				Help:        "Email fetch duration in seconds",
				ConstLabels: commonLabels,
				Buckets:     prometheus.DefBuckets,
			},
			[]string{"provider"},
		),
		EmailProcessTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "email_process_total",
				Help:        "Total number of email processing operations",
				ConstLabels: commonLabels,
			},
			[]string{"operation", "status"},
		),
		EmailProcessDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:        "email_process_duration_seconds",
				Help:        "Email processing duration in seconds",
				ConstLabels: commonLabels,
				Buckets:     prometheus.DefBuckets,
			},
			[]string{"operation"},
		),
		EmailErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "email_errors_total",
				Help:        "Total number of email-related errors",
				ConstLabels: commonLabels,
			},
			[]string{"operation", "error_type"},
		),

		// OAuth metrics
		OAuthRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "oauth_requests_total",
				Help:        "Total number of OAuth requests",
				ConstLabels: commonLabels,
			},
			[]string{"provider", "operation", "status"},
		),
		OAuthRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:        "oauth_request_duration_seconds",
				Help:        "OAuth request duration in seconds",
				ConstLabels: commonLabels,
				Buckets:     prometheus.DefBuckets,
			},
			[]string{"provider", "operation"},
		),
		OAuthErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "oauth_errors_total",
				Help:        "Total number of OAuth errors",
				ConstLabels: commonLabels,
			},
			[]string{"provider", "operation", "error_type"},
		),

		// LLM metrics
		LLMRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "llm_requests_total",
				Help:        "Total number of LLM requests",
				ConstLabels: commonLabels,
			},
			[]string{"operation", "model", "status"},
		),
		LLMRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:        "llm_request_duration_seconds",
				Help:        "LLM request duration in seconds",
				ConstLabels: commonLabels,
				Buckets:     prometheus.DefBuckets,
			},
			[]string{"operation", "model"},
		),
		LLMErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "llm_errors_total",
				Help:        "Total number of LLM errors",
				ConstLabels: commonLabels,
			},
			[]string{"operation", "model", "error_type"},
		),
		LLMTokensTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "llm_tokens_total",
				Help:        "Total number of tokens processed",
				ConstLabels: commonLabels,
			},
			[]string{"operation", "model", "token_type"},
		),
	}
}

// newTracer creates a new OpenTelemetry tracer
func newTracer(serviceName, environment string) (trace.Tracer, error) {
	ctx := context.Background()

	// Create OTLP exporter
	exporter, err := otlptrace.New(
		ctx,
		otlptracegrpc.NewClient(
			otlptracegrpc.WithInsecure(),
			otlptracegrpc.WithEndpoint("localhost:4317"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create resource with service information
	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion("1.0.0"),
			attribute.String("environment", environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	// Set global trace provider
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Tracer(serviceName), nil
}

// WithSpan creates a new span and returns a context with the span
func (m *Monitor) WithSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return m.Tracer.Start(ctx, name, opts...)
}

// RecordError records an error with the current span
func (m *Monitor) RecordError(span trace.Span, err error, opts ...trace.EventOption) {
	span.RecordError(err, opts...)
	span.SetStatus(codes.Error, err.Error())
}

// LogError logs an error with structured logging
func (m *Monitor) LogError(msg string, err error, fields ...zap.Field) {
	fields = append(fields, zap.Error(err))
	m.Logger.Error(msg, fields...)
}

// LogInfo logs an info message with structured logging
func (m *Monitor) LogInfo(msg string, fields ...zap.Field) {
	m.Logger.Info(msg, fields...)
}

// LogDebug logs a debug message with structured logging
func (m *Monitor) LogDebug(msg string, fields ...zap.Field) {
	m.Logger.Debug(msg, fields...)
}

// Close closes all monitoring components
func (m *Monitor) Close(ctx context.Context) error {
	// Flush logger
	if err := m.Logger.Sync(); err != nil {
		return fmt.Errorf("failed to sync logger: %w", err)
	}

	// Shutdown tracer provider
	if tp, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider); ok {
		if err := tp.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown tracer provider: %w", err)
		}
	}

	return nil
} 