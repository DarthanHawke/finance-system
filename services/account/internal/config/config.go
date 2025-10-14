package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Configuration struct {
	Env           string `mapstructure:"ENV" env-default:"prod"`
	GRPSServer    `mapstructure:",squash"`
	TLS           `mapstructure:",squash"`
	DataBase      `mapstructure:",squash"`
	Redis         `mapstructure:",squash"`
	JWT           `mapstructure:",squash"`
	JWTKeys       `mapstructure:",squash"`
	SSOClient     `mapstructure:",squash"`
	PaymentClient `mapstructure:",squash"`
}

type GRPSServer struct {
	Port    int `mapstructure:"SERVER_PORT" env-default:"50052"`
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
	Name     string `mapstructure:"DB_NAME" env-default:"payment_db"`
	SSLMode  string `mapstructure:"SSL_MODE" env-default:"disable"`
	RootCert string `mapstructure:"DB_ROOT_CERT" env-default:""`
	Cert     string `mapstructure:"DB_CERT" env-default:""`
	Key      string `mapstructure:"DB_KEY" env-default:""`
}

type KafkaConfig struct {
	Brokers          []string      `json:"brokers"`
	GroupID          string        `json:"group_id"`
	SessionTimeout   time.Duration `json:"session_timeout"`
	RebalanceTimeout time.Duration `json:"rebalance_timeout"`
	MaxRetries       int           `json:"max_retries"`
	RetryInterval    time.Duration `json:"retry_interval"`
}

type Redis struct {
	Addr     string `mapstructure:"REDIS_ADDR"`
	Password string `mapstructure:"REDIS_PASSWORD"`
	DB       int    `mapstructure:"REDIS_DB"`
}

type JWT struct {
	AccessTokenTTL  time.Duration `mapstructure:"JWT_ACCESS_TOKEN_TTL" env-default:"15m"`
	RefreshTokenTTL time.Duration `mapstructure:"JWT_REFRESH_TOKEN_TTL" env-default:"168h"`
	Issuer          string        `mapstructure:"JWT_ISSUER" env-default:"payment-system"`
}

type JWTKeys struct {
	JWTPublicKeyPath string `mapstructure:"JWT_KEY_PUBLIC" env-default:"./security/public.example.pem"`
}

type SSOClient struct {
	Address      string        `mapstructure:"CLIENT_SSO_ADRESS" env-default:"sso-service:50051"`
	Timeout      time.Duration `mapstructure:"CLIENT_SSO_TIMEOUT" env-default:"10s"`
	RetriesCount int           `mapstructure:"CLIENT_SSO_RETRIES" env-default:"5"`
}
type PaymentClient struct {
	Address      string        `mapstructure:"CLIENT_PAYMENT_ADRESS" env-default:"payment-service:50058"`
	Timeout      time.Duration `mapstructure:"CLIENT_PAYMENT_TIMEOUT" env-default:"10s"`
	RetriesCount int           `mapstructure:"CLIENT_PAYMENT_RETRIES" env-default:"5"`
}

type SystemUsers struct {
	SystemUsers []User `mapstructure:"system_users"`
}

type User struct {
	Role     string `mapstructure:"role"`
	FullName string `mapstructure:"full_name"`
	Email    string `mapstructure:"email"`
	Password string `mapstructure:"password"`
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

func LoadSystemUsers(yaml string) (config *SystemUsers, err error) {
	v := viper.New()
	v.SetConfigName(fmt.Sprintf(".yaml.%s", yaml))
	v.AddConfigPath(".")
	v.SetConfigType("yaml")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return config, nil
}
