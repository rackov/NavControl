package main

import (
	"github.com/gin-gonic/gin"
	"github.com/rackov/NavControl/services/rest-api/internal/handlers"
)

func restMan(router *gin.Engine, h *handlers.Handler) {
	controller := router.Group("/controller")
	{
		controller.GET("/sm/level", h.GetLogLevel)
		controller.POST("/sm/level", h.SetLogLevel)
		controller.GET("/sm/log", h.ReadLogs)
		controller.GET("/sm", h.GetServiceModules)
		controller.POST("/sm", h.CreateServiceModule)
		controller.DELETE("/sm/:id_sm", h.DeleteServiceModule)
	}
}

/*


func (h *Handler) GetServiceModules(c *gin.Context) {
	serviceModules, err := h.service.GetServiceModules()
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to retrieve service modules"})
		return
	}

	c.JSON(200, serviceModules)
}
// CreateServiceModule создает новый сервис менеджер
func (h *Handler) CreateServiceModule(c *gin.Context) {
	var serviceModule ServiceModuleOld

	if err := c.ShouldBindJSON(&serviceModule); err != nil {
		c.JSON(400, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Проверка на существование сервиса с такими же параметрами
	exists, err := h.service.CheckServiceModuleExists(serviceModule)
	if err != nil {
		c.JSON(500, gin.H{
			"error": "Internal server error",
		})
		return
	}

	if exists {
		response := gin.H{
			"id_module":      0,
			"id_sm":          serviceModule.IDSm,
			"http_code":      409,
			"error_title":    "Conflict",
			"error_message":  "Service module already exists",
		}
		c.JSON(409, response)
		return
	}

	// Создание нового сервиса
	createdService, err := h.service.CreateServiceModule(serviceModule)
	if err != nil {
		c.JSON(500, gin.H{
			"error": "Failed to create service module",
		})
		return
	}

	c.JSON(201, createdService)
}


*/

/*

// UpdateReceiverStatus обновляет состояние сервиса приемника (запустить/остановить)
func (h *Handler) UpdateReceiverStatus(c *gin.Context) {
	// Получаем параметры из URL
	idSm := c.Param("id_sm")
	idRec := c.Param("id_rec")

	// Преобразуем ID в числа
	smId, err := strconv.Atoi(idSm)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid service manager ID"})
		return
	}

	recId, err := strconv.Atoi(idRec)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid receiver ID"})
		return
	}

	// Структура для тела запроса
	var req struct {
		IsActive bool `json:"is_active"`
	}

	// Парсим тело запроса
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Проверяем существование клиента
	h.mu.Lock()
	client, exists := h.services[smId]
	h.mu.Unlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service manager not found"})
		return
	}

	// Вызываем метод обновления статуса приемника через gRPC
	receiverInfo, err := client.UpdateReceiverStatus(c.Request.Context(), &proto.UpdateReceiverStatusRequest{
		IdReceiver: int32(recId),
		IdSm:       int32(smId),
		IsActive:   req.IsActive,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Возвращаем обновленную информацию о приемнике
	response := gin.H{
		"id_receiver":   receiverInfo.IdReceiver,
		"id_sm":         receiverInfo.IdSm,
		"ip_receiver":   receiverInfo.IpReceiver,
		"port_receiver": receiverInfo.PortReceiver,
		"name":          receiverInfo.Name,
		"protocol":      receiverInfo.Protocol,
		"ip_nats":       receiverInfo.IpNats,
		"port_nats":     receiverInfo.PortNats,
		"key_nats":      receiverInfo.KeyNats,
		"status":        receiverInfo.Status,
		"is_active":     receiverInfo.IsActive,
		"description":   receiverInfo.Description,
	}

	c.JSON(http.StatusOK, response)
}
*/
