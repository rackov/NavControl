package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rackov/NavControl/pkg/config"
	"github.com/rackov/NavControl/proto"
	"github.com/rackov/NavControl/services/rest-api/internal/restgrpc"
)

func (h *Handler) GetServiceManager(c *gin.Context) {
	serviceType := c.Param("id")

	id, err := strconv.Atoi(serviceType)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid service ID"})
		return
	}

	client, exists := h.services[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	manager, err := client.GetServiceManager(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, manager)
}

func (h *Handler) GetServiceModules(c *gin.Context) {
	result := []*proto.ServiceManager{}
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, client := range h.services {
		manager, err := client.GetServiceManager(c.Request.Context())
		if err != nil {
			// c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get %d manager: %v", manager.IdSm, err)})

			manager.Status = err.Error()
		}
		h.logger.Infof("Received service manager: %v", manager)
		// manager.IdSm = int32(id)
		result = append(result, manager)
	}

	c.JSON(http.StatusOK, result)
}

// Добавляем новый метод для создания сервиса через POST
func (h *Handler) CreateServiceModule(c *gin.Context) {
	h.logger.Info("Received request to create a new service")

	// Определяем структуру для входных данных
	var req config.ServiceManager
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorf("Failed to bind request JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Проверяем, что тип сервиса корректен
	if req.TypeSm != "RECEIVER" && req.TypeSm != "WRITER" && req.TypeSm != "RETRANSLATOR" {
		h.logger.Errorf("Invalid service type: %s", req.TypeSm)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid service type"})
		return
	}

	// Создаем новый gRPC клиент
	client, err := restgrpc.NewClient(req, h.logger)
	if err != nil {
		h.logger.Errorf("Failed to create gRPC client: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Добавляем клиент в карту сервисов
	h.services[req.IdSm] = client
	h.logger.Infof("Successfully created new service of type %s at %s:%d", req.TypeSm, req.IpSm, req.PortSm)

	// Возвращаем успешный ответ
	c.JSON(http.StatusCreated, gin.H{
		"type":   req.TypeSm,
		"host":   req.IpSm,
		"port":   req.PortSm,
		"active": req.Active,
	})
}
