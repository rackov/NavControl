package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rackov/NavControl/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ListClient получает список клиентов ретранслятора
func (h *Handler) ListClient(c *gin.Context) {
	idSm, err := strconv.Atoi(c.Param("id_sm"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id_sm parameter"})
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

	// Проверяем, что сервис является RETRANSLATOR
	if manager.TypeSm != "RETRANSLATOR" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Service is not a RETRANSLATOR"})
		return
	}

	response, err := client.RetranslatorClient().ListClient(context.Background(), &emptypb.Empty{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// AddClient добавляет нового клиента ретранслятора
func (h *Handler) AddClient(c *gin.Context) {
	var client proto.Client
	if err := c.ShouldBindJSON(&client); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	idSm, err := strconv.Atoi(c.Param("id_sm"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id_sm parameter"})
		return
	}

	serviceClient, exists := h.services[idSm]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	manager, err := serviceClient.GetServiceManager(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Проверяем, что сервис является RETRANSLATOR
	if manager.TypeSm != "RETRANSLATOR" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Service is not a RETRANSLATOR"})
		return
	}

	newClient, err := serviceClient.RetranslatorClient().AddClient(context.Background(), &client)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, newClient)
}

// UpdateClient обновляет информацию о клиенте ретранслятора
func (h *Handler) UpdateClient(c *gin.Context) {
	var client proto.Client
	if err := c.ShouldBindJSON(&client); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	serviceClient, exists := h.services[int(client.IdSm)]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	manager, err := serviceClient.GetServiceManager(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Проверяем, что сервис является RETRANSLATOR
	if manager.TypeSm != "RETRANSLATOR" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Service is not a RETRANSLATOR"})
		return
	}

	updatedClient, err := serviceClient.RetranslatorClient().UpdateClient(context.Background(), &client)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedClient)
}

// ChangeActiveClient изменяет статус активности клиента (включает/выключает)
func (h *Handler) ChangeActiveClient(c *gin.Context) {
	idSm, err := strconv.Atoi(c.Param("id_sm"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id_sm parameter"})
		return
	}

	idRet, err := strconv.Atoi(c.Param("id_ret"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id_ret parameter"})
		return
	}

	setClient := &proto.SetClient{
		IdRetranslator: int32(idRet),
		IdSm:           int32(idSm),
	}

	serviceClient, exists := h.services[idSm]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	manager, err := serviceClient.GetServiceManager(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Проверяем, что сервис является RETRANSLATOR
	if manager.TypeSm != "RETRANSLATOR" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Service is not a RETRANSLATOR"})
		return
	}

	var updatedClient *proto.Client
	// Определяем, какой метод вызывать, в зависимости от HTTP метода
	if c.Request.Method == "PATCH" {
		// Для PATCH мы можем использовать UpClient или DownClient в зависимости от текущего состояния
		// В данном случае просто вызываем UpClient как активацию
		updatedClient, err = serviceClient.RetranslatorClient().UpClient(context.Background(), setClient)
	} else {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "method not allowed"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedClient)
}

// DeleteClient удаляет клиента ретранслятора
func (h *Handler) DeleteClient(c *gin.Context) {
	idSm, err := strconv.Atoi(c.Param("id_sm"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id_sm parameter"})
		return
	}

	idRet, err := strconv.Atoi(c.Param("id_ret"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id_ret parameter"})
		return
	}

	setClient := &proto.SetClient{
		IdRetranslator: int32(idRet),
		IdSm:           int32(idSm),
	}

	serviceClient, exists := h.services[idSm]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	manager, err := serviceClient.GetServiceManager(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Проверяем, что сервис является RETRANSLATOR
	if manager.TypeSm != "RETRANSLATOR" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Service is not a RETRANSLATOR"})
		return
	}

	_, err = serviceClient.RetranslatorClient().DeleteClient(context.Background(), setClient)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "client deleted successfully"})
}

// ListDevices получает список устройств ретранслятора
func (h *Handler) ListDevices(c *gin.Context) {
	idSm, err := strconv.Atoi(c.Param("id_sm"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id_sm parameter"})
		return
	}

	setClient := &proto.SetClient{
		IdSm: int32(idSm),
	}

	serviceClient, exists := h.services[idSm]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	manager, err := serviceClient.GetServiceManager(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Проверяем, что сервис является RETRANSLATOR
	if manager.TypeSm != "RETRANSLATOR" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Service is not a RETRANSLATOR"})
		return
	}

	devices, err := serviceClient.RetranslatorClient().ListDevices(context.Background(), setClient)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, devices)
}
