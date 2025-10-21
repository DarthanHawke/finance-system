// Пакет models содержит различные dto'шки, модельки, константы
package models

import "time"

// MTI (Message Type Indicator) константы
const (
	MTIFinancialRequest  = "0200" // Financial Transaction Request
	MTIFinancialResponse = "0210" // Financial Transaction Response
	MTIReversalAdvice    = "0420" // Reversal Advice
)

// Processing Code константы
const (
	// Transaction Types (первые 2 цифры)
	PCTransactionPurchase   = "00" // Purchase
	PCTransactionWithdrawal = "01" // Withdrawal
	PCTransactionTransfer   = "06" // Transfer
	PCTransactionPayment    = "07" // Payment
	PCTransactionRefund     = "20" // Refund
	PCTransactionDeposit    = "21" // Deposit

	// Account Types (3-4 и 5-6 цифры)
	PCAccountDefault    = "00" // Default/Unspecified
	PCAccountSavings    = "10" // Savings Account
	PCAccountChecking   = "20" // Checking Account
	PCAccountCreditCard = "30" // Credit Card
	PCAccountUniversal  = "40" // Universal Account
	PCAccountElectronic = "60" // Electronic Money
)

// Response Code константы
const (
	RCSuccess                 = "00" // Approved
	RCReferToIssuer           = "01" // Refer to issuer
	RCInvalidMerchant         = "03" // Invalid merchant
	RCDoNotHonor              = "05" // Do not honor
	RCInvalidTransaction      = "12" // Invalid transaction
	RCInvalidAmount           = "13" // Invalid amount
	RCInvalidCard             = "14" // Invalid card
	RCNoSuchIssuer            = "15" // No such issuer
	RCFormatError             = "30" // Format error
	RCLostCard                = "41" // Lost card
	RCStolenCard              = "43" // Stolen card
	RCInsufficientFunds       = "51" // Insufficient funds
	RCExpiredCard             = "54" // Expired card
	RCInvalidPIN              = "55" // Invalid PIN
	RCTransactionNotPermitted = "57" // Transaction not permitted
	RCExceedWithdrawalLimit   = "61" // Exceed withdrawal limit
	RCSecurityViolation       = "63" // Security violation
	RCExceedActivityLimit     = "65" // Exceed activity limit
	RCPINTryExceed            = "75" // PIN try exceed
	RCTimeout                 = "91" // Timeout
	RCSystemError             = "96" // System error
)

// OperationType типы операций для роутинга
type OperationType string

const (
	OpPurchase   OperationType = "purchase"
	OpWithdrawal OperationType = "withdrawal"
	OpTransfer   OperationType = "transfer"
	OpPayment    OperationType = "payment"
	OpRefund     OperationType = "refund"
	OpDeposit    OperationType = "deposit"
	OpReversal   OperationType = "reversal"
	OpUnknown    OperationType = "unknown"
)

// ISO8583Message основная структура ISO сообщения
type ISO8583Message struct {
	MTI               string    `json:"mti"`                // Message Type Indicator
	PrimaryAccount    string    `json:"primary_account"`    // поле 2
	ProcessingCode    string    `json:"processing_code"`    // поле 3
	Amount            string    `json:"amount"`             // поле 4
	TransmissionTime  time.Time `json:"transmission_time"`  // поле 7
	STAN              string    `json:"stan"`               // поле 11
	LocalTime         time.Time `json:"local_time"`         // поле 12
	LocalDate         time.Time `json:"local_date"`         // поле 13
	ExpirationDate    string    `json:"expiration_date"`    // поле 14
	SettlementDate    string    `json:"settlement_date"`    // поле 15
	MerchantType      string    `json:"merchant_type"`      // поле 18
	AcquirerID        string    `json:"acquirer_id"`        // поле 32
	Track2Data        string    `json:"track_2_data"`       // поле 35
	RRN               string    `json:"rrn"`                // поле 37
	AuthorizationID   string    `json:"authorization_id"`   // поле 38
	ResponseCode      string    `json:"response_code"`      // поле 39
	TerminalID        string    `json:"terminal_id"`        // поле 41
	MerchantID        string    `json:"merchant_id"`        // поле 42
	AdditionalData    string    `json:"additional_data"`    // поле 48
	Currency          string    `json:"currency"`           // поле 49
	PINBlock          string    `json:"pin_block"`          // поле 52
	AdditionalAmount  string    `json:"additional_amount"`  // поле 54
	ICCSystemData     string    `json:"icc_system_data"`    // поле 55
	OriginalData      string    `json:"original_data"`      // поле 56
	ForwarderID       string    `json:"forwarder_id"`       // поле 61
	ReplacementAmount string    `json:"replacement_amount"` // поле 95
	ReceivingID       string    `json:"receiving_id"`       // поле 100
	AccountID1        string    `json:"account_id_1"`       // поле 102
	AccountID2        string    `json:"account_id_2"`       // поле 103
}

// ISO8583Payload для событий с ISO сообщениями
type ISO8583Payload struct {
	ISOMessage   *ISO8583Message `json:"iso_message"`
	Success      bool            `json:"success"`
	ErrorCode    string          `json:"error_code,omitempty"`
	ErrorMessage string          `json:"error_message,omitempty"`
}

// CurrencyInfo информация о валюте
type CurrencyInfo struct {
	Code     string
	Decimals int // количество знаков после запятой
	Name     string
}

var CurrencyMap = map[string]CurrencyInfo{
	"RUB": {Code: "643", Decimals: 2, Name: "Russian Ruble"},
	"USD": {Code: "840", Decimals: 2, Name: "US Dollar"},
	"EUR": {Code: "978", Decimals: 2, Name: "Euro"},
}

// AnalysisResult результат анализа Processing Code
type AnalysisResult struct {
	RawCode         string
	TransactionType string // первые 2 цифры
	FromAccountType string // 3-4 цифры
	ToAccountType   string // 5-6 цифры
	OperationType   OperationType
}
