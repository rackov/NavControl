// pkg/arnavi/arnavi.go
package arnavi

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/rackov/NavControl/pkg/models"
	"github.com/rackov/NavControl/services/receiver/internal/protocol"
)

// ArnaviProtocol реализует интерфейс NavigationProtocol для протокола Arnavi
type ArnaviProtocol struct {
	listener net.Listener
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewArnaviProtocol создает новый экземпляр протокола Arnavi
func NewArnaviProtocol() protocol.NavigationProtocol {
	ctx, cancel := context.WithCancel(context.Background())
	return &ArnaviProtocol{
		ctx:    ctx,
		cancel: cancel,
	}
}

// GetName возвращает имя протокола
func (a *ArnaviProtocol) GetName() string {
	return "Arnavi"
}

// Start запускает TCP-сервер для приема данных
func (a *ArnaviProtocol) Start(port int, dataChan chan<- models.NavRecord) error {
	var err error
	a.listener, err = net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %v", port, err)
	}

	log.Printf("%s protocol started on port %d", a.GetName(), port)

	// Запускаем обработчик соединений в отдельной горутине
	go a.handleConnections(dataChan)

	return nil
}

// handleConnections обрабатывает входящие соединения
func (a *ArnaviProtocol) handleConnections(dataChan chan<- models.NavRecord) {
	for {
		select {
		case <-a.ctx.Done():
			return
		default:
			conn, err := a.listener.Accept()
			if err != nil {
				// Проверяем, не был ли сервер остановлен
				if a.ctx.Err() != nil {
					return
				}
				log.Printf("Error accepting connection: %v", err)
				continue
			}

			// Обрабатываем соединение в отдельной горутине
			go a.handleConnection(conn, dataChan)
		}
	}
}

// handleConnection обрабатывает одно соединение
func (a *ArnaviProtocol) handleConnection(conn net.Conn, dataChan chan<- models.NavRecord) {
	defer conn.Close()

	// В реальном приложении здесь был бы парсинг данных из протокола Arnavi
	// Для примера просто создадим тестовую запись и отправим в канал
	record := models.NavRecord{
		Client:              1,
		PacketID:            1,
		NavigationTimestamp: 1,
		Longitude:           1,
		Latitude:            1,
	}

	dataChan <- record
	log.Printf("Sent test record from %s", conn.RemoteAddr())
}

// Stop останавливает TCP-сервер
func (a *ArnaviProtocol) Stop() error {
	// Отменяем контекст
	a.cancel()

	// Закрываем слушатель
	if a.listener != nil {
		if err := a.listener.Close(); err != nil {
			return fmt.Errorf("error closing listener: %v", err)
		}
	}

	log.Printf("%s protocol stopped", a.GetName())
	return nil
}
