package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// ISO8583Config конфигурация для ISO сообщений
type ISO8583Config struct {
	BankCode      string `mapstructure:"bank_code" json:"bank_code"`
	TerminalID    string `mapstructure:"terminal_id" json:"terminal_id"`
	MerchantID    string `mapstructure:"merchant_id" json:"merchant_id"`
	AcquirerID    string `mapstructure:"acquirer_id" json:"acquirer_id"`
	ForwarderID   string `mapstructure:"forwarder_id" json:"forwarder_id"`
	CurrencyCode  string `mapstructure:"currency_code" json:"currency_code"`
	InstitutionID string `mapstructure:"institution_id" json:"institution_id"`
}

// LoadISO8583Config загружает конфигурацию ISO8583 из JSON файла
func LoadISO8583Config(configPath string) (*ISO8583Config, error) {
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("json")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read ISO8583 config: %w", err)
	}

	var config ISO8583Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ISO8583 config: %w", err)
	}

	return &config, nil
}
