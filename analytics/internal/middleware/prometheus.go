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
		Name: "analytics_http_requests_total",
		Help: "Total number of HTTP requests processed by the analytics service.",
	}, []string{"method", "path", "status"})

	HttpRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "analytics_http_request_duration_seconds",
		Help:    "Latency of HTTP requests processed by the analytics service in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path", "status"})

	// Queue lag — how far behind is analytics processing
	AnalyticsEventsQueueDepth = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "analytics_events_queue_depth",
		Help: "Number of unprocessed analytics events in queue.",
	}, []string{"topic"})

	// Processing failures
	AnalyticsProcessingFailuresTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "analytics_processing_failures_total",
		Help: "Total number of failures during analytics event processing.",
	}, []string{"reason"})

	// Event processing latency
	AnalyticsProcessingDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "analytics_event_processing_duration_seconds",
		Help:    "Duration of event processing in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"event_type"})

	// DB connection pool
	DbConnectionPoolSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "db_connection_pool_size",
		Help: "Database connection pool status.",
	}, []string{"service", "state"}) // state: idle | in_use | waiting
)

func init() {
	prometheus.MustRegister(HttpRequestsTotal)
	prometheus.MustRegister(HttpRequestDuration)
	prometheus.MustRegister(AnalyticsEventsQueueDepth)
	prometheus.MustRegister(AnalyticsProcessingFailuresTotal)
	prometheus.MustRegister(AnalyticsProcessingDuration)
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