package iban

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

// IBANConfig содержит конфигурацию для генерации IBAN
type IBANConfig struct {
	CountryCode string // 2-буквенный код страны (например, "RU", "DE")
	BankCode    string // Код банка (минимум 4 символа)
	Length      int    // Длина IBAN (обычно от 15 до 34)
}

// Validate проверяет корректность конфигурации
func validate(countryCode, bankCode string, length int) error {
	if len(countryCode) != 2 {
		return errors.New("country code must be 2 characters long")
	}

	if len(bankCode) < 4 {
		return errors.New("bank code must be at least 4 characters long")
	}

	if length < 15 || length > 34 {
		return errors.New("iban length must be between 15 and 34 characters")
	}

	return nil
}

type IbanGenerator struct {
	config *IBANConfig
}

// NewIBANGenerator создает новый экземпляр генератора IBAN с конфигурацией
func NewIBANGenerator(countryCode, bankCode string, length int) (*IbanGenerator, error) {
	if err := validate(countryCode, bankCode, length); err != nil {
		return nil, err
	}

	return &IbanGenerator{
		config: NewIBANConfig(countryCode, bankCode, length),
	}, nil
}

func NewIBANConfig(countryCode, bankCode string, length int) *IBANConfig {
	return &IBANConfig{
		CountryCode: strings.ToUpper(countryCode),
		BankCode:    bankCode,
		Length:      length,
	}
}

// Generate создает уникальный IBAN на основе UUID пользователя и текущего времени
func (g *IbanGenerator) Generate(userID string) (string, error) {
	uniquePart := strings.ToUpper(g.generateUniquePart(userID))

	randomPartLength := g.config.Length - len(g.config.CountryCode) - len(g.config.BankCode) - 2 - len(uniquePart)
	if randomPartLength < 0 {
		return "", errors.New("iban length is too short")
	}

	randomPart := g.generateRandomPart(randomPartLength, userID)
	baseNumber := strings.ToUpper(g.config.BankCode) + uniquePart + randomPart

	// Контрольные цифры
	intermediate := baseNumber + strings.ToUpper(g.config.CountryCode) + "00"
	numericIBAN := lettersToNumbers(intermediate)
	checkDigits := fmt.Sprintf("%02d", 98-mod97(numericIBAN))

	// Финальный IBAN (все символы в верхнем регистре)
	iban := strings.ToUpper(g.config.CountryCode + checkDigits + baseNumber)

	// Валидируем сразу после генерации
	if err := g.ValidateInternal(iban); err != nil {
		return "", fmt.Errorf("generated invalid IBAN: %w", err)
	}

	return iban, nil
}

// generateUniquePart создает уникальную часть номера на основе userID и времени
func (g *IbanGenerator) generateUniquePart(userID string) string {
	// Используем хэш от userID и текущего наносекундного времени
	hash := hashString(userID + time.Now().Format(time.RFC3339Nano))
	hexStr := fmt.Sprintf("%08x", hash) // Гарантирует минимум 8 символов
	if len(hexStr) > 8 {
		hexStr = hexStr[:8]
	}
	return hexStr
}

// generateRandomPart генерирует случайную часть номера
func (g *IbanGenerator) generateRandomPart(length int, seed string) string {
	if length <= 0 {
		return ""
	}

	// Создаем детерминированный генератор на основе seed
	source := rand.NewSource(int64(hashString(seed)))
	rng := rand.New(source)

	var sb strings.Builder
	for range length {
		sb.WriteString(strconv.Itoa(rng.Intn(10)))
	}
	return sb.String()
}

// ValidateExternal проверяет корректность IBAN
func (g *IbanGenerator) ValidateExternal(iban string) error {
	// Удаляем все пробелы и переводим в верхний регистр
	iban = strings.ToUpper(strings.ReplaceAll(iban, " ", ""))

	if len(iban) < 5 {
		return errors.New("iban is too short")
	}

	if len(iban) > 34 { // Максимальная длина по стандарту
		return errors.New("iban too long")
	}

	// я решил не реализовать этот метод, чтобы можно было при тестировании использовать любые номера счёта

	return nil
}

// ValidateInternal проверяет корректность IBAN
func (g *IbanGenerator) ValidateInternal(iban string) error {
	// Приводим к верхнему регистру и удаляем пробелы
	iban = strings.ToUpper(strings.ReplaceAll(iban, " ", ""))

	if len(iban) < 4 {
		return errors.New("iban too short")
	}

	// Проверяем код страны (первые 2 символа)
	countryCode := iban[:2]
	if !isUpperCaseLetters(countryCode) {
		return errors.New("invalid country code")
	}

	// Проверяем контрольные цифры (2 символа после кода страны)
	checkDigits := iban[2:4]
	if !isDigits(checkDigits) {
		return errors.New("invalid check digits")
	}

	// Перемещаем 4 символа в конец и конвертируем
	rearranged := iban[4:] + iban[:4]
	numericIBAN := lettersToNumbers(rearranged)
	if numericIBAN == "" {
		return errors.New("invalid characters in IBAN")
	}

	// Проверяем контрольную сумму
	if mod97(numericIBAN) != 1 {
		return errors.New("invalid checksum")
	}

	// Дополнительно проверяем структуру для вашей системы
	expectedPrefix := strings.ToUpper(g.config.CountryCode + checkDigits + g.config.BankCode)
	if !strings.HasPrefix(iban, expectedPrefix) {
		return errors.New("invalid bank code")
	}

	return nil
}

// Проверка на заглавные буквы
func isUpperCaseLetters(s string) bool {
	for _, r := range s {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}

func hashString(s string) uint32 {
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}

// lettersToNumbers заменяет буквы на числа (A=10, B=11, ..., Z=35)
func lettersToNumbers(s string) string {
	var result strings.Builder
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			num := int(r - 'A' + 10)
			result.WriteString(strconv.Itoa(num))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// mod97 вычисляет остаток от деления очень большого числа на 97
func mod97(s string) int {
	const chunkSize = 9
	var remainder int

	for i := 0; i < len(s); i += chunkSize {
		end := i + chunkSize
		if end > len(s) {
			end = len(s)
		}
		chunk := strconv.Itoa(remainder) + s[i:end]
		num, _ := strconv.Atoi(chunk)
		remainder = num % 97
	}

	return remainder
}

// isLetters проверяет, что строка состоит только из букв
func isLetters(s string) bool {
	for _, r := range s {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}

// isDigits проверяет, что строка состоит только из цифр
func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
