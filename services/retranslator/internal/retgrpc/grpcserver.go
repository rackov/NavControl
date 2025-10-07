package retgrpc

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/rackov/NavControl/pkg/logger"
	"github.com/rackov/NavControl/proto"
	"github.com/rackov/NavControl/services/retranslator/internal/sendegts"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/emptypb"
)

// GRPCServer реализует gRPC сервер
type GRPCServer struct {
	rst    *sendegts.SendServer
	logger *logger.Logger
	proto.UnimplementedRetranslatorControlServer
	proto.UnimplementedServiceInfoServer
}

func NewGRPCServer(l *logger.Logger, rst *sendegts.SendServer) *GRPCServer {
	return &GRPCServer{
		rst:    rst,
		logger: l,
	}
}
func (s *GRPCServer) GetInfoClient(ctx context.Context, cl *proto.SetClient) (*proto.Client, error) {
	return s.rst.GetInfoClient(cl)
}
func (s *GRPCServer) GetServiceManager(context.Context, *emptypb.Empty) (*proto.ServiceManager, error) {
	return &proto.ServiceManager{
		PortSm:      int32(50001),
		TypeSm:      "RETRANSLATOR",
		IpBroker:    "",
		PortBroker:  0,
		TopicBroker: "",
		Active:      true,
		LogLevel:    "info",
	}, nil
}

func (s *GRPCServer) StartGRPCServer(port int) error {
	lis, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		s.logger.Errorf("Failed to listen: %v", err)
		return fmt.Errorf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	// Регистрируем сервисы
	proto.RegisterServiceInfoServer(grpcServer, s)
	proto.RegisterRetranslatorControlServer(grpcServer, s)

	// Включаем reflection API для отладки
	reflection.Register(grpcServer)

	s.logger.Infof("gRPC server started on port %d", port)

	return grpcServer.Serve(lis)
}

func (s *GRPCServer) ListDevices(stx context.Context, set *proto.SetClient) (*proto.Devices, error) {
	return s.rst.ListDevices(set)
}
func (s *GRPCServer) DeleteClient(stx context.Context, set *proto.SetClient) (*proto.Client, error) {
	return s.rst.DeleteClient(set)
}
func (s *GRPCServer) AddClient(stx context.Context, cl *proto.Client) (*proto.Client, error) {

	newCl, err := s.rst.AddClient(cl)
	if err != nil {
		return cl, err
	}
	return newCl, err
}

func (s *GRPCServer) ListClient(ctx context.Context, st *empty.Empty) (*proto.Clients, error) {
	return s.rst.ListClient()

}
func (s *GRPCServer) DownClient(ctx context.Context, st *proto.SetClient) (*proto.Client, error) {

	return s.rst.StopClient(st)
}
func (s *GRPCServer) UpClient(ctx context.Context, st *proto.SetClient) (*proto.Client, error) {
	return s.rst.RunClient(st)

}

func (s *GRPCServer) IsEmpty(ctx context.Context, _ *emptypb.Empty) (*proto.IsEmptyVar, error) {
	client, err := s.rst.ListClient()
	if err != nil {
		return &proto.IsEmptyVar{
			IsEmpty: true,
		}, err
	}
	if len(client.Clients) == 0 {
		return &proto.IsEmptyVar{
			IsEmpty: true,
		}, nil
	}
	return &proto.IsEmptyVar{
		IsEmpty: false,
	}, nil
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
	logReq := logger.Filter{
		Level:     req.Level,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		Limit:     req.Limit,
		IdService: req.IdService,
		Protocol:  req.Protocol,
		Port:      req.Port,
		PosEnd:    req.PosEnd,
	}
	logs, err := s.logger.ReadLogs(logReq)
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
