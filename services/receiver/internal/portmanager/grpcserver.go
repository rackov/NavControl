package portmanager

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/rackov/NavControl/pkg/logger"
	"github.com/rackov/NavControl/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/emptypb"
)

// GRPCServer реализует gRPC сервер
type GRPCServer struct {
	pm     *PortManager
	logger *logger.Logger
	proto.UnimplementedReceiverControlServer
	proto.UnimplementedServiceInfoServer
}

func (s *GRPCServer) GetServiceManager(context.Context, *emptypb.Empty) (*proto.ServiceManager, error) {
	s.pm.mu.Lock()
	defer s.pm.mu.Unlock()

	return s.pm.GetServiceManager()

}
func (s *GRPCServer) GetInfo(ctx context.Context, req *emptypb.Empty) (*proto.ServiceInfoResponse, error) {
	return &proto.ServiceInfoResponse{
		Name:           "RECEIVER",
		Version:        "1.0.0",
		Build:          "1",
		BuildDate:      "2025-08-09",
		GitHash:        "https://github.com/rackov/NavControl.git",
		GitBranch:      "master",
		GitState:       "clean",
		GitSummary:     "clean",
		GitDescription: "clean",
		GitUrl:         "https://github.com/rackov/NavControl.git",
		GitUser:        "rackov",
		GitEmail:       "rackov@gmail.com",
		GitRemote:      "https://github.com/rackov/NavControl.git",
	}, nil
}

// список протоколов
func (s *GRPCServer) GetProtocols(context.Context, *empty.Empty) (*proto.Protocols, error) {
	return &proto.Protocols{
		ProtocolList: []string{"EGTS", "Arnavi"},
	}, nil
}

// Получить статус порта
func (s *GRPCServer) GetPortStatus(ctx context.Context, req *proto.GetClientsRequest) (*proto.PortDefinition, error) {
	status, err := s.pm.GetPortStatus(req.PortReceiver)
	if err != nil {
		return nil, err
	}

	return s.pm.GetPortStatus(status.PortReceiver)
}

// ListPorts возвращает список всех портов с их состоянием
func (s *GRPCServer) ListPorts(ctx context.Context, req *emptypb.Empty) (*proto.ListPortsResponse, error) {
	ports, err := s.pm.ListPorts()
	if err != nil {
		s.logger.Errorf("Failed to list ports: %v", err)
		return &proto.ListPortsResponse{
			Success: false,
			Message: "Failed to list ports: " + err.Error(),
		}, nil
	}

	return &proto.ListPortsResponse{
		Ports:   ports,
		Success: true,
		Message: "Ports listed successfully",
	}, nil
}

// NewGRPCServer создает новый gRPC сервер
func NewGRPCServer(pm *PortManager) *GRPCServer {
	return &GRPCServer{
		pm:     pm,
		logger: pm.logger,
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
func (s *GRPCServer) OpenPort(ctx context.Context, req *proto.PortDefinition) (*proto.PortOperationResponse, error) {
	err := s.pm.StartPort(req.PortReceiver)
	if err != nil {
		return &proto.PortOperationResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}
	req.Active = true
	if err = s.pm.SaveConfigPort("edit", req); err != nil {
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
func (s *GRPCServer) ClosePort(ctx context.Context, req *proto.PortDefinition) (*proto.PortOperationResponse, error) {
	err := s.pm.StopPort(req.PortReceiver)
	if err != nil {
		return &proto.PortOperationResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}
	req.Active = false
	if err = s.pm.SaveConfigPort("edit", req); err != nil {
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
	req.IdReceiver = int32(s.pm.NewId() + 1)
	err := s.pm.AddPort(req)
	if err != nil {
		return &proto.PortOperationResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	if err = s.pm.SaveConfigPort("add", req); err != nil {
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
func (s *GRPCServer) DeletePort(ctx context.Context, req *proto.PortDefinition) (*proto.PortOperationResponse, error) {
	err := s.pm.DeletePort(req.PortReceiver)
	if err != nil {
		return &proto.PortOperationResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}
	if err = s.pm.SaveConfigPort("delete", req); err != nil {
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
		s.logger.Errorf("Failed to listen: %v", err)
		return fmt.Errorf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	// Регистрируем сервисы
	proto.RegisterReceiverControlServer(grpcServer, s)
	proto.RegisterServiceInfoServer(grpcServer, s)

	// Включаем reflection API для отладки
	reflection.Register(grpcServer)

	s.logger.Infof("gRPC server started on port %d", port)

	return grpcServer.Serve(lis)
}

// === Методы LoggingService ===

// GetLogLevel возвращает текущий уровень логирования
func (s *GRPCServer) GetLogLevel(ctx context.Context, _ *emptypb.Empty) (*proto.LogLevelResponse, error) {
	level := s.logger.GetLevel()
	return &proto.LogLevelResponse{
		Level:   level,
		Success: true,
		Message: "Current log level: " + level,
	}, nil
}

// SetLogLevel устанавливает уровень логирования
func (s *GRPCServer) SetLogLevel(ctx context.Context, req *proto.SetLogLevelRequest) (*proto.SetLogLevelResponse, error) {
	err := s.logger.SetLevel(req.Level)
	if err != nil {
		s.logger.Errorf("Failed to set log level: %v", err)
		return &proto.SetLogLevelResponse{
			Success: false,
		}, nil
	}

	s.logger.Infof("Log level changed to %s", req.Level)
	return &proto.SetLogLevelResponse{
		Success: true,
	}, nil
}

// ReadLogs читает логи с применением фильтров
func (s *GRPCServer) ReadLogs(ctx context.Context, req *proto.ReadLogsRequest) (*proto.ReadLogsResponse, error) {
	// Получаем логи с фильтрами

	s.logger.Info("Reading logs")
	logs, err := s.logger.ReadLogs(req.Level, req.StartDate, req.EndDate, req.Limit)
	if err != nil {
		s.logger.Errorf("Failed to read logs: %v", err)
		return &proto.ReadLogsResponse{
			Success: false,
			Message: "Failed to read logs: " + err.Error(),
		}, nil
	}

	return &proto.ReadLogsResponse{
		LogLines: logs,
		Success:  true,
		Message:  "Logs retrieved successfully",
	}, nil
}
func (s *GRPCServer) IsEmpty(ctx context.Context, _ *emptypb.Empty) (*proto.IsEmptyVar, error) {
	ports, err := s.pm.ListPorts()
	if err != nil {
		return &proto.IsEmptyVar{
			IsEmpty: true,
		}, err
	}
	if len(ports) == 0 {
		return &proto.IsEmptyVar{
			IsEmpty: true,
		}, nil
	}
	return &proto.IsEmptyVar{
		IsEmpty: false,
	}, nil
}
