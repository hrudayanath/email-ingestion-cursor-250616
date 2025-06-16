package middleware

import (
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"email-harvester/internal/monitoring"
)

// MonitoringMiddleware wraps an http.Handler with monitoring capabilities
func MonitoringMiddleware(monitor *monitoring.Monitor, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Extract trace context from request headers
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		// Create a new span for the request
		spanCtx, span := monitor.WithSpan(ctx, "http.request",
			trace.WithAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.url", r.URL.String()),
				attribute.String("http.user_agent", r.UserAgent()),
				attribute.String("http.remote_addr", r.RemoteAddr),
			),
		)
		defer span.End()

		// Create a response writer that captures the status code
		rw := newResponseWriter(w)

		// Add trace context to request
		r = r.WithContext(spanCtx)

		// Increment in-flight requests
		monitor.Metrics.HTTPRequestsInFlight.WithLabelValues(r.Method, r.URL.Path).Inc()
		defer monitor.Metrics.HTTPRequestsInFlight.WithLabelValues(r.Method, r.URL.Path).Dec()

		// Log request start
		monitor.LogDebug("Request started",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("remote_addr", r.RemoteAddr),
			zap.String("user_agent", r.UserAgent()),
		)

		// Call the next handler
		next.ServeHTTP(rw, r)

		// Calculate request duration
		duration := time.Since(start).Seconds()

		// Record metrics
		monitor.Metrics.HTTPRequestsTotal.WithLabelValues(
			r.Method,
			r.URL.Path,
			http.StatusText(rw.statusCode),
		).Inc()

		monitor.Metrics.HTTPRequestDuration.WithLabelValues(
			r.Method,
			r.URL.Path,
		).Observe(duration)

		// Set span attributes based on response
		span.SetAttributes(
			attribute.Int("http.status_code", rw.statusCode),
			attribute.Float64("http.duration_seconds", duration),
		)

		// Log request completion
		monitor.LogInfo("Request completed",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status_code", rw.statusCode),
			zap.Float64("duration_seconds", duration),
		)

		// Record error if status code indicates an error
		if rw.statusCode >= 400 {
			monitor.RecordError(span, http.ErrAbortHandler, trace.WithAttributes(
				attribute.Int("http.status_code", rw.statusCode),
			))
		}
	})
}

// responseWriter is a custom response writer that captures the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// newResponseWriter creates a new responseWriter
func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK}
}

// WriteHeader captures the status code and calls the underlying WriteHeader
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// ErrorMiddleware wraps an http.Handler with error handling and monitoring
func ErrorMiddleware(monitor *monitoring.Monitor, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic
				monitor.LogError("Request panic recovered",
					nil,
					zap.Any("error", err),
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
				)

				// Record the error in the span if available
				if span := trace.SpanFromContext(r.Context()); span.IsRecording() {
					monitor.RecordError(span, fmt.Errorf("panic: %v", err))
				}

				// Return 500 Internal Server Error
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// DebugMiddleware wraps an http.Handler with debug logging
func DebugMiddleware(monitor *monitoring.Monitor, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log request details
		monitor.LogDebug("Request details",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("query", r.URL.RawQuery),
			zap.String("remote_addr", r.RemoteAddr),
			zap.String("user_agent", r.UserAgent()),
			zap.Strings("headers", getHeaderValues(r)),
		)

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}

// getHeaderValues returns all header values as strings
func getHeaderValues(r *http.Request) []string {
	var values []string
	for name, headers := range r.Header {
		for _, value := range headers {
			values = append(values, fmt.Sprintf("%s: %s", name, value))
		}
	}
	return values
} 