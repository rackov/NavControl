package handlers

import (
	"net/http"
	"strconv"
	"strings"

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

// GetLogLevel возвращает текущий уровень логирования сервиса
func (h *Handler) GetLogLevel(c *gin.Context) {
	idStr := c.Query("id_sm")
	if idStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id_sm parameter is required"})
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid service ID"})
		return
	}

	client, exists := h.services[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	// Проверяем соединение и получаем клиент
	_, err = client.GetServiceManager(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Вызываем gRPC метод для получения уровня логирования
	logLevel, err := client.GetLogLevel(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, logLevel)
}

// SetLogLevel устанавливает уровень логирования сервиса
func (h *Handler) SetLogLevel(c *gin.Context) {
	var req struct {
		IDSm  int    `json:"id_sm"`
		Level string `json:"level"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client, exists := h.services[req.IDSm]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	// Проверяем соединение и получаем клиент
	_, err := client.GetServiceManager(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	req.Level = strings.ToLower(req.Level)
	// Вызываем gRPC метод для установки уровня логирования
	response, err := client.SetLogLevel(c.Request.Context(), req.Level)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// response. = strings.ToUpper(response.Level)

	c.JSON(http.StatusOK, response)
}

// ReadLogs читает логи сервиса с применением фильтров
func (h *Handler) ReadLogs(c *gin.Context) {
	idStr := c.Query("id_sm")
	if idStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id_sm parameter is required"})
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid service ID"})
		return
	}

	client, exists := h.services[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	// Проверяем соединение и получаем клиент
	_, err = client.GetServiceManager(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Создаем запрос на чтение логов
	readLogsReq := &proto.ReadLogsRequest{}

	// Парсим параметры запроса
	if level := c.Query("level"); level != "" {
		readLogsReq.Level = level
	}

	if startDateStr := c.Query("start_date"); startDateStr != "" {
		startDate, err := strconv.ParseInt(startDateStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_date parameter"})
			return
		}
		readLogsReq.StartDate = startDate
	}

	if endDateStr := c.Query("end_date"); endDateStr != "" {
		endDate, err := strconv.ParseInt(endDateStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_date parameter"})
			return
		}
		readLogsReq.EndDate = endDate
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		limit, err := strconv.ParseInt(limitStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit parameter"})
			return
		}
		readLogsReq.Limit = int32(limit)
	}

	// Вызываем gRPC метод для чтения логов
	response, err := client.ReadLogs(c.Request.Context(), readLogsReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetServiceModules возвращает список всех сервисов
func (h *Handler) GetServiceModules(c *gin.Context) {
	result := []*config.ServiceManager{}
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, client := range h.services {
		manager, err := client.GetServiceManager(c.Request.Context())
		if err != nil {
			// c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get %d manager: %v", manager.IdSm, err)})
			manager.Status = "ofline"
			manager.ErrorMsg = err.Error()
		}
		h.logger.Infof("Received service manager: %v", manager)
		// manager.IdSm = int32(id)
		manager.LogLevel = strings.ToUpper(manager.LogLevel)
		result = append(result, manager)
	}

	c.JSON(http.StatusOK, result)
}

// Находим максимальный ID
func (h *Handler) maxId() int {
	h.mu.Lock()
	defer h.mu.Unlock()

	Id := 0
	for _, ser := range h.cfg.ServiceList {
		if ser.IdSm > Id {
			Id = ser.IdSm
		}
	}
	return Id
}
func (h *Handler) findId(idSm int) int {
	h.mu.Lock()
	defer h.mu.Unlock()

	for i, ser := range h.cfg.ServiceList {
		if ser.IdSm == idSm {
			return i
		}
	}
	return -1
}

// создание сервиса через POST
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
	req.IdSm = h.maxId() + 1
	req.Active = true
	req.Status = "online"
	req.ErrorMsg = ""

	// Создаем новый gRPC клиент
	client, err := restgrpc.NewClient(req, h.logger)
	if err != nil {
		h.logger.Errorf("Failed to create gRPC client: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	manager, err := client.GetServiceManager(c.Request.Context())
	if err != nil {
		h.logger.Errorf("Не возможно подключится к сурвису: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if manager.TypeSm != req.TypeSm {
		h.logger.Errorf("Не верный тип сервиса %s", manager.TypeSm)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не верный тип сервиса"})
		return

	}
	// Добавляем клиент в карту сервисов
	h.services[req.IdSm] = client
	h.cfg.ServiceList = append(h.cfg.ServiceList, req)

	h.logger.Infof("Successfully created new service of type %s at %s:%d", req.TypeSm, req.IpSm, req.PortSm)
	// if err:=h.cfg.SaveCfg(), err!=nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось сохранить конфигурацию"})
	// 	return
	// }
	err = h.cfg.SaveCfg()
	if err != nil {
		h.logger.Errorf("Невозможно сохранить конфигурацию: %v", err)
		return
	}

	// Возвращаем успешный ответ
	c.JSON(http.StatusCreated, req)
}

// Удаляем сервис
func (h *Handler) DeleteServiceModule(c *gin.Context) {

	id := c.Param("id_sm")
	idInt, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	client, exists := h.services[idInt]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	h.logger.Infof("Received request to delete service with ID %d", idInt)
	conf, err := client.GetInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.services[idInt].Close()
	delete(h.services, idInt)
	idind := h.findId(idInt)
	if idind != -1 {
		h.cfg.ServiceList = append(h.cfg.ServiceList[:idind], h.cfg.ServiceList[idind+1:]...)
		err = h.cfg.SaveCfg()
		if err != nil {
			h.logger.Errorf("Невозможно сохранить конфигурацию: %v", err)
			return
		}
	}

	c.JSON(http.StatusOK, conf)
}
