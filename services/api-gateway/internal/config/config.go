package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Configuration struct {
	Env           string `mapstructure:"ENV" env-default:"prod"`
	HTTPServer    `mapstructure:",squash"`
	TLS           `mapstructure:",squash"`
	BillingClient `mapstructure:",squash"`
}

type HTTPServer struct {
	Port int `mapstructure:"SERVER_PORT" env-default:"8080"`
}

type TLS struct {
	CA      string `mapstructure:"TLS_CA" env-default:"./security/CA.example.crt"`
	TLSKey  string `mapstructure:"TLS_SERVER" env-default:"./security/server.example.key"`
	TLSCert string `mapstructure:"TLS_CERT" env-default:"./security/server.example.crt"`
}

type BillingClient struct {
	Address      string        `mapstructure:"CLIENT_BILLING_ADRESS" env-default:"sso-service:50052"`
	Timeout      time.Duration `mapstructure:"CLIENT_BILLING_TIMEOUT" env-default:"10s"`
	RetriesCount int           `mapstructure:"CLIENT_BILLING_RETRIES" env-default:"5"`
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
