package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Configuration struct {
	Env               string `mapstructure:"ENV" env-default:"prod"`
	ISO8583ConfigPath string `mapstructure:"ISO8583_CONFIG_PATH" env-default:"iso8583.example"`
	GRPCServer        `mapstructure:",squash"`
	TLS               `mapstructure:",squash"`
	DataBase          `mapstructure:",squash"`
	Redis             `mapstructure:",squash"`
	Kafka             `mapstructure:",squash"`
}

type GRPCServer struct {
	Port    int `mapstructure:"SERVER_PORT" env-default:"50058"`
	Timeout int `mapstructure:"SERVER_TIMEOUT" env-default:"10"`
}

type TLS struct {
	CA      string `mapstructure:"TLS_CA" env-default:"./security/CA.example.crt"`
	TLSKey  string `mapstructure:"TLS_SERVER" env-default:"./security/server.example.key"`
	TLSCert string `mapstructure:"TLS_CERT" env-default:"./security/server.example.crt"`
}

type DataBase struct {
	Host     string `mapstructure:"DB_HOST" env-default:"postgres"`
	Port     int    `mapstructure:"DB_PORT" env-default:"5432"`
	User     string `mapstructure:"DB_USER" env-default:"postgres"`
	Password string `mapstructure:"DB_PASSWORD" env-default:"secret"`
	Name     string `mapstructure:"DB_NAME" env-default:"transaction_db"`
	SSLMode  string `mapstructure:"SSL_MODE" env-default:"disable"`
	RootCert string `mapstructure:"DB_ROOT_CERT" env-default:""`
	Cert     string `mapstructure:"DB_CERT" env-default:""`
	Key      string `mapstructure:"DB_KEY" env-default:""`
}

type Redis struct {
	Addr     string `mapstructure:"REDIS_ADDR"`
	Password string `mapstructure:"REDIS_PASSWORD"`
	DB       int    `mapstructure:"REDIS_DB"`
}

type Kafka struct {
	Brokers                  string  `mapstructure:"KAFKA_BROKERS" env-default:"localhost:9092"`
	ConsumerTopics           string  `mapstructure:"KAFKA_CONSUMER_TOPICS"`
	ConsumerGroupID          string  `mapstructure:"KAFKA_CONSUMER_GROUP_ID" env-default:"transaction-service"`
	ConsumerMinBytes         int     `mapstructure:"KAFKA_CONSUMER_MIN_BYTES" env-default:"1"`
	ConsumerMaxBytes         int     `mapstructure:"KAFKA_CONSUMER_MAX_BYTES" env-default:"10e6"`
	ConsumerMaxWait          int     `mapstructure:"KAFKA_CONSUMER_MAX_WAIT" env-default:"500"`
	ConsumerCommitInterval   int     `mapstructure:"KAFKA_CONSUMER_COMMIT_INTERVAL" env-default:"1000"`
	ConsumerSessionTimeout   int     `mapstructure:"KAFKA_CONSUMER_SESSION_TIMEOUT" env-default:"10000"`
	ConsumerRebalanceTimeout int     `mapstructure:"KAFKA_CONSUMER_REBALANCE_TIMEOUT" env-default:"60000"`
	ConsumerStartOffset      int64   `mapstructure:"KAFKA_CONSUMER_START_OFFSET" env-default:"-1"`
	ConsumerConcurrency      int     `mapstructure:"KAFKA_CONSUMER_CONCURRENCY" env-default:"1"`
	ProducerTopic            string  `mapstructure:"KAFKA_PRODUCER_TOPIC" env-default:"transaction-commands"`
	ProducerBatchSize        int     `mapstructure:"KAFKA_PRODUCER_BATCH_SIZE" env-default:"100"`
	ProducerBatchTimeout     int     `mapstructure:"KAFKA_PRODUCER_BATCH_TIMEOUT" env-default:"100"`
	ProducerRequiredAcks     int     `mapstructure:"KAFKA_PRODUCER_REQUIRED_ACKS" env-default:"-1"`
	ProducerMaxAttempts      int     `mapstructure:"KAFKA_PRODUCER_MAX_ATTEMPTS" env-default:"3"`
	ProducerWriteTimeout     int     `mapstructure:"KAFKA_PRODUCER_WRITE_TIMEOUT" env-default:"5000"`
	DLQTopic                 string  `mapstructure:"KAFKA_DLQ_TOPIC" env-default:"dlq"`
	DLQBatchSize             int     `mapstructure:"KAFKA_DLQ_BATCH_SIZE" env-default:"10"`
	DLQBatchTimeout          int     `mapstructure:"KAFKA_DLQ_BATCH_TIMEOUT" env-default:"100"`
	DLQMaxAttempts           int     `mapstructure:"KAFKA_DLQ_MAX_ATTEMPTS" env-default:"1"`
	RetryMaxAttempts         int     `mapstructure:"KAFKA_RETRY_MAX_ATTEMPTS" env-default:"3"`
	RetryInitialWait         int     `mapstructure:"KAFKA_RETRY_INITIAL_WAIT" env-default:"100"`
	RetryMaxWait             int     `mapstructure:"KAFKA_RETRY_MAX_WAIT" env-default:"5000"`
	RetryMultiplier          float64 `mapstructure:"KAFKA_RETRY_MULTIPLIER" env-default:"2.0"`
}

func (c DataBase) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s&sslrootcert=%s&sslcert=%s&sslkey=%s",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Name,
		c.SSLMode,
		c.RootCert,
		c.Cert,
		c.Key,
	)
}

func LoadConfig(env string) (config *Configuration, err error) {
	v := viper.New()
	v.SetConfigName(fmt.Sprintf(".env.%s", env))
	v.AddConfigPath(".")
	v.SetConfigType("env")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return config, nil
}
