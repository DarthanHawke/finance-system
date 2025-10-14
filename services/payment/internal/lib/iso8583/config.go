package iso8583

// Config содержит константные значения для ISO 8583
type Config struct {
	// Банковские идентификаторы
	AcquiringInstitutionID  string // код банка-эквайера
	ForwardingInstitutionID string // код банка-отправителя
	TerminalID              string // идентификатор терминала
	MerchantID              string // идентификатор мерчанта
	MerchantName            string // название мерчанта
	MerchantLocation        string // местоположение мерчанта

	// Настройки системы
	SystemTraceNumber string // начальный номер трассировки
	CurrencyCode      string // числовой код валюты (840 - USD, 643 - RUB и т.д.)
	CountryCode       string // код страны (643 - Россия и т.д.)

	// Настройки обработки
	ProcessingID string // код обработки по умолчанию
	POSEntryMode string // режим ввода POS
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() *Config {
	return &Config{
		AcquiringInstitutionID:  "100000",
		ForwardingInstitutionID: "200000",
		TerminalID:              "TERM001",
		MerchantID:              "MERCH001",
		MerchantName:            "Test Merchant",
		MerchantLocation:        "MOSCOW",
		SystemTraceNumber:       "000001",
		CurrencyCode:            "643", // RUB
		CountryCode:             "643", // Russia
		ProcessingID:            "000000",
		POSEntryMode:            "021",
	}
}
