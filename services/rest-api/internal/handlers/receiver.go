package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rackov/NavControl/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ListPorts получает список портов для указанного сервиса
func (h *Handler) ListPorts(c *gin.Context) {
	idStr := c.Param("id_sm")
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
	manager, err := client.GetServiceManager(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Проверяем, что сервис является RECEIVER
	if manager.TypeSm != "RECEIVER" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Service is not a RECEIVER"})
		return
	}

	// Вызываем gRPC метод для получения списка портов
	response, err := client.ReceiverClient().ListPorts(context.Background(), &emptypb.Empty{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetPortStatus получает статус порта
func (h *Handler) GetPortStatus(c *gin.Context) {
	// TODO: Реализовать при необходимости
}

// GetConnectedClients получает список подключенных клиентов
func (h *Handler) GetConnectedClients(c *gin.Context) {
	idStr := c.Param("id_sm")
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
	manager, err := client.GetServiceManager(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Проверяем, что сервис является RECEIVER
	if manager.TypeSm != "RECEIVER" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Service is not a RECEIVER"})
		return
	}

	// Получаем параметры запроса
	protocolName := c.Query("protocol_name")
	portReceiverStr := c.Query("port_receiver")

	var portReceiver int32
	if portReceiverStr != "" {
		port, err := strconv.ParseInt(portReceiverStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid port_receiver parameter"})
			return
		}
		portReceiver = int32(port)
	}

	// Создаем запрос
	request := &proto.GetClientsRequest{
		ProtocolName: protocolName,
		PortReceiver: portReceiver,
	}

	// Вызываем gRPC метод для получения списка клиентов
	response, err := client.ReceiverClient().GetConnectedClients(context.Background(), request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// DisconnectClient отключает клиента
func (h *Handler) DisconnectClient(c *gin.Context) {
	idStr := c.Param("id_sm")
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
	manager, err := client.GetServiceManager(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Проверяем, что сервис является RECEIVER
	if manager.TypeSm != "RECEIVER" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Service is not a RECEIVER"})
		return
	}

	// Парсим тело запроса
	var req proto.DisconnectClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Вызываем gRPC метод для отключения клиента
	response, err := client.ReceiverClient().DisconnectClient(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// AddPort добавляет новый порт в конфигурацию
func (h *Handler) AddPort(c *gin.Context) {
	// Парсим тело запроса
	var req proto.PortDefinition
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id := int(req.IdSm)
	client, exists := h.services[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	// Проверяем соединение и получаем клиент
	manager, err := client.GetServiceManager(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Проверяем, что сервис является RECEIVER
	if manager.TypeSm != "RECEIVER" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Service is not a RECEIVER"})
		return
	}

	// Вызываем gRPC метод для добавления порта
	response, err := client.ReceiverClient().AddPort(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// ChangeActive открывает или закрывает порт
func (h *Handler) ChangeActive(c *gin.Context) {
	idSmStr := c.Param("id_sm")
	idSm, err := strconv.Atoi(idSmStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid service ID"})
		return
	}

	idRecStr := c.Param("id_rec")
	idRec, err := strconv.Atoi(idRecStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid port ID"})
		return
	}

	client, exists := h.services[idSm]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	// Проверяем соединение и получаем клиент
	manager, err := client.GetServiceManager(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Проверяем, что сервис является RECEIVER
	if manager.TypeSm != "RECEIVER" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Service is not a RECEIVER"})
		return
	}

	// Получаем текущий статус порта для определения операции
	request := &proto.GetClientsRequest{
		PortReceiver: int32(idRec),
	}

	port, err := client.ReceiverClient().GetPortStatus(context.Background(), request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Создаем объект порта с измененным статусом
	portDefinition := &proto.PortDefinition{
		IdReceiver:   port.IdReceiver,
		IdSm:         port.IdSm,
		Name:         port.Name,
		PortReceiver: port.PortReceiver,
		Protocol:     port.Protocol,
		Description:  port.Description,
	}

	// Выполняем открытие или закрытие порта в зависимости от текущего статуса
	var response *proto.PortOperationResponse
	if port.Active {
		// Закрываем порт
		response, err = client.ReceiverClient().ClosePort(context.Background(), portDefinition)
	} else {
		// Открываем порт
		response, err = client.ReceiverClient().OpenPort(context.Background(), portDefinition)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// DeletePort удаляет порт из конфигурации
func (h *Handler) DeletePort(c *gin.Context) {
	idSmStr := c.Param("id_sm")
	idSm, err := strconv.Atoi(idSmStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid service ID"})
		return
	}

	idRecStr := c.Param("id_rec")
	idRec, err := strconv.Atoi(idRecStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid port ID"})
		return
	}

	client, exists := h.services[idSm]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	// Проверяем соединение и получаем клиент
	manager, err := client.GetServiceManager(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Проверяем, что сервис является RECEIVER
	if manager.TypeSm != "RECEIVER" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Service is not a RECEIVER"})
		return
	}

	// Получаем информацию о порте
	request := &proto.GetClientsRequest{
		PortReceiver: int32(idRec),
	}

	port, err := client.ReceiverClient().GetPortStatus(context.Background(), request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Вызываем gRPC метод для удаления порта
	response, err := client.ReceiverClient().DeletePort(context.Background(), port)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}
