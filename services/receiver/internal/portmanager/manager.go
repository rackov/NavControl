package portmanager

import (
	"context"
	"fmt"
	"time"

	"sync"

	"github.com/nats-io/nats.go"
	"github.com/rackov/NavControl/pkg/config"
	"github.com/rackov/NavControl/pkg/logger"
	"github.com/rackov/NavControl/pkg/models"
	"github.com/rackov/NavControl/proto"
	"github.com/rackov/NavControl/services/receiver/internal/handler/arnavi"
	"github.com/rackov/NavControl/services/receiver/internal/handler/egts"
	"github.com/rackov/NavControl/services/receiver/internal/protocol"
)

// PortInfo содержит информацию о порте и его состоянии
type PortInfo struct {
	PortNumber       int32
	Protocol         string
	Active           bool
	Name             string
	ProtocolInstance protocol.NavigationProtocol
}

// PortManager управляет навигационными протоколами на разных портах
type PortManager struct {
	mu    sync.RWMutex
	muCfg sync.RWMutex
	ports map[int32]*PortInfo
	// dataChan chan models.NavRecord
	ctx    context.Context
	cancel context.CancelFunc
	cfg    *config.Services
	logger *logger.Logger
	nc     *nats.Conn
}

// NewPortManager создает новый менеджер портов
func NewPortManager(cfg *config.Services) *PortManager {
	ctx, cancel := context.WithCancel(context.Background())
	log, err := logger.NewLogger(cfg.LogConfig)
	if err != nil {
		fmt.Println(err)
	}
	return &PortManager{
		ports: make(map[int32]*PortInfo),
		// dataChan: make(chan models.NavRecord, 100),
		ctx:    ctx,
		cancel: cancel,
		logger: log,
		cfg:    cfg,
	}
}

// Start запускает менеджер портов
func (pm *PortManager) Start() error {
	nc, err := nats.Connect(pm.cfg.NatsAddress)
	if err != nil {
		return err
	}
	pm.logger.Info("NATS connected")
	pm.nc = nc

	// запуск портов согласно конфигурации
	go func() {
		for {
			select {
			case <-pm.ctx.Done():
				pm.logger.Info("PortManager stopped")
				pm.Stop()
				return
			// case data := <-pm.dataChan:
			// 	// Отправляем данные на NATS
			// 	// err := pm.nc.Publish(pm.natsTopic, data)
			// 	pm.logger.Infof("Publishing data to NATS: %v", data)
			default:
				if pm.nc == nil || !pm.nc.IsConnected() {
					pm.logger.Errorf("NATS connection lost, reconnecting...")
					pm.reconect()
				}
				time.Sleep(time.Duration(pm.cfg.NatsTimeOut) * time.Second)

			}
		}
	}()
	return nil

}
func (pm *PortManager) reconect() {
	pm.muCfg.Lock()
	for i, r := range pm.cfg.Receivers {
		if r.Active {
			pm.StopPort(r.Port)
			pm.cfg.Receivers[i].Status = "disconnected Nats"
		}
	}
	pm.muCfg.Unlock()

	nc, err := nats.Connect(pm.cfg.NatsAddress)
	if err != nil {
		pm.logger.Errorf("Error connecting to NATS: %v", err)
		return
	}
	pm.nc = nc
	pm.logger.Info("NATS connected")
	pm.muCfg.Lock()
	for i, r := range pm.cfg.Receivers {
		if r.Active {
			pm.StartPort(r.Port)
			pm.cfg.Receivers[i].Status = "ok"
		}
	}
	pm.muCfg.Unlock()

}

// ListPorts возвращает список всех портов с их состоянием
func (pm *PortManager) ListPorts() ([]*proto.PortDefinition, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	ports := make([]*proto.PortDefinition, 0, len(pm.ports))

	for _, portInfo := range pm.ports {
		// Получаем количество подключений для порта
		var connectionsCount int32 = 0
		if portInfo.Active {
			if arnaviProto, ok := portInfo.ProtocolInstance.(*arnavi.ArnaviProtocol); ok {
				clients := arnaviProto.GetClients()
				connectionsCount = int32(len(clients))
			}
		}

		ports = append(ports, &proto.PortDefinition{
			PortReceiver:     portInfo.PortNumber,
			Protocol:         portInfo.Protocol,
			IsActive:         portInfo.Active,
			Name:             portInfo.Name,
			ConnectionsCount: connectionsCount,
		})
	}

	return ports, nil
}

