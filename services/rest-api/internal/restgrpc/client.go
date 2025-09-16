package restgrpc

import (
	"context"
	"fmt"
	"time"

	"github.com/rackov/NavControl/pkg/config"
	"github.com/rackov/NavControl/pkg/logger"
	"github.com/rackov/NavControl/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Client struct {
	info   config.ServiceManager
	conn   *grpc.ClientConn
	logger *logger.Logger // Добавляем поле для логгера

	// Общий клиент для всех сервисов
	infoClient proto.ServiceInfoClient

	// Специфичные клиенты (инициализируются в зависимости от типа сервиса)
	receiverClient proto.ReceiverControlClient // только для RECEIVER
	// writerClient       proto.WriteControlClient        // только для WRITER
	// retranslatorClient proto.RetranslatorControlClient // только для RETRANSLATOR
}

func (c *Client) connect() error {
	// Проверяем текущее состояние подключения
	if c.conn != nil {
		state := c.conn.GetState()
		if state != connectivity.Shutdown && state != connectivity.TransientFailure {
			// Соединение активно, не нужно переподключаться
			return nil
		} else {
			// Закрываем старое подключение
			c.conn.Close()
			c.logger.Warnf("Connection is in state %s. Will reconnect...", state.String())
		}
	}

	var err error

	addr := fmt.Sprintf("%s:%d", c.info.IpSm, c.info.PortSm)

	c.logger.Infof("Connecting to gRPC server at %s", addr)

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	c.conn, err = grpc.NewClient(addr, opts...)
	if err != nil {
		c.logger.Errorf("Failed to connect to gRPC server: %v", err)
		return fmt.Errorf("failed to connect to gRPC server: %v", err)
	}

	c.logger.Infof("Successfully connected to gRPC server at %s", addr)

	// Обновляем клиенты с новым подключением
	c.infoClient = proto.NewServiceInfoClient(c.conn)

	// Инициализируем специфические клиенты в зависимости от типа сервиса
	switch c.info.TypeSm {
	case "RECEIVER":
		c.receiverClient = proto.NewReceiverControlClient(c.conn)
		// case "WRITER":
		//     c.writerClient = proto.NewWriteControlClient(c.conn)
		// case "RETRANSLATOR":
		//     c.retranslatorClient = proto.NewRetranslatorControlClient(c.conn)
	}

	return nil
}

func NewClient(service config.ServiceManager, logger *logger.Logger) (*Client, error) {
	client := &Client{
		info:   service,
		logger: logger,
	}

	// Используем общий метод connect для установления соединения
	if err := client.connect(); err != nil {
		logger.Errorf("Failed to connect to gRPC server: %v", err)
		return nil, fmt.Errorf("failed to connect to gRPC server: %v", err)
	}

	logger.Infof("Successfully created gRPC client for %s", service.Name)

	return client, nil
}

func (c *Client) GetServiceManager(ctx context.Context) (*proto.ServiceManager, error) {
	c.logger.Info("Calling GetServiceManager gRPC method")

	// Используем общий метод connect для проверки и восстановления соединения при необходимости
	if err := c.connect(); err != nil {
		c.logger.Errorf("Failed to ensure connection: %v", err)
		return &proto.ServiceManager{
			Name:        c.info.Name,
			IpSm:        c.info.IpSm,
			PortSm:      int32(c.info.PortSm),
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

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	manager, err := c.infoClient.GetServiceManager(ctx, &emptypb.Empty{})
	if err != nil {
		c.logger.Errorf("Failed to call GetServiceManager: %v", err)
		return &proto.ServiceManager{
			Name:        c.info.Name,
			IpSm:        c.info.IpSm,
			PortSm:      int32(c.info.PortSm),
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
