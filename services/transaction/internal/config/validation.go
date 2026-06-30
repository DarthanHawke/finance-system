// Пакет config реализовывает работу с конфиигурациями сервиса
package config

import (
	"fmt"
	"strings"
	"time"
)

// Validate проверяет всю конфигурацию
func (c *Configuration) Validate() error {
	var errs []string

	if err := c.GRPCServer.Validate(); err != nil {
		errs = append(errs, err.Error())
	}

	if err := c.DataBase.Validate(); err != nil {
		errs = append(errs, err.Error())
	}

	if err := c.Redis.Validate(); err != nil {
		errs = append(errs, err.Error())
	}

	if err := c.Kafka.Validate(); err != nil {
		errs = append(errs, err.Error())
	}

	if err := c.Logs.Validate(); err != nil {
		errs = append(errs, err.Error())
	}

	if err := c.Tracing.Validate(); err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return fmt.Errorf("configuration validation failed:\n  %s",
			strings.Join(errs, "\n  "))
	}

	return nil
}

func (g *GRPCServer) Validate() error {
	var errs []string

	if g.Port < 1 || g.Port > 65535 {
		errs = append(errs, fmt.Sprintf("port must be between 1 and 65535, got %d", g.Port))
	}

	if g.Timeout < 1 {
		errs = append(errs, fmt.Sprintf("timeout must be >= 1 second, got %d", g.Timeout))
	}

	if len(errs) > 0 {
		return fmt.Errorf("grpc server: %s", strings.Join(errs, "; "))
	}

	return nil
}

func (d *DataBase) Validate() error {
	var errs []string

	if d.Host == "" {
		errs = append(errs, "host cannot be empty")
	}

	if d.Port < 1 || d.Port > 65535 {
		errs = append(errs, fmt.Sprintf("port must be between 1 and 65535, got %d", d.Port))
	}

	if d.User == "" {
		errs = append(errs, "user cannot be empty")
	}

	if d.Name == "" {
		errs = append(errs, "database name cannot be empty")
	}

	if len(errs) > 0 {
		return fmt.Errorf("database: %s", strings.Join(errs, "; "))
	}

	return nil
}

func (r *Redis) Validate() error {
	var errs []string

	if r.Addr == "" {
		errs = append(errs, "addr cannot be empty")
	}

	if r.DB < 0 {
		errs = append(errs, fmt.Sprintf("db must be >= 0, got %d", r.DB))
	}

	if len(errs) > 0 {
		return fmt.Errorf("redis: %s", strings.Join(errs, "; "))
	}

	return nil
}

