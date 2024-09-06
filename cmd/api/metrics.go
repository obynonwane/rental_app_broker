package main

import (
	"net/http"
	"runtime"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Define Prometheus metrics

// HttpRequestsTotal counts the total number of HTTP requests received by the server.
var HttpRequestsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests",
	},
	[]string{"path", "method"},
)

// HttpRequestDuration measures the duration of HTTP requests in seconds.
var HttpRequestDuration = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Duration of HTTP requests in seconds",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"path", "method"},
)

// ErrorCounter counts the total number of HTTP errors returned by the server.
var ErrorCounter = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_errors_total",
		Help: "Total number of HTTP errors",
	},
	[]string{"path", "method", "status"},
)

// FunctionCallCounter counts the total number of calls to specific functions.
var FunctionCallCounter = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "function_calls_total",
		Help: "Total number of function calls",
	},
	[]string{"function_name"},
)

// FunctionLatency measures the duration of specific function calls in seconds.
var FunctionLatency = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "function_latency_seconds",
		Help:    "Latency of function calls in seconds",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"function_name"},
)

// CPUUsageGauge tracks the current CPU usage as a percentage.
var CPUUsageGauge = promauto.NewGauge(
	prometheus.GaugeOpts{
		Name: "cpu_usage_percent",
		Help: "CPU usage in percent",
	},
)

// MemoryUsageGauge tracks the current memory usage in bytes.
var MemoryUsageGauge = promauto.NewGauge(
	prometheus.GaugeOpts{
		Name: "memory_usage_bytes",
		Help: "Memory usage in bytes",
	},
)

// GoroutinesGauge tracks the number of goroutines currently running.
var GoroutinesGauge = promauto.NewGauge(
	prometheus.GaugeOpts{
		Name: "goroutines_count",
		Help: "Number of goroutines currently running",
	},
)

// GCCountGauge tracks the number of garbage collection cycles completed.
var GCCountGauge = promauto.NewCounter(
	prometheus.CounterOpts{
		Name: "gc_cycles_total",
		Help: "Total number of completed garbage collection cycles",
	},
)

// CollectSystemMetrics periodically collects system-level metrics like CPU and memory usage.
func CollectSystemMetrics() {
	for {
		// Update CPU and memory usage metrics
		CPUUsageGauge.Set(getCPUUsage())
		m := &runtime.MemStats{}
		runtime.ReadMemStats(m)
		MemoryUsageGauge.Set(float64(m.Alloc))

		// Update goroutine count
		GoroutinesGauge.Set(float64(runtime.NumGoroutine()))

		// Update garbage collection count
		GCCountGauge.Add(float64(m.NumGC))

		time.Sleep(10 * time.Second) // Collect metrics every 10 seconds
	}
}

// getCPUUsage simulates CPU usage (this should be replaced with actual CPU usage monitoring).
func getCPUUsage() float64 {
	// Placeholder for CPU usage; replace with actual logic if needed.
	return float64(runtime.NumCPU())
}

// PrometheusMiddleware is an HTTP middleware that tracks HTTP requests and their durations.
func PrometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		defer func() {
			// Calculate request duration and update Prometheus metrics
			duration := time.Since(start).Seconds()
			HttpRequestDuration.WithLabelValues(r.URL.Path, r.Method).Observe(duration)
			HttpRequestsTotal.WithLabelValues(r.URL.Path, r.Method).Inc()

			// Increment error counter if response status indicates an error
			if rw.Status() >= 400 {
				ErrorCounter.WithLabelValues(r.URL.Path, r.Method, http.StatusText(rw.Status())).Inc()
			}
		}()

		// Call the next handler in the chain
		next.ServeHTTP(rw, r)
	})
}

// TrackFunctionCall tracks the call to a specific function and its latency.
func TrackFunctionCall(functionName string, fn func()) {
	start := time.Now()

	// Increment the function call counter
	FunctionCallCounter.WithLabelValues(functionName).Inc()

	// Execute the function
	fn()

	// Measure and record the latency of the function call
	duration := time.Since(start).Seconds()
	FunctionLatency.WithLabelValues(functionName).Observe(duration)
}
