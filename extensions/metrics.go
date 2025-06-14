package extensions

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics is a struct that holds various Prometheus metrics for HTTP server monitoring.
type Metrics struct {
	RequestCounter     *prometheus.CounterVec
	RequestDuration    *prometheus.HistogramVec
	RequestSize        *prometheus.HistogramVec
	ActiveConnections  prometheus.Gauge
	ConcurrentRequests prometheus.Gauge
	ErrorCounter       *prometheus.CounterVec
	ServerUptime       prometheus.Counter
}

func NewMetrics() *Metrics {
	return &Metrics{
		RequestCounter: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "The total number of processed events",
		}, []string{"path", "method", "status"}),
		RequestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "The duration of the request",
		}, []string{"path", "method", "status"}),
		RequestSize: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_request_size_bytes",
			Help:    "The size of HTTP requests in bytes",
			Buckets: []float64{100, 1000, 10000, 100000, 1000000}, // 100B, 1KB, 10KB, 100KB, 1MB
		}, []string{"path", "method"}),
		ActiveConnections: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "http_active_connections",
			Help: "Number of active HTTP connections",
		}),
		ConcurrentRequests: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "http_concurrent_requests",
			Help: "Number of concurrent HTTP requests being processed",
		}),
		ErrorCounter: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_errors_total",
			Help: "Total number of HTTP errors",
		}, []string{"path", "method", "status_code", "error_type"}),
		ServerUptime: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "http_server_uptime_seconds",
			Help: "Total uptime of the HTTP server in seconds",
		}),
	}
}

// Register registers the metrics with the provided HTTP ServeMux.
func (m *Metrics) Register(mux *http.ServeMux) error {
	if mux == nil {
		return errors.New("mux cannot be nil")
	}

	mux.Handle("/metrics", promhttp.Handler())

	prometheus.MustRegister(
		m.RequestCounter,
		m.RequestDuration,
		m.RequestSize,
		m.ActiveConnections,
		m.ConcurrentRequests,
		m.ErrorCounter,
		m.ServerUptime,
	)

	return nil
}

// Unregister unregisters the metrics from Prometheus.
func (m *Metrics) Unregister() {
	prometheus.Unregister(m.RequestCounter)
	prometheus.Unregister(m.RequestDuration)
	prometheus.Unregister(m.RequestSize)
	prometheus.Unregister(m.ActiveConnections)
	prometheus.Unregister(m.ConcurrentRequests)
	prometheus.Unregister(m.ErrorCounter)
	prometheus.Unregister(m.ServerUptime)
}

// StartUptimeTracking starts a goroutine that increments the server uptime metric every second.
func (m *Metrics) StartUptimeTracking() {
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			m.ServerUptime.Inc()
		}
	}()
}

// Middleware is an HTTP middleware that collects metrics for incoming requests.
func (m *Metrics) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		m.ConcurrentRequests.Inc()
		defer m.ConcurrentRequests.Dec()

		requestSize := float64(r.ContentLength)
		if requestSize > 0 {
			m.RequestSize.WithLabelValues(r.URL.Path, r.Method).Observe(requestSize)
		}

		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     200,
		}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start).Seconds()
		statusCode := wrapped.statusCode
		statusText := http.StatusText(statusCode)

		m.RequestCounter.WithLabelValues(r.URL.Path, r.Method, statusText).Inc()

		m.RequestDuration.WithLabelValues(r.URL.Path, r.Method, statusText).Observe(duration)

		if statusCode >= 400 {
			errorType := "client_error"
			if statusCode >= 500 {
				errorType = "server_error"
			}
			m.ErrorCounter.WithLabelValues(r.URL.Path, r.Method, fmt.Sprintf("%d", statusCode), errorType).Inc()
		}
	})
}