// AddPort добавляет порт в конфигурацию
func (pm *PortManager) AddPort(portNumber int32, protocolName string, active bool, name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Проверяем, не занят ли уже этот порт
	if _, exists := pm.ports[portNumber]; exists {
		return fmt.Errorf("port %d is already in use", portNumber)
	}

	// Создаем экземпляр протокола в зависимости от имени
	var protocolInstance protocol.NavigationProtocol
	switch protocolName {
	case "Arnavi":
		protocolInstance = arnavi.NewArnaviProtocol(pm.logger)
	case "EGTS":
		protocolInstance = egts.NewEgtsProtocol(pm.logger)
	default:
		return fmt.Errorf("unsupported protocol: %s", protocolName)
	}

	// Создаем информацию о порте
	portInfo := &PortInfo{
		PortNumber:       portNumber,
		Protocol:         protocolName,
		Active:           active,
		Name:             name,
		ProtocolInstance: protocolInstance,
	}
	pm.muCfg.Lock()
	pm.cfg.Receivers = append(pm.cfg.Receivers, config.Receiver{
		Port:     portNumber,
		Active:   active,
		Protocol: protocolName,
		Name:     name,
		Status:   "ok",
	})
	pm.muCfg.Unlock()

	pm.ports[portNumber] = portInfo
	pm.logger.Infof("Added port %d with protocol %s", portNumber, protocolName)
	nat := models.NatsConf{Nc: pm.nc, Topic: pm.cfg.NatsTopic}
	// Если порт должен быть активным, запускаем его
	if active {
		return protocolInstance.Start(int(portNumber), nat)
	}

	return nil
}

// StartPort запускает протокол на указанном порту
func (pm *PortManager) StartPort(portNumber int32) error {
	pm.mu.RLock()
	portInfo, exists := pm.ports[portNumber]
	pm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("port %d not found", portNumber)
	}

	if portInfo.Active {
		return fmt.Errorf("port %d is already active", portNumber)
	}
	nat := models.NatsConf{Nc: pm.nc, Topic: pm.cfg.NatsTopic}

	err := portInfo.ProtocolInstance.Start(int(portNumber), nat)
	if err != nil {
		return err
	}

	pm.mu.Lock()
	portInfo.Active = true
	pm.mu.Unlock()

	pm.logger.Infof("Started port %d", portNumber)
	return nil
}

// StopPort останавливает протокол на указанном порту
func (pm *PortManager) StopPort(portNumber int32) error {
	pm.mu.RLock()
	portInfo, exists := pm.ports[portNumber]
	pm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("port %d not found", portNumber)
	}

	if !portInfo.Active {
		return fmt.Errorf("port %d is already inactive", portNumber)
	}

	err := portInfo.ProtocolInstance.Stop()
	if err != nil {
		return err
	}

	pm.mu.Lock()
	portInfo.Active = false
	pm.mu.Unlock()

	pm.logger.Infof("Stopped port %d", portNumber)
	return nil
}

// DeletePort удаляет порт из конфигурации
func (pm *PortManager) DeletePort(portNumber int32) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	portInfo, exists := pm.ports[portNumber]
	if !exists {
		return fmt.Errorf("port %d not found", portNumber)
	}

	// Если порт активен, останавливаем его
	if portInfo.Active {
		err := portInfo.ProtocolInstance.Stop()
		if err != nil {
			return fmt.Errorf("failed to stop port %d: %v", portNumber, err)
		}
	}

	delete(pm.ports, portNumber)
	pm.logger.Infof("Deleted port %d", portNumber)
	return nil
}

// GetActiveConnectionsCount возвращает количество активных подключений
func (pm *PortManager) GetActiveConnectionsCount(portNumber int32) (int32, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if portNumber != 0 {
		// Если указан конкретный порт
		portInfo, exists := pm.ports[portNumber]
		if !exists {
			return 0, fmt.Errorf("port %d not found", portNumber)
		}

		if !portInfo.Active {
			return 0, nil
		}

		// Получаем количество клиентов для порта
		if arnaviProto, ok := portInfo.ProtocolInstance.(*arnavi.ArnaviProtocol); ok {
			clients := arnaviProto.GetClients()
			return int32(len(clients)), nil
		}
		if egtsProto, ok := portInfo.ProtocolInstance.(*egts.EgtsProtocol); ok {
			clients := egtsProto.GetClients()
			return int32(len(clients)), nil
		}

		return 0, fmt.Errorf("failed to get clients count for port %d", portNumber)
	}

	// Если порт не указан, считаем для всех портов
	var total int32
	for _, portInfo := range pm.ports {
		if !portInfo.Active {
			continue
		}

		if arnaviProto, ok := portInfo.ProtocolInstance.(*arnavi.ArnaviProtocol); ok {
			clients := arnaviProto.GetClients()
			total += int32(len(clients))
		}
		if egtsProto, ok := portInfo.ProtocolInstance.(*egts.EgtsProtocol); ok {
			clients := egtsProto.GetClients()
			total += int32(len(clients))
		}
	}

	return total, nil
}

