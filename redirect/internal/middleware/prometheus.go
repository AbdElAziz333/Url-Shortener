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
		Name: "redirect_http_requests_total",
		Help: "Total number of HTTP requests processed by the redirect service.",
	}, []string{"method", "path", "status"})

	HttpRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "redirect_http_request_duration_seconds",
		Help:    "Latency of HTTP requests processed by the redirect service in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path", "status"})

	// Cache hit/miss ratio — the most important metric for redirect performance
	RedirectCacheHitsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "redirect_cache_hits_total",
		Help: "Total number of cache hits or misses.",
	}, []string{"result"}) // result: hit | miss | expired

	// Lookup latency broken down by source
	RedirectLookupDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "redirect_lookup_duration_seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"source"}) // source: cache | database

	// 404s for unknown short codes (detect enumeration attacks)
	RedirectNotFoundTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "redirect_not_found_total",
		Help: "Total lookups for non-existent short codes.",
	})

	// DB connection pool
	DbConnectionPoolSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "db_connection_pool_size",
		Help: "Database connection pool status.",
	}, []string{"service", "state"}) // state: idle | in_use | waiting
)

func init() {
	prometheus.MustRegister(HttpRequestsTotal)
	prometheus.MustRegister(HttpRequestDuration)
	prometheus.MustRegister(RedirectCacheHitsTotal)
	prometheus.MustRegister(RedirectLookupDuration)
	prometheus.MustRegister(RedirectNotFoundTotal)
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