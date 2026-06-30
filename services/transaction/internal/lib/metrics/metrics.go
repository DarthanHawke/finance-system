// Пакет metrics содержит настройки для метрик prometheus
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Outbox
	OutboxEventsProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "outbox_events_processed_total",
		Help: "Total number of outbox events processed by outcome.",
	}, []string{"outcome"})

	OutboxBatchSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "outbox_batch_size",
		Help: "Number of events in the current batch.",
	})

	OutboxBatchDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "outbox_batch_duration_seconds",
		Help:    "Duration of batch processing in seconds.",
		Buckets: prometheus.DefBuckets,
	})

	OutboxGetEventsErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "outbox_get_events_errors_total",
		Help: "Total number of errors fetching pending events.",
	})

	// Consumer
	ConsumerMessagesProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "consumer_messages_processed_total",
		Help: "Total number of Kafka messages processed by outcome.",
	}, []string{"outcome"})

	ConsumerRetryableErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "consumer_retryable_errors_total",
		Help: "Total number of retryable errors.",
	})

	ConsumerFetchErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "consumer_fetch_errors_total",
		Help: "Total number of errors fetching messages from Kafka.",
	})

	ConsumerCommitErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "consumer_commit_errors_total",
		Help: "Total number of errors committing offsets.",
	})

	ConsumerDedupErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "consumer_dedup_errors_total",
		Help: "Total number of deduplication check errors.",
	})

	ConsumerMessageDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "consumer_message_duration_seconds",
		Help:    "Duration of processing a single Kafka message.",
		Buckets: prometheus.DefBuckets,
	})

	// Producer
	KafkaProduceTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "kafka_produce_total",
		Help: "Total number of Kafka produce operations.",
	}, []string{"outcome"})

	KafkaDLQSent = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "kafka_dlq_sent_total",
		Help: "Total number of messages sent to DLQ.",
	}, []string{"outcome"})

	KafkaProduceDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "kafka_produce_duration_seconds",
		Help:    "Duration of Kafka produce operations.",
		Buckets: prometheus.DefBuckets,
	})

	// Redis
	RedisCommandDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "redis_command_duration_seconds",
		Help:    "Duration of Redis commands.",
		Buckets: prometheus.DefBuckets,
	}, []string{"command"})
	CacheHitTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cache_hit_total",
		Help: "Total number of cache hits.",
	})

	CacheMissTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cache_miss_total",
		Help: "Total number of cache misses.",
	})

	// Transaction
	CorruptedPayloadTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "corrupted_payload_total",
		Help: "Total number of events with corrupted payload that could not be unmarshalled.",
	})
)
