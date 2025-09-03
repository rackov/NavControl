// pkg/arnavi/arnavi.go
package arnavi

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/rackov/NavControl/pkg/logger"
	"github.com/rackov/NavControl/pkg/models"
	"github.com/rackov/NavControl/services/receiver/internal/protocol"
)

// ClientInfo содержит информацию о подключенном клиенте
type ClientInfo struct {
	ID          string
	RemoteAddr  string
	ConnectTime string
	Protocol    string
}

// ArnaviProtocol реализует интерфейс NavigationProtocol для протокола Arnavi
type ArnaviProtocol struct {
	listener    net.Listener
	ctx         context.Context
	cancel      context.CancelFunc
	clients     map[string]ClientInfo
	clientsMu   sync.Mutex
	connections map[net.Conn]struct{}
	connMu      sync.Mutex
	logger      *logger.Logger
}

// NewArnaviProtocol создает новый экземпляр протокола Arnavi
func NewArnaviProtocol(log *logger.Logger) protocol.NavigationProtocol {
	ctx, cancel := context.WithCancel(context.Background())
	return &ArnaviProtocol{
		ctx:         ctx,
		cancel:      cancel,
		clients:     make(map[string]ClientInfo),
		connections: make(map[net.Conn]struct{}),
		logger:      log,
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

	a.logger.Infof("%s protocol started on port %d", a.GetName(), port)

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
				a.logger.Errorf("Error accepting connection: %v", err)
				continue
			}

			// Добавляем соединение в список
			a.connMu.Lock()
			a.connections[conn] = struct{}{}
			a.connMu.Unlock()

			// Создаем информацию о клиенте
			clientID := fmt.Sprintf("%s-%d", conn.RemoteAddr().String(), time.Now().Unix())
			clientInfo := ClientInfo{
				ID:          clientID,
				RemoteAddr:  conn.RemoteAddr().String(),
				Protocol:    a.GetName(),
				ConnectTime: time.Now().Format(time.RFC3339),
			}

			// Добавляем клиента в список
			a.clientsMu.Lock()
			a.clients[clientID] = clientInfo
			a.clientsMu.Unlock()

			// Обрабатываем соединение в отдельной горутине
			go a.handleConnection(conn, clientID, dataChan)
		}
	}
}

// handleConnection обрабатывает одно соединение
func (a *ArnaviProtocol) handleConnection(conn net.Conn, clientID string, dataChan chan<- models.NavRecord) {
	defer func() {
		conn.Close()

		// Удаляем соединение из списка
		a.connMu.Lock()
		delete(a.connections, conn)
		a.connMu.Unlock()

		// Удаляем клиента из списка
		a.clientsMu.Lock()
		delete(a.clients, clientID)
		a.clientsMu.Unlock()

		a.logger.Infof("Client %s disconnected", clientID)
	}()

	// В реальном приложении здесь был бы парсинг данных из протокола Arnavi
	// Для примера просто создадим тестовую запись и отправим в канал
	buf := make([]byte, 1024)
	conn.Read(buf)
	record := models.NavRecord{
		Client:   1,
		PacketID: 1,
	}

	dataChan <- record
	a.logger.Infof("Sent test record from client %s", clientID)
}

// Stop останавливает TCP-сервер
func (a *ArnaviProtocol) Stop() error {
	// Отменяем контекст
	a.cancel()

	// Закрываем все активные соединения
	a.connMu.Lock()
	for conn := range a.connections {
		conn.Close()
	}
	a.connections = make(map[net.Conn]struct{})
	a.connMu.Unlock()

	// Закрываем слушатель
	if a.listener != nil {
		if err := a.listener.Close(); err != nil {
			return fmt.Errorf("error closing listener: %v", err)
		}
	}

	a.logger.Infof("%s protocol stopped", a.GetName())
	return nil
}

// GetClients возвращает список подключенных клиентов
func (a *ArnaviProtocol) GetClients() []ClientInfo {
	a.clientsMu.Lock()
	defer a.clientsMu.Unlock()

	clients := make([]ClientInfo, 0, len(a.clients))
	for _, client := range a.clients {
		clients = append(clients, client)
	}

	return clients
}

// DisconnectClient отключает клиента по ID
func (a *ArnaviProtocol) DisconnectClient(clientID string) error {
	a.clientsMu.Lock()
	client, exists := a.clients[clientID]
	a.clientsMu.Unlock()

	if !exists {
		return fmt.Errorf("client %s not found", clientID)
	}

	// Ищем соединение клиента и закрываем его
	a.connMu.Lock()
	defer a.connMu.Unlock()

	for conn := range a.connections {
		if conn.RemoteAddr().String() == client.RemoteAddr {
			conn.Close()
			return nil
		}
	}

	return fmt.Errorf("connection for client %s not found", clientID)
}
