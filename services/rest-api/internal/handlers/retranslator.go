package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rackov/NavControl/pkg/models"
	"github.com/rackov/NavControl/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (h *Handler) ListAllRetranslator(c *gin.Context) {
	responseNew := []*proto.Client{}
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, client := range h.services {
		manager, err := client.GetServiceManager(c.Request.Context())
		if err != nil {
			continue
		}
		if manager.TypeSm != "RETRANSLATOR" {
			continue
		}
		response, err := client.RetranslatorClient().ListClient(context.Background(), &emptypb.Empty{})
		if err != nil {
			continue

		}
		for _, r := range response.Clients {
			r.IdSm = int32(manager.IdSm)
			responseNew = append(responseNew, r)
		}
	}
	c.JSON(http.StatusOK, responseNew)
}

// ListClient получает список клиентов ретранслятора
func (h *Handler) ListClient(c *gin.Context) {
	errUse := models.UsesMsgError{}

	idSm, err := strconv.Atoi(c.Param("id_sm"))
	if err != nil {
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Неверный ID сервиса"
		errUse.HttpCode = http.StatusBadRequest
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	client, exists := h.services[idSm]
	if !exists {
		errUse.ErrorMsg = "Service not found"
		errUse.ErrorTitle = "Сервис не найден"
		errUse.HttpCode = http.StatusNotFound
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	// Проверяем соединение и получаем клиент
	manager, err := client.GetServiceManager(c.Request.Context())
	if err != nil {
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Ошибка доступа к методу сервиса"
		errUse.HttpCode = http.StatusInternalServerError
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	// Проверяем, что сервис является RETRANSLATOR
	if manager.TypeSm != "RETRANSLATOR" {
		errUse.ErrorMsg = "Service is not a RETRANSLATOR"
		errUse.ErrorTitle = "Ошибка типа сервиса"
		errUse.HttpCode = http.StatusBadRequest
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	response, err := client.RetranslatorClient().ListClient(context.Background(), &emptypb.Empty{})
	if err != nil {
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Ошибка получения списка клиентов"
		errUse.HttpCode = http.StatusInternalServerError
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	c.JSON(http.StatusOK, response.Clients)
}

// AddClient добавляет нового клиента ретранслятора
func (h *Handler) AddClient(c *gin.Context) {
	errUse := models.UsesMsgError{}

	var client proto.Client
	if err := c.ShouldBindJSON(&client); err != nil {
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Не удалось распознать запрос JSON"
		errUse.HttpCode = http.StatusBadRequest
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	idSm, err := strconv.Atoi(c.Param("id_sm"))
	if err != nil {
		errUse.ErrorMsg = "invalid id_sm parameter"
		errUse.ErrorTitle = "Неверный параметр id_sm"
		errUse.HttpCode = http.StatusBadRequest
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	serviceClient, exists := h.services[idSm]
	if !exists {
		errUse.ErrorMsg = "Service not found"
		errUse.ErrorTitle = "Сервис не найден"
		errUse.HttpCode = http.StatusNotFound
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	manager, err := serviceClient.GetServiceManager(c.Request.Context())
	if err != nil {
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Ошибка доступа к методу сервиса"
		errUse.HttpCode = http.StatusInternalServerError
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	// Проверяем, что сервис является RETRANSLATOR
	if manager.TypeSm != "RETRANSLATOR" {
		errUse.ErrorMsg = "Service is not a RETRANSLATOR"
		errUse.ErrorTitle = "Ошибка типа сервиса"
		errUse.HttpCode = http.StatusBadRequest
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	newClient, err := serviceClient.RetranslatorClient().AddClient(context.Background(), &client)
	if err != nil {
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Ошибка добавления клиента"
		errUse.HttpCode = http.StatusInternalServerError
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	c.JSON(http.StatusOK, newClient)
}

// UpdateClient обновляет информацию о клиенте ретранслятора
func (h *Handler) UpdateClient(c *gin.Context) {
	errUse := models.UsesMsgError{}

	var client proto.Client
	if err := c.ShouldBindJSON(&client); err != nil {
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Не удалось распознать запрос JSON"
		errUse.HttpCode = http.StatusBadRequest
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	serviceClient, exists := h.services[int(client.IdSm)]
	if !exists {
		errUse.ErrorMsg = "Service not found"
		errUse.ErrorTitle = "Сервис не найден"
		errUse.HttpCode = http.StatusNotFound
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	manager, err := serviceClient.GetServiceManager(c.Request.Context())
	if err != nil {
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Ошибка доступа к методу сервиса"
		errUse.HttpCode = http.StatusInternalServerError
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	// Проверяем, что сервис является RETRANSLATOR
	if manager.TypeSm != "RETRANSLATOR" {
		errUse.ErrorMsg = "Service is not a RETRANSLATOR"
		errUse.ErrorTitle = "Ошибка типа сервиса"
		errUse.HttpCode = http.StatusBadRequest
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	updatedClient, err := serviceClient.RetranslatorClient().UpdateClient(context.Background(), &client)
	if err != nil {
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Ошибка обновления клиента"
		errUse.HttpCode = http.StatusInternalServerError
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	c.JSON(http.StatusOK, updatedClient)
}

// ChangeActiveClient изменяет статус активности клиента (включает/выключает)
func (h *Handler) ChangeActiveClient(c *gin.Context) {
	errUse := models.UsesMsgError{}

	idSm, err := strconv.Atoi(c.Param("id_sm"))
	if err != nil {
		errUse.ErrorMsg = "invalid id_sm parameter"
		errUse.ErrorTitle = "Неверный параметр id_sm"
		errUse.HttpCode = http.StatusBadRequest
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	idRet, err := strconv.Atoi(c.Param("id_ret"))
	if err != nil {
		errUse.ErrorMsg = "invalid id_ret parameter"
		errUse.ErrorTitle = "Неверный параметр id_ret"
		errUse.HttpCode = http.StatusBadRequest
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	setClient := &proto.SetClient{
		IdRetranslator: int32(idRet),
		IdSm:           int32(idSm),
	}

	serviceClient, exists := h.services[idSm]
	if !exists {
		errUse.ErrorMsg = "Service not found"
		errUse.ErrorTitle = "Сервис не найден"
		errUse.HttpCode = http.StatusNotFound
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	manager, err := serviceClient.GetServiceManager(c.Request.Context())
	if err != nil {
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Ошибка доступа к методу сервиса"
		errUse.HttpCode = http.StatusInternalServerError
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	// Проверяем, что сервис является RETRANSLATOR
	if manager.TypeSm != "RETRANSLATOR" {
		errUse.ErrorMsg = "Service is not a RETRANSLATOR"
		errUse.ErrorTitle = "Ошибка типа сервиса"
		errUse.HttpCode = http.StatusBadRequest
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	var updatedClient *proto.Client
	// Определяем, какой метод вызывать, в зависимости от HTTP метода
	clientinfo, err := serviceClient.RetranslatorClient().GetInfoClient(context.Background(), setClient)

	if !clientinfo.IsActive {
		updatedClient, err = serviceClient.RetranslatorClient().UpClient(context.Background(), setClient)
	} else {
		updatedClient, err = serviceClient.RetranslatorClient().DownClient(context.Background(), setClient)
	}

	if err != nil {
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Ошибка изменения статуса клиента"
		errUse.HttpCode = http.StatusInternalServerError
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	c.JSON(http.StatusOK, updatedClient)
}

// DeleteClient удаляет клиента ретранслятора
func (h *Handler) DeleteClient(c *gin.Context) {
	errUse := models.UsesMsgError{}

	idSm, err := strconv.Atoi(c.Param("id_sm"))
	if err != nil {
		errUse.ErrorMsg = "invalid id_sm parameter"
		errUse.ErrorTitle = "Неверный параметр id_sm"
		errUse.HttpCode = http.StatusBadRequest
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	idRet, err := strconv.Atoi(c.Param("id_ret"))
	if err != nil {
		errUse.ErrorMsg = "invalid id_ret parameter"
		errUse.ErrorTitle = "Неверный параметр id_ret"
		errUse.HttpCode = http.StatusBadRequest
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	setClient := &proto.SetClient{
		IdRetranslator: int32(idRet),
		IdSm:           int32(idSm),
	}

	serviceClient, exists := h.services[idSm]
	if !exists {
		errUse.ErrorMsg = "Service not found"
		errUse.ErrorTitle = "Сервис не найден"
		errUse.HttpCode = http.StatusNotFound
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	manager, err := serviceClient.GetServiceManager(c.Request.Context())
	if err != nil {
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Ошибка доступа к методу сервиса"
		errUse.HttpCode = http.StatusInternalServerError
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	// Проверяем, что сервис является RETRANSLATOR
	if manager.TypeSm != "RETRANSLATOR" {
		errUse.ErrorMsg = "Service is not a RETRANSLATOR"
		errUse.ErrorTitle = "Ошибка типа сервиса"
		errUse.HttpCode = http.StatusBadRequest
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	_, err = serviceClient.RetranslatorClient().DeleteClient(context.Background(), setClient)
	if err != nil {
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Ошибка удаления клиента"
		errUse.HttpCode = http.StatusInternalServerError
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "client deleted successfully"})
}

// ListDevices получает список устройств ретранслятора
func (h *Handler) ListDevices(c *gin.Context) {
	errUse := models.UsesMsgError{}

	idSm, err := strconv.Atoi(c.Param("id_sm"))
	if err != nil {
		errUse.ErrorMsg = "invalid id_sm parameter"
		errUse.ErrorTitle = "Неверный параметр id_sm"
		errUse.HttpCode = http.StatusBadRequest
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	setClient := &proto.SetClient{
		IdSm: int32(idSm),
	}

	serviceClient, exists := h.services[idSm]
	if !exists {
		errUse.ErrorMsg = "Service not found"
		errUse.ErrorTitle = "Сервис не найден"
		errUse.HttpCode = http.StatusNotFound
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	manager, err := serviceClient.GetServiceManager(c.Request.Context())
	if err != nil {
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Ошибка доступа к методу сервиса"
		errUse.HttpCode = http.StatusInternalServerError
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	// Проверяем, что сервис является RETRANSLATOR
	if manager.TypeSm != "RETRANSLATOR" {
		errUse.ErrorMsg = "Service is not a RETRANSLATOR"
		errUse.ErrorTitle = "Ошибка типа сервиса"
		errUse.HttpCode = http.StatusBadRequest
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	devices, err := serviceClient.RetranslatorClient().ListDevices(context.Background(), setClient)
	if err != nil {
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Ошибка получения списка устройств"
		errUse.HttpCode = http.StatusInternalServerError
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	c.JSON(http.StatusOK, devices)
}
