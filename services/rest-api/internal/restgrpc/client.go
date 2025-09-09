package restgrpc

import (
	"context"
	"fmt"
	"time"

	"github.com/rackov/NavControl/pkg/config"
	"github.com/rackov/NavControl/pkg/logger"
	"github.com/rackov/NavControl/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Client struct {
	info   config.ServiceManager
	conn   *grpc.ClientConn
	client proto.ServiceInfoClient
	logger *logger.Logger // Добавляем поле для логгера
}

func NewClient(server config.ServiceManager, logger *logger.Logger) (*Client, error) {
	addr := fmt.Sprintf("%s:%d", server.IpSm, server.PortSm)

	logger.Infof("Connecting to gRPC server at %s", addr)

	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Errorf("Failed to connect to gRPC server: %v", err)
		return nil, fmt.Errorf("failed to connect to gRPC server: %v", err)
	}

	logger.Infof("Successfully connected to gRPC server at %s", addr)

	return &Client{
		info:   server,
		conn:   conn,
		client: proto.NewServiceInfoClient(conn),
		logger: logger, // Сохраняем логгер в структуре
	}, nil
}

func (c *Client) GetServiceManager(ctx context.Context) (*proto.ServiceManager, error) {
	c.logger.Info("Calling GetServiceManager gRPC method")

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	manager, err := c.client.GetServiceManager(ctx, &emptypb.Empty{})
	if err != nil {
		c.logger.Errorf("Failed to call GetServiceManager: %v", err)
		return &proto.ServiceManager{
			Name:        c.info.Name,
			IpSm:        c.info.IpSm,
			PortSm:      int32(c.info.PortSm),
			IdSm:        int32(c.info.IdSm),
			TypeSm:      c.info.TypeSm,
			IpBroker:    c.info.IpBroker,
			PortBroker:  int32(c.info.PortBroker),
			TopicBroker: c.info.TopicBroker,
			Active:      false,
			Status:      "offline",
			Description: c.info.Description,
			LogLevel:    c.info.LogLevel,
		}, err
	}

	c.logger.Infof("Successfully received ServiceManager: %+v", manager)
	return manager, nil
}

func (c *Client) Close() error {
	c.logger.Info("Closing gRPC connection")
	err := c.conn.Close()
	if err != nil {
		c.logger.Errorf("Failed to close gRPC connection: %v", err)
		return err
	}
	c.logger.Info("gRPC connection closed successfully")
	return nil
}
