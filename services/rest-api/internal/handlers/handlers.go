package handlers

import (
	"sync"

	"github.com/rackov/NavControl/pkg/config"
	"github.com/rackov/NavControl/pkg/logger"
	"github.com/rackov/NavControl/services/rest-api/internal/restgrpc"
)

type Handler struct {
	services map[int]*restgrpc.Client
	logger   *logger.Logger // Добавляем логгер в структуру Handler
	cfg      *config.CfgRestApi
	mu       sync.RWMutex
}

// Обновляем конструктор для принятия логгера
func NewHandler(cfg *config.CfgRestApi, logger *logger.Logger) (*Handler, error) {
	h := &Handler{
		services: make(map[int]*restgrpc.Client),
		logger:   logger, // Сохраняем логгер
		cfg:      cfg,
	}

	// Инициализация gRPC клиентов для каждого сервиса
	for _, service := range cfg.ServiceList {
		client, err := restgrpc.NewClient(service, logger) // Передаем логгер в gRPC клиент
		if err != nil {
			logger.Errorf("Failed to create gRPC client for %d: %v", service.IdSm, err)
			return nil, err
		}
		h.services[service.IdSm] = client
		logger.Infof("Successfully connected to %d service at %s:%d", service.IdSm, service.IpSm, service.PortSm)
	}

	return h, nil
}

// Обновляем метод Close для корректного закрытия всех соединений
func (h *Handler) Close() {
	h.logger.Info("Closing all gRPC connections")
	for name, client := range h.services {
		if err := client.Close(); err != nil {
			h.logger.Errorf("Failed to close connection for %d: %v", name, err)
		} else {
			h.logger.Infof("Successfully closed connection for %d", name)
		}
	}
}
