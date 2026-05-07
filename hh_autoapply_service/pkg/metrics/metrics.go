package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP
	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests by method, path, and status code.",
	}, []string{"method", "path", "status"})

	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	// Business
	AutoApplyRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "autoapply_requests_total",
		Help: "Total auto-apply requests by outcome status.",
	}, []string{"status"}) // created | completed | failed

	// Kafka
	KafkaMessagesPublished = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "kafka_messages_published_total",
		Help: "Total Kafka messages published by topic.",
	}, []string{"topic"})

	// Redis cache
	RedisCacheHits = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "redis_cache_hits_total",
		Help: "Redis cache hits by cache name.",
	}, []string{"cache"}) // token | autoapply

	RedisCacheMisses = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "redis_cache_misses_total",
		Help: "Redis cache misses by cache name.",
	}, []string{"cache"})
)
