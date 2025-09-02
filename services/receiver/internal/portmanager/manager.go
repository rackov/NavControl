// pkg/portmanager/portmanager.go
package portmanager

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/rackov/NavControl/pkg/models"
	"github.com/rackov/NavControl/services/receiver/internal/protocol"
)

// PortManager управляет навигационными протоколами на разных портах
type PortManager struct {
	mu       sync.RWMutex
	ports    map[int]protocol.NavigationProtocol
	dataChan chan models.NavRecord
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewPortManager создает новый менеджер портов
func NewPortManager() *PortManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &PortManager{
		ports:    make(map[int]protocol.NavigationProtocol),
		dataChan: make(chan models.NavRecord, 100),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// AddProtocol добавляет протокол на указанный порт
func (pm *PortManager) AddProtocol(port int, p protocol.NavigationProtocol) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Проверяем, не занят ли уже этот порт
	if _, exists := pm.ports[port]; exists {
		return fmt.Errorf("port %d is already in use", port)
	}

	pm.ports[port] = p
	log.Printf("Added protocol %s on port %d", p.GetName(), port)
	return nil
}

// StartProtocol запускает протокол на указанном порту
func (pm *PortManager) StartProtocol(port int) error {
	pm.mu.RLock()
	p, exists := pm.ports[port]
	pm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no protocol found on port %d", port)
	}

	return p.Start(port, pm.dataChan)
}

// StopProtocol останавливает протокол на указанном порту
func (pm *PortManager) StopProtocol(port int) error {
	pm.mu.RLock()
	p, exists := pm.ports[port]
	pm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no protocol found on port %d", port)
	}

	return p.Stop()
}

// Stop останавливает все протоколы и освобождает ресурсы
func (pm *PortManager) Stop() {
	pm.cancel()

	pm.mu.Lock()
	defer pm.mu.Unlock()

	for port, p := range pm.ports {
		if err := p.Stop(); err != nil {
			log.Printf("Error stopping protocol on port %d: %v", port, err)
		}
	}

	close(pm.dataChan)
	log.Println("Port manager stopped")
}

// GetDataChan возвращает канал для получения навигационных данных
func (pm *PortManager) GetDataChan() <-chan models.NavRecord {
	return pm.dataChan
}

// GetProtocolsInfo возвращает информацию о всех запущенных протоколах и портах
func (pm *PortManager) GetProtocolsInfo() map[int]string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	info := make(map[int]string)
	for port, p := range pm.ports {
		info[port] = p.GetName()
	}

	return info
}
