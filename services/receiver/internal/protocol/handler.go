package protocol

import (
	"github.com/rackov/NavControl/pkg/models"
)

// NavigationProtocol определяет контракт для всех навигационных протоколов.
// Любой протокол (Arnavi, EGTS, NDTP) должен реализовать все эти методы.
type NavigationProtocol interface {
	// GetName возвращает имя протокола (например, "EGTS").
	GetName() string

	// Start запускает TCP-сервер для прослушивания входящих соединений.
	// Принимает канал, в который будут отправляться распарсенные данные.
	Start(port int, nc models.NatsConf) error

	// Stop останавливает TCP-сервер и освобождает ресурсы.
	Stop() error
}