// GetConnectedClients возвращает список подключенных клиентов
func (pm *PortManager) GetConnectedClients(portNumber int32) ([]*proto.ClientInfo, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var clients []*proto.ClientInfo

	if portNumber != 0 {
		// Если указан конкретный порт
		portInfo, exists := pm.ports[portNumber]
		if !exists {
			return nil, fmt.Errorf("port %d not found", portNumber)
		}

		if !portInfo.Active {
			return clients, nil
		}

		// Получаем клиентов для порта
		if arnaviProto, ok := portInfo.ProtocolInstance.(*arnavi.ArnaviProtocol); ok {
			arnaviClients := arnaviProto.GetClients()
			for _, client := range arnaviClients {
				clients = append(clients, &proto.ClientInfo{
					Id:           client.ID,
					Address:      client.RemoteAddr,
					ConnectTime:  client.ConnectTime,
					ProtocolName: client.Protocol,
				})
			}
		}
		if egtsProto, ok := portInfo.ProtocolInstance.(*egts.EgtsProtocol); ok {
			egtsClients := egtsProto.GetClients()
			for _, client := range egtsClients {
				clients = append(clients, &proto.ClientInfo{
					Id:           client.ID,
					Address:      client.RemoteAddr,
					ConnectTime:  client.ConnectTime,
					ProtocolName: client.Protocol,
				})
			}
		}
		return clients, nil
	}

	// Если порт не указан, собираем клиентов со всех портов
	for _, portInfo := range pm.ports {
		if !portInfo.Active {
			continue
		}

		if arnaviProto, ok := portInfo.ProtocolInstance.(*arnavi.ArnaviProtocol); ok {
			arnaviClients := arnaviProto.GetClients()
			for _, client := range arnaviClients {
				clients = append(clients, &proto.ClientInfo{
					Id:           client.ID,
					Address:      client.RemoteAddr,
					ConnectTime:  client.ConnectTime,
					ProtocolName: client.Protocol,
				})
			}
		}
		if egtsProto, ok := portInfo.ProtocolInstance.(*egts.EgtsProtocol); ok {
			egtsClients := egtsProto.GetClients()
			for _, client := range egtsClients {
				clients = append(clients, &proto.ClientInfo{
					Id:           client.ID,
					Address:      client.RemoteAddr,
					ConnectTime:  client.ConnectTime,
					ProtocolName: client.Protocol,
				})
			}
		}

	}

	return clients, nil
}

// DisconnectClient отключает клиента
func (pm *PortManager) DisconnectClient(clientID string) error {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Ищем клиента на всех портах
	for _, portInfo := range pm.ports {
		if !portInfo.Active {
			continue
		}

		if arnaviProto, ok := portInfo.ProtocolInstance.(*arnavi.ArnaviProtocol); ok {
			err := arnaviProto.DisconnectClient(clientID)
			if err == nil {
				pm.logger.Infof("Disconnected client %s", clientID)
				return nil
			}
		}
		if egtsProto, ok := portInfo.ProtocolInstance.(*egts.EgtsProtocol); ok {
			err := egtsProto.DisconnectClient(clientID)
			if err == nil {
				pm.logger.Infof("Disconnected client %s", clientID)
				return nil
			}
		}
	}

	return fmt.Errorf("client %s not found", clientID)

}

// Stop останавливает все протоколы и освобождает ресурсы
func (pm *PortManager) Stop() {
	pm.cancel()

	pm.mu.Lock()
	defer pm.mu.Unlock()

	for portNumber, portInfo := range pm.ports {
		if portInfo.Active {
			if err := portInfo.ProtocolInstance.Stop(); err != nil {
				pm.logger.Infof("Error stopping protocol on port %d: %v", portNumber, err)
			}
		}
	}

	pm.logger.Infof("Port manager stopped")
}

// GetDataChan возвращает канал для получения навигационных данных
// func (pm *PortManager) GetDataChan() <-chan models.NavRecord {
// 	return pm.dataChan
// }

// GetPortsInfo возвращает информацию о всех портах
func (pm *PortManager) GetPortsInfo() []*proto.PortDefinition {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	ports := make([]*proto.PortDefinition, 0, len(pm.ports))
	for _, portInfo := range pm.ports {
		ports = append(ports, &proto.PortDefinition{
			PortReceiver: portInfo.PortNumber,
			Protocol:     portInfo.Protocol,
			IsActive:     portInfo.Active,
			Name:         portInfo.Name,
		})
	}

	return ports
}
