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
	receiverClient     proto.ReceiverControlClient     // только для RECEIVER
	writerClient       proto.WriteControlClient        // только для WRITER
	retranslatorClient proto.RetranslatorControlClient // только для RETRANSLATOR
}

func (c *Client) GetInfo() (config.ServiceManager, error) {
	return c.info, nil
}
func (c *Client) ReceiverClient() proto.ReceiverControlClient {
	return c.receiverClient
}
func (c *Client) WriterClient() proto.WriteControlClient {
	return c.writerClient
}
func (c *Client) RetranslatorClient() proto.RetranslatorControlClient {
	return c.retranslatorClient
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
	case "WRITER":
		c.writerClient = proto.NewWriteControlClient(c.conn)
	case "RETRANSLATOR":
		c.retranslatorClient = proto.NewRetranslatorControlClient(c.conn)
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

func (c *Client) GetServiceManager(ctx context.Context) (*config.ServiceManager, error) {
	c.logger.Info("Calling GetServiceManager gRPC method")

	// Используем общий метод connect для проверки и восстановления соединения при необходимости
	if err := c.connect(); err != nil {
		c.logger.Errorf("Failed to ensure connection: %v", err)
		c.info.Active = false
		c.info.Status = "offline"
		c.info.ErrorMsg = err.Error()

		return &c.info, err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	manager, err := c.infoClient.GetServiceManager(ctx, &emptypb.Empty{})
	if err != nil {
		c.logger.Errorf("Failed to call GetServiceManager: %v", err)
		c.info.Active = false
		c.info.Status = "offline"
		c.info.ErrorMsg = err.Error()

		return &c.info, err
	}

	c.logger.Infof("Successfully received ServiceManager: %+v", manager)
	c.info.Active = true
	c.info.Status = "online"
	c.info.ErrorMsg = ""
	c.info.TypeSm = manager.TypeSm
	c.info.IpBroker = manager.IpBroker
	c.info.PortBroker = int(manager.PortBroker)

	return &c.info, nil
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

// GetLogLevel возвращает текущий уровень логирования сервиса
func (c *Client) GetLogLevel(ctx context.Context) (*proto.LogLevelResponse, error) {
	c.logger.Info("Calling GetLogLevel gRPC method")

	// Используем общий метод connect для проверки и восстановления соединения при необходимости
	if err := c.connect(); err != nil {
		c.logger.Errorf("Failed to ensure connection: %v", err)
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	response, err := c.infoClient.GetLogLevel(ctx, &emptypb.Empty{})
	if err != nil {
		c.logger.Errorf("Failed to call GetLogLevel: %v", err)
		return nil, err
	}

	c.logger.Info("Successfully received LogLevel response")
	return response, nil
}

// SetLogLevel устанавливает уровень логирования сервиса
func (c *Client) SetLogLevel(ctx context.Context, level string) (*proto.SetLogLevelResponse, error) {
	c.logger.Infof("Calling SetLogLevel gRPC method with level: %s", level)

	// Используем общий метод connect для проверки и восстановления соединения при необходимости
	if err := c.connect(); err != nil {
		c.logger.Errorf("Failed to ensure connection: %v", err)
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	request := &proto.SetLogLevelRequest{
		Level: level,
	}

	response, err := c.infoClient.SetLogLevel(ctx, request)
	if err != nil {
		c.logger.Errorf("Failed to call SetLogLevel: %v", err)
		return nil, err
	}

	c.logger.Info("Successfully set log level")
	return response, nil
}

// ReadLogs читает логи сервиса с применением фильтров
func (c *Client) ReadLogs(ctx context.Context, request *proto.ReadLogsRequest) (*proto.ReadLogsResponse, error) {
	c.logger.Info("Calling ReadLogs gRPC method")

	// Используем общий метод connect для проверки и восстановления соединения при необходимости
	if err := c.connect(); err != nil {
		c.logger.Errorf("Failed to ensure connection: %v", err)
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*10) // Увеличенный таймаут для чтения логов
	defer cancel()

	response, err := c.infoClient.ReadLogs(ctx, request)
	if err != nil {
		c.logger.Errorf("Failed to call ReadLogs: %v", err)
		return nil, err
	}

	c.logger.Info("Successfully received logs")
	return response, nil
}