func (k *Kafka) Validate() error {
	var errs []string

	// Brokers
	if k.Brokers == "" {
		errs = append(errs, "brokers cannot be empty")
	}

	// Consumer
	if k.ConsumerTopics == "" {
		errs = append(errs, "consumer_topics cannot be empty")
	}

	if k.ConsumerGroupID == "" {
		errs = append(errs, "consumer_group_id cannot be empty")
	}

	if k.ConsumerMinBytes < 1 {
		errs = append(errs, fmt.Sprintf("consumer_min_bytes must be >= 1, got %d", k.ConsumerMinBytes))
	}

	if k.ConsumerMaxBytes < k.ConsumerMinBytes {
		errs = append(errs, fmt.Sprintf("consumer_max_bytes (%d) must be >= consumer_min_bytes (%d)",
			k.ConsumerMaxBytes, k.ConsumerMinBytes))
	}

	if k.ConsumerMaxWait < 1 {
		errs = append(errs, fmt.Sprintf("consumer_max_wait must be >= 1ms, got %d", k.ConsumerMaxWait))
	}

	if k.ConsumerCommitInterval < 1 {
		errs = append(errs, fmt.Sprintf("consumer_commit_interval must be >= 1ms, got %d", k.ConsumerCommitInterval))
	}

	if k.ConsumerSessionTimeout < 1 {
		errs = append(errs, fmt.Sprintf("consumer_session_timeout must be >= 1ms, got %d", k.ConsumerSessionTimeout))
	}

	if k.ConsumerRebalanceTimeout < 1 {
		errs = append(errs, fmt.Sprintf("consumer_rebalance_timeout must be >= 1ms, got %d", k.ConsumerRebalanceTimeout))
	}

	if k.ConsumerConcurrency < 1 {
		errs = append(errs, fmt.Sprintf("consumer_concurrency must be >= 1, got %d", k.ConsumerConcurrency))
	}

	// Producer
	if k.ProducerTopic == "" {
		errs = append(errs, "producer_topic cannot be empty")
	}

	if k.ProducerBatchSize < 1 {
		errs = append(errs, fmt.Sprintf("producer_batch_size must be >= 1, got %d", k.ProducerBatchSize))
	}

	if k.ProducerBatchTimeout < 1 {
		errs = append(errs, fmt.Sprintf("producer_batch_timeout must be >= 1ms, got %d", k.ProducerBatchTimeout))
	}

	// ProducerRequiredAcks: -1 (all), 0 (none), 1 (leader)
	validAcks := map[int]bool{-1: true, 0: true, 1: true}
	if !validAcks[k.ProducerRequiredAcks] {
		errs = append(errs, fmt.Sprintf("producer_required_acks must be -1, 0, or 1, got %d", k.ProducerRequiredAcks))
	}

	if k.ProducerMaxAttempts < 1 {
		errs = append(errs, fmt.Sprintf("producer_max_attempts must be >= 1, got %d", k.ProducerMaxAttempts))
	}

	if k.ProducerWriteTimeout < 1 {
		errs = append(errs, fmt.Sprintf("producer_write_timeout must be >= 1ms, got %d", k.ProducerWriteTimeout))
	}

	// DLQ
	if k.DLQTopic == "" {
		errs = append(errs, "dlq_topic cannot be empty")
	}

	if k.DLQBatchSize < 1 {
		errs = append(errs, fmt.Sprintf("dlq_batch_size must be >= 1, got %d", k.DLQBatchSize))
	}

	if k.DLQBatchTimeout < 1 {
		errs = append(errs, fmt.Sprintf("dlq_batch_timeout must be >= 1ms, got %d", k.DLQBatchTimeout))
	}

	if k.DLQMaxAttempts < 1 {
		errs = append(errs, fmt.Sprintf("dlq_max_attempts must be >= 1, got %d", k.DLQMaxAttempts))
	}

	// Retry
	if k.RetryMaxAttempts < 1 {
		errs = append(errs, fmt.Sprintf("retry_max_attempts must be >= 1, got %d", k.RetryMaxAttempts))
	}

	if k.RetryInitialWait < 1 {
		errs = append(errs, fmt.Sprintf("retry_initial_wait must be >= 1ms, got %d", k.RetryInitialWait))
	}

	if k.RetryMaxWait < k.RetryInitialWait {
		errs = append(errs, fmt.Sprintf("retry_max_wait (%d) must be >= retry_initial_wait (%d)",
			k.RetryMaxWait, k.RetryInitialWait))
	}

	if k.RetryMultiplier <= 1.0 {
		errs = append(errs, fmt.Sprintf("retry_multiplier must be > 1.0, got %.2f", k.RetryMultiplier))
	}

	// Processor
	if k.ProcessorBatchSize < 1 {
		errs = append(errs, fmt.Sprintf("processor_batch_size must be >= 1, got %d", k.ProcessorBatchSize))
	}

	if k.ProcessorHandlePeriod < 1 {
		errs = append(errs, fmt.Sprintf("processor_handle_period must be >= 1ms, got %d", k.ProcessorHandlePeriod))
	}

	// Concurrency
	if k.Concurrency < 1 {
		errs = append(errs, fmt.Sprintf("concurrency must be >= 1, got %d", k.Concurrency))
	}

	if len(errs) > 0 {
		return fmt.Errorf("kafka: %s", strings.Join(errs, "; "))
	}

	return nil
}

func (l *Logs) Validate() error {
	var errs []string

	if l.MaxSize < 1 {
		errs = append(errs, fmt.Sprintf("max_size must be >= 1MB, got %d", l.MaxSize))
	}

	if l.MaxBackups < 0 {
		errs = append(errs, fmt.Sprintf("max_backups must be >= 0, got %d", l.MaxBackups))
	}

	if l.MaxAge < 0 {
		errs = append(errs, fmt.Sprintf("max_age must be >= 0 days, got %d", l.MaxAge))
	}

	if len(errs) > 0 {
		return fmt.Errorf("logs: %s", strings.Join(errs, "; "))
	}

	return nil
}

func (t *Tracing) Validate() error {
	var errs []string

	if t.OTLPEndpoint == "" {
		errs = append(errs, "otlp_endpoint cannot be empty")
	}

	if len(errs) > 0 {
		return fmt.Errorf("tracing: %s", strings.Join(errs, "; "))
	}

	return nil
}

// nextPowerOfTwo возвращает ближайшую степень двойки >= n
func nextPowerOfTwo(n int) int {
	if n <= 1 {
		return 1
	}
	// Алгоритм для нахождения степени двойки
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n++
	return n
}

// DurationFromMilliseconds возвращает time.Duration из миллисекунд
func DurationFromMilliseconds(ms int) time.Duration {
	return time.Duration(ms) * time.Millisecond
}
