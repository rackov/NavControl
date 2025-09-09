package retgrpc

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/rackov/NavControl/pkg/logger"
	"github.com/rackov/NavControl/pkg/manager"
	"github.com/rackov/NavControl/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/emptypb"
)

// GRPCServer реализует gRPC сервер
type GRPCServer struct {
	// pm     *PortManager
	logger *logger.Logger

	proto.UnimplementedServiceInfoServer
}

func NewGRPCServer(l *logger.Logger) *GRPCServer {
	return &GRPCServer{
		logger: l,
	}
}
func (s *GRPCServer) GetServiceManager(context.Context, *emptypb.Empty) (*proto.ServiceManager, error) {
	return &proto.ServiceManager{
		PortSm:      int32(50001),
		TypeSm:      manager.StrToServiceManagerType("retranslator"),
		IpBroker:    "192.168.194.242",
		PortBroker:  4222,
		TopicBroker: "nav.*",
		Active:      true,
		LogLevel:    logger.StrToLoglevel("info"),
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

	// Включаем reflection API для отладки
	reflection.Register(grpcServer)

	s.logger.Infof("gRPC server started on port %d", port)

	return grpcServer.Serve(lis)
}
