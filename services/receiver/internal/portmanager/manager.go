package portmanager

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"sync"

	"github.com/rackov/NavControl/pkg/models"
	"github.com/rackov/NavControl/proto"
	"github.com/rackov/NavControl/services/receiver/internal/handler/arnavi"
	"github.com/rackov/NavControl/services/receiver/internal/protocol"
	"google.golang.org/grpc"
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
	mu       sync.RWMutex
	ports    map[int32]*PortInfo
	dataChan chan models.NavRecord
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewPortManager создает новый менеджер портов
func NewPortManager() *PortManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &PortManager{
		ports:    make(map[int32]*PortInfo),
		dataChan: make(chan models.NavRecord, 100),
		ctx:      ctx,
		cancel:   cancel,
	}
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
		protocolInstance = arnavi.NewArnaviProtocol()
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

	pm.ports[portNumber] = portInfo
	log.Printf("Added port %d with protocol %s", portNumber, protocolName)

	// Если порт должен быть активным, запускаем его
	if active {
		return protocolInstance.Start(int(portNumber), pm.dataChan)
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

	err := portInfo.ProtocolInstance.Start(int(portNumber), pm.dataChan)
	if err != nil {
		return err
	}

	pm.mu.Lock()
	portInfo.Active = true
	pm.mu.Unlock()

	log.Printf("Started port %d", portNumber)
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

	log.Printf("Stopped port %d", portNumber)
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
	log.Printf("Deleted port %d", portNumber)
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
				log.Printf("Disconnected client %s", clientID)
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
				log.Printf("Error stopping protocol on port %d: %v", portNumber, err)
			}
		}
	}

	close(pm.dataChan)
	log.Println("Port manager stopped")
}

// GetDataChan возвращает канал для получения навигационных данных
func (pm *PortManager) GetDataChan() <-chan models.NavRecord {
	return pm.dataChan
}

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

// GRPCServer реализует gRPC сервер
type GRPCServer struct {
	pm *PortManager
	proto.UnimplementedReceiverControlServer
}

// NewGRPCServer создает новый gRPC сервер
func NewGRPCServer(pm *PortManager) *GRPCServer {
	return &GRPCServer{
		pm: pm,
	}
}

// GetActiveConnectionsCount возвращает количество активных подключений
func (s *GRPCServer) GetActiveConnectionsCount(ctx context.Context, req *proto.GetClientsRequest) (*proto.GetActiveCount, error) {
	count, err := s.pm.GetActiveConnectionsCount(req.PortReceiver)
	if err != nil {
		return nil, err
	}

	return &proto.GetActiveCount{Count: count}, nil
}

// GetConnectedClients возвращает список подключенных клиентов
func (s *GRPCServer) GetConnectedClients(ctx context.Context, req *proto.GetClientsRequest) (*proto.GetClientsResponse, error) {
	clients, err := s.pm.GetConnectedClients(req.PortReceiver)
	if err != nil {
		return nil, err
	}

	return &proto.GetClientsResponse{Clients: clients}, nil
}

// DisconnectClient отключает клиента
func (s *GRPCServer) DisconnectClient(ctx context.Context, req *proto.DisconnectClientRequest) (*proto.DisconnectClientResponse, error) {
	err := s.pm.DisconnectClient(req.ClientId)
	if err != nil {
		return &proto.DisconnectClientResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &proto.DisconnectClientResponse{
		Success: true,
		Message: "Client disconnected successfully",
	}, nil
}

// OpenPort открывает (активирует) порт
func (s *GRPCServer) OpenPort(ctx context.Context, req *proto.PortIdentifier) (*proto.PortOperationResponse, error) {
	err := s.pm.StartPort(req.PortReceiver)
	if err != nil {
		return &proto.PortOperationResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &proto.PortOperationResponse{
		Success: true,
		Message: "Port opened successfully",
	}, nil
}

// ClosePort закрывает (деактивирует) порт
func (s *GRPCServer) ClosePort(ctx context.Context, req *proto.PortIdentifier) (*proto.PortOperationResponse, error) {
	err := s.pm.StopPort(req.PortReceiver)
	if err != nil {
		return &proto.PortOperationResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &proto.PortOperationResponse{
		Success: true,
		Message: "Port closed successfully",
	}, nil
}

// AddPort добавляет новый порт в конфигурацию
func (s *GRPCServer) AddPort(ctx context.Context, req *proto.PortDefinition) (*proto.PortOperationResponse, error) {
	err := s.pm.AddPort(req.PortReceiver, req.Protocol, req.IsActive, req.Name)
	if err != nil {
		return &proto.PortOperationResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &proto.PortOperationResponse{
		Success: true,
		Message: "Port added successfully",
	}, nil
}

// DeletePort удаляет порт из конфигурации
func (s *GRPCServer) DeletePort(ctx context.Context, req *proto.PortIdentifier) (*proto.PortOperationResponse, error) {
	err := s.pm.DeletePort(req.PortReceiver)
	if err != nil {
		return &proto.PortOperationResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &proto.PortOperationResponse{
		Success: true,
		Message: "Port deleted successfully",
	}, nil
}

// StartGRPCServer запускает gRPC сервер
func (s *GRPCServer) StartGRPCServer(port int) error {
	lis, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	proto.RegisterReceiverControlServer(grpcServer, s)

	log.Printf("gRPC server started on port %d", port)

	return grpcServer.Serve(lis)
}
