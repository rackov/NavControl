package mgrpc

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/rackov/NavControl/pkg/logger"
	"github.com/rackov/NavControl/proto"
	"github.com/rackov/NavControl/services/writer/internal/wrdbnats"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/emptypb"
)

// GRPCServer реализует gRPC сервер
type GRPCServer struct {
	logger *logger.Logger
	wrman  *wrdbnats.HubServer

	proto.UnimplementedServiceInfoServer
	proto.UnimplementedWriteControlServer
}

func NewGRPCServer(wrman *wrdbnats.HubServer) *GRPCServer {
	return &GRPCServer{
		wrman:  wrman,
		logger: wrman.GetLogger(),
	}
}
func (s *GRPCServer) GetServiceManager(context.Context, *emptypb.Empty) (*proto.ServiceManager, error) {
	conf := s.wrman.GetConfig()
	return &proto.ServiceManager{
		PortSm:      int32(conf.GrpcPort),
		TypeSm:      "WRITER",
		IpBroker:    conf.NatsAddress,
		PortBroker:  4222,
		TopicBroker: conf.NatsTopic,
		Active:      true,
		LogLevel:    conf.LogConfig.LogLevel,
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
	proto.RegisterWriteControlServer(grpcServer, s)

	// Включаем reflection API для отладки
	reflection.Register(grpcServer)

	s.logger.Infof("gRPC server started on port %d", port)

	return grpcServer.Serve(lis)
}

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
	return &proto.IsEmptyVar{
		IsEmpty: true,
	}, nil

}

// Writer

// func (s *GRPCServer) AddWrite(ctx context.Context, sw *proto.WriteService) (*proto.StateServ, error) {
// 	// s.mu.Lock()
// 	// defer s.mu.Unlock()
// 	f := -1
// 	for i, c := range s.wrman.ListService() {
// 		if (c.IdSm == sw.GetIdSm()) && (c.IdWriter == sw.GetIdWriter()) {
// 			f = i
// 			break
// 		}
// 	}
// 	if f != -1 {
// 		return &tgrpc.StateServ{Message: fmt.Sprintf("Repiat Idsm:%d  IdWrite:%d ", sw.GetIdSm(), sw.GetIdWriter())},
// 			fmt.Errorf("Repiat Idsm:%d  IdWrite:%d ", sw.GetIdSm(), sw.GetIdWriter())
// 	}
// 	err := s.wrman.AddService(sw)
// 	if err == nil {

// 		s.ServList = append(s.ServList, sw)
// 		s.save(Fname)
// 	}

// 	return &tgrpc.StateServ{Message: fmt.Sprintf("mes: %v", err)}, err
// }
