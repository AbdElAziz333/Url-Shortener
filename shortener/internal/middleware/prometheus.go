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
		Name: "shortener_http_requests_total",
		Help: "Total number of HTTP requests processed by the shortener service.",
	}, []string{"method", "path", "status"})

	HttpRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "shortener_http_request_duration_seconds",
		Help:    "Latency of HTTP requests processed by the shortener service in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path", "status"})

	// How many URLs are being created
	UrlsCreatedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "shortener_urls_created_total",
		Help: "Total number of short URLs created.",
	}, []string{"status"}) // status: success | failure

	// Collision rate on hash generation (important for shortener correctness)
	HashCollisionsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "shortener_hash_collisions_total",
		Help: "Total number of hash collisions encountered during URL shortening.",
	})

	// DB write latency (separate from HTTP latency)
	DbWriteDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "shortener_db_write_duration_seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"operation"}) // operation: insert | check_exists

	// DB connection pool
	DbConnectionPoolSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "db_connection_pool_size",
		Help: "Database connection pool status.",
	}, []string{"service", "state"}) // state: idle | in_use | waiting
)

func init() {
	prometheus.MustRegister(HttpRequestsTotal)
	prometheus.MustRegister(HttpRequestDuration)
	prometheus.MustRegister(UrlsCreatedTotal)
	prometheus.MustRegister(HashCollisionsTotal)
	prometheus.MustRegister(DbWriteDuration)
	prometheus.MustRegister(DbConnectionPoolSize)
}

func Prometheus() gin.HandlerFunc {
	return func(c *gin.Context) {
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