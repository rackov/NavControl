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
func (s *GRPCServer) GetServiceManager(context.Context, *emptypb.Empty) (*proto.ServiceManager, error) {
	return &proto.ServiceManager{
		PortSm:      int32(50001),
		TypeSm:      "RETRANSLATOR",
		IpBroker:    "192.168.194.242",
		PortBroker:  4222,
		TopicBroker: "nav.*",
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
