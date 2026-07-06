// stan пакет для работы со значением stan
package stan

import (
	"fmt"
	"sync"
	"time"
)

// STAN структура релизующая методы для работы со STAN
type STAN struct {
	mu         sync.RWMutex
	counters   map[string]int // ключ: "date_transactionType_senderType" -> счетчик
	dateFormat string
}

func NewSTAN() *STAN {
	return &STAN{
		counters:   make(map[string]int),
		dateFormat: "2006-01-02",
	}
}

// GetNextSTAN возвращает следующий STAN для комбинации параметров
func (s *STAN) GetNextSTAN(transactionType, senderType, recipientType string, date time.Time) (string, error) {
	key := s.generateKey(transactionType, senderType, recipientType, date)

	s.mu.Lock()
	defer s.mu.Unlock()

	// Увеличиваем счетчик
	s.counters[key]++

	// Ограничиваем максимальное значение (999999)
	if s.counters[key] > 999999 {
		return "", fmt.Errorf("STAN counter overflow for key: %s", key)
	}

	// Форматируем как 6-значный номер
	return fmt.Sprintf("%06d", s.counters[key]), nil
}

// GetCurrentSTAN возвращает текущий STAN без увеличения счетчика
func (s *STAN) GetCurrentSTAN(transactionType, senderType, recipientType string, date time.Time) (string, error) {
	key := s.generateKey(transactionType, senderType, recipientType, date)

	s.mu.RLock()
	defer s.mu.RUnlock()

	counter, exists := s.counters[key]
	if !exists {
		return "000001", nil
	}

	return fmt.Sprintf("%06d", counter), nil
}

// ResetCounters сбрасывает счетчики
func (s *STAN) ResetCounters() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.counters = make(map[string]int)
}

// generateKey создает ключ для мапы счетчиков
func (s *STAN) generateKey(transactionType, senderType, recipientType string, date time.Time) string {
	return fmt.Sprintf("%s_%s_%s %s", date.Format(s.dateFormat), transactionType, senderType, recipientType)
}
