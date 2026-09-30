package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests",
		},
		[]string{"method", "route", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Request latency",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route", "status"},
	)
)

func InitMetrics() {
	prometheus.MustRegister(httpRequestsTotal, httpRequestDuration)
}

func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start).Seconds()
		status := c.Writer.Status()
		route := c.FullPath()

		if route == "" {
			route = "unknown"
		}

		statusCode := strconv.Itoa(status)

		httpRequestsTotal.WithLabelValues(
			c.Request.Method,
			route,
			statusCode,
		).Inc()

		httpRequestDuration.WithLabelValues(
			c.Request.Method,
			route,
			statusCode,
		).Observe(duration)
	}
}
