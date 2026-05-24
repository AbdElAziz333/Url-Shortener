package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"gorm.io/gorm"
)

var (
	HttpRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "gateway_http_requests_total",
		Help: "Total number of HTTP requests processed by the gateway service.",
	}, []string{"method", "path", "status"})

	HttpRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "gateway_http_request_duration_seconds",
		Help:    "Latency of HTTP requests processed by the gateway service in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path", "status"})

	HttpRequestsInFlight = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gateway_http_requests_in_flight",
		Help: "Current number of HTTP requests being processed.",
	})

	HttpRequestSize = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "gateway_http_request_size_bytes",
		Buckets: prometheus.ExponentialBuckets(100, 10, 6),
	}, []string{"method", "path"})

	HttpResponseSize = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "gateway_http_response_size_bytes",
		Buckets: prometheus.ExponentialBuckets(100, 10, 6),
	}, []string{"method", "path", "status"})

	// Downstream service call latency (from gateway to each microservice)
	UpstreamCallDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "gateway_upstream_call_duration_seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"service", "status"}) // service: shortener | redirect | analytics

	// DB connection pool
	DbConnectionPoolSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "db_connection_pool_size",
		Help: "Database connection pool status.",
	}, []string{"service", "state"}) // state: idle | in_use | waiting
)

func init() {
	prometheus.MustRegister(HttpRequestsTotal)
	prometheus.MustRegister(HttpRequestDuration)
	prometheus.MustRegister(HttpRequestsInFlight)
	prometheus.MustRegister(HttpRequestSize)
	prometheus.MustRegister(HttpResponseSize)
	prometheus.MustRegister(UpstreamCallDuration)
	prometheus.MustRegister(DbConnectionPoolSize)
}

func Prometheus() gin.HandlerFunc {
	return func(c *gin.Context) {
		HttpRequestsInFlight.Inc()
		defer HttpRequestsInFlight.Dec()

		start := time.Now()
		c.Next()
		duration := time.Since(start).Seconds()

		status := strconv.Itoa(c.Writer.Status())
		method := c.Request.Method
		
		path := c.FullPath()
		if path == "" {
			path = "unmatched"
		}

		HttpRequestsTotal.WithLabelValues(method, path, status).Inc()
		HttpRequestDuration.WithLabelValues(method, path, status).Observe(duration)

		reqSize := c.Request.ContentLength
		if reqSize < 0 {
			reqSize = 0
		}
		HttpRequestSize.WithLabelValues(method, path).Observe(float64(reqSize))

		resSize := c.Writer.Size()
		if resSize < 0 {
			resSize = 0
		}
		HttpResponseSize.WithLabelValues(method, path, status).Observe(float64(resSize))
	}
}

func StartDBStatsTracking(db *gorm.DB, serviceName string) {
	sqlDB, err := db.DB()
	if err != nil {
		return
	}
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			stats := sqlDB.Stats()
			DbConnectionPoolSize.WithLabelValues(serviceName, "idle").Set(float64(stats.Idle))
			DbConnectionPoolSize.WithLabelValues(serviceName, "in_use").Set(float64(stats.InUse))
			DbConnectionPoolSize.WithLabelValues(serviceName, "waiting").Set(float64(stats.WaitCount))
		}
	}()
}