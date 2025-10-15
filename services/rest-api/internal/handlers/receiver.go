package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rackov/NavControl/pkg/models"
	"github.com/rackov/NavControl/proto"
	"github.com/rackov/NavControl/services/rest-api/internal/restgrpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (h *Handler) ListAllReceiver(c *gin.Context) {
	responseNew := []models.ReceiverModule{}
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, client := range h.services {
		manager, err := client.GetServiceManager(c.Request.Context())
		if err != nil {
			continue
		}
		if manager.TypeSm != "RECEIVER" {
			continue
		}
		response, err := client.ReceiverClient().ListPorts(context.Background(), &emptypb.Empty{})
		if err != nil {
			continue

		}
		for _, r := range response.Ports {
			responseNew = append(responseNew,
				models.ReceiverModule{
					IDReceiver:       int(r.IdReceiver),
					IDSm:             manager.IdSm,
					Name:             r.Name,
					PortReceiver:     int(r.PortReceiver),
					Protocol:         r.Protocol,
					Active:           r.Active,
					Status:           r.Status,
					Description:      r.Description,
					ErrorMsg:         "",
					ConnectionsCount: int(r.ConnectionsCount),
				})
		}
	}
	c.JSON(http.StatusOK, responseNew)
}

// ListPorts получает список портов для указанного сервиса
func (h *Handler) ListPorts(c *gin.Context) {
	errUse := models.UsesMsgError{}

	idStr := c.Param("id_sm")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Неверный ID сервиса"
		errUse.HttpCode = http.StatusBadRequest
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	client, exists := h.services[id]
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

	// Проверяем, что сервис является RECEIVER
	if manager.TypeSm != "RECEIVER" {
		errUse.ErrorMsg = "Service is not a RECEIVER"
		errUse.ErrorTitle = "Ошибка типа сервиса"
		errUse.HttpCode = http.StatusBadRequest
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	// Вызываем gRPC метод для получения списка портов
	response, err := client.ReceiverClient().ListPorts(context.Background(), &emptypb.Empty{})
	if err != nil {
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Ошибка получения списка портов"
		errUse.HttpCode = http.StatusInternalServerError
		c.JSON(errUse.HttpCode, errUse)
		return
	}
	responseNew := []models.ReceiverModule{}
	for _, r := range response.Ports {
		responseNew = append(responseNew,
			models.ReceiverModule{
				IDReceiver:       int(r.IdReceiver),
				IDSm:             id,
				Name:             r.Name,
				PortReceiver:     int(r.PortReceiver),
				Protocol:         r.Protocol,
				Active:           r.Active,
				Status:           r.Status,
				Description:      r.Description,
				ErrorMsg:         "",
				ConnectionsCount: int(r.ConnectionsCount),
			})
	}
	c.JSON(http.StatusOK, responseNew)
}

// GetPortStatus получает статус порта
func (h *Handler) GetPortStatus(c *gin.Context) {
	// TODO: Реализовать при необходимости
}
func (h *Handler) ListAllClients(ctx context.Context) (conClients []models.Clients) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, client := range h.services {
		manager, err := client.GetServiceManager(ctx)
		if err != nil {
			continue
		}
		if manager.TypeSm != "RECEIVER" {
			continue
		}
		response, err := client.ReceiverClient().GetConnectedClients(context.Background(), &proto.GetClientsRequest{})
		if err != nil {
			continue

		}
		// Исправленный вариант
		for _, r := range response.Clients {
			clientInfo := models.Clients{
				IDReceiver:   int(r.IdReceiver),
				IDSm:         manager.IdSm,
				PortReceiver: int(r.PortReceiver),
				Address:      r.Address,
				ProtocolName: r.ProtocolName,
				ClientID:     r.Id,
				ConnectTime:  int(r.ConnectTime),
				LastTime:     int(r.LastTime),
				CountPackets: int(r.CountPackets),
				Multiple:     r.Multiple,
			}

			// Проверяем, что Device не nil перед обращением к его полям
			if r.Device != nil {
				clientInfo.IdInfo = models.IdInfo{
					Tid:  r.Device.Tid,
					Imei: r.Device.Imei,
				}
			}

			conClients = append(conClients, clientInfo)
		}
	}
	return
}

// GetConnectedClients получает список подключенных клиентов
func (h *Handler) GetConnectedClients(c *gin.Context) {
	errUse := models.UsesMsgError{}
	id_sm := c.Query("id_sm")
	id_rec := c.Query("id_rec")

	if id_sm == "" {
		conClients := h.ListAllClients(c.Request.Context())
		c.JSON(http.StatusOK, conClients)
		return

	}
	id, err := strconv.Atoi(id_sm)
	if err != nil {
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Неверный id_sm сервиса"
		errUse.HttpCode = http.StatusBadRequest
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	client, exists := h.services[id]
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

	// Проверяем, что сервис является RECEIVER
	if manager.TypeSm != "RECEIVER" {
		errUse.ErrorMsg = "Service is not a RECEIVER"
		errUse.ErrorTitle = "Ошибка типа сервиса"
		errUse.HttpCode = http.StatusBadRequest
		c.JSON(errUse.HttpCode, errUse)
		return
	}
	request := &proto.GetClientsRequest{}
	var idRec int
	if id_rec != "" {
		idRec, err = strconv.Atoi(id_rec)
		if err != nil {
			errUse.ErrorMsg = err.Error()
			errUse.ErrorTitle = "Неверный id_receiver сервиса"
			errUse.HttpCode = http.StatusBadRequest
			c.JSON(errUse.HttpCode, errUse)
			return
		}
		portDef, err := h.GetPortNumberByID(int32(idRec), client)
		if err != nil {
			errUse.ErrorMsg = err.Error()
			errUse.ErrorTitle = "Ошибка при получении номера порта"
			errUse.HttpCode = http.StatusBadRequest
			c.JSON(errUse.HttpCode, errUse)
			return
		}
		request.PortReceiver = portDef.PortReceiver
	}

	response, err := client.ReceiverClient().GetConnectedClients(context.Background(), request)
	if err != nil {
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Ошибка получения списка клиентов"
		errUse.HttpCode = http.StatusInternalServerError
		c.JSON(errUse.HttpCode, errUse)
		return
	}
	conClients := []models.Clients{}
	// Исправленный вариант в GetConnectedClients
	for _, r := range response.Clients {
		clientInfo := models.Clients{
			IDReceiver:   int(r.IdReceiver),
			IDSm:         manager.IdSm,
			PortReceiver: int(r.PortReceiver),
			Address:      r.Address,
			ProtocolName: r.ProtocolName,
			ClientID:     r.Id,
			ConnectTime:  int(r.ConnectTime),
			LastTime:     int(r.LastTime),
			CountPackets: int(r.CountPackets),
			Multiple:     r.Multiple,
		}

		// Проверяем, что Device не nil перед обращением к его полям
		if r.Device != nil {
			clientInfo.IdInfo = models.IdInfo{
				Tid:  r.Device.Tid,
				Imei: r.Device.Imei,
			}
		}

		conClients = append(conClients, clientInfo)
	}

	c.JSON(http.StatusOK, conClients)
}

// DisconnectClient отключает клиента
func (h *Handler) DisconnectClient(c *gin.Context) {
	errUse := models.UsesMsgError{}

	idStr := c.Param("id_sm")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Неверный ID сервиса"
		errUse.HttpCode = http.StatusBadRequest
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	client, exists := h.services[id]
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

	// Проверяем, что сервис является RECEIVER
	if manager.TypeSm != "RECEIVER" {
		errUse.ErrorMsg = "Service is not a RECEIVER"
		errUse.ErrorTitle = "Ошибка типа сервиса"
		errUse.HttpCode = http.StatusBadRequest
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	// Парсим тело запроса
	var req proto.DisconnectClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Не удалось распознать запрос JSON"
		errUse.HttpCode = http.StatusBadRequest
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	// Вызываем gRPC метод для отключения клиента
	response, err := client.ReceiverClient().DisconnectClient(context.Background(), &req)
	if err != nil {
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Ошибка отключения клиента"
		errUse.HttpCode = http.StatusInternalServerError
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	c.JSON(http.StatusOK, response)
}
func (h *Handler) ChangePortDescription(c *gin.Context) {
	var req proto.PortDefinition
	errUse := models.UsesMsgError{}

	if err := c.ShouldBindJSON(&req); err != nil {
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Не удалось распознать запрос JSON"
		errUse.HttpCode = http.StatusBadRequest
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	id := int(req.IdSm)
	client, exists := h.services[id]
	if !exists {
		errUse.ErrorMsg = "Service not found"
		errUse.ErrorTitle = "Не удалось распознать запрос JSON"
		errUse.HttpCode = http.StatusNotFound
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	// Проверяем соединение и получаем клиент
	manager, err := client.GetServiceManager(c.Request.Context())
	if err != nil {
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Ошибка доступа к методу  сервиса"
		errUse.HttpCode = http.StatusInternalServerError
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	// Проверяем, что сервис является RECEIVER
	if manager.TypeSm != "RECEIVER" {
		errUse.ErrorMsg = "Service is not a RECEIVER"
		errUse.ErrorTitle = "Ошибка типа  сервиса"
		errUse.HttpCode = http.StatusBadRequest
		c.JSON(errUse.HttpCode, errUse)

		return
	}

	response, err := client.ReceiverClient().ChangePortDescription(context.Background(), &req)

	if err != nil {
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Ошибка изменения порта"
		errUse.HttpCode = http.StatusInternalServerError
		c.JSON(errUse.HttpCode, errUse)
		return
	}
	c.JSON(http.StatusOK, response.PortDetails)

}

// AddPort добавляет новый порт в конфигурацию
func (h *Handler) AddPort(c *gin.Context) {
	// Парсим тело запроса
	var req proto.PortDefinition
	errUse := models.UsesMsgError{}

	if err := c.ShouldBindJSON(&req); err != nil {
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Не удалось распознать запрос JSON"
		errUse.HttpCode = http.StatusBadRequest
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	id := int(req.IdSm)
	client, exists := h.services[id]
	if !exists {
		errUse.ErrorMsg = "Service not found"
		errUse.ErrorTitle = "Не удалось распознать запрос JSON"
		errUse.HttpCode = http.StatusNotFound
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	// Проверяем соединение и получаем клиент
	manager, err := client.GetServiceManager(c.Request.Context())
	if err != nil {
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Ошибка доступа к методу  сервиса"
		errUse.HttpCode = http.StatusInternalServerError
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	// Проверяем, что сервис является RECEIVER
	if manager.TypeSm != "RECEIVER" {
		errUse.ErrorMsg = "Service is not a RECEIVER"
		errUse.ErrorTitle = "Ошибка типа  сервиса"
		errUse.HttpCode = http.StatusBadRequest
		c.JSON(errUse.HttpCode, errUse)

		return
	}

	// Вызываем gRPC метод для добавления порта
	response, err := client.ReceiverClient().AddPort(context.Background(), &req)
	if err != nil {
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Ошибка добавления порта"
		errUse.HttpCode = http.StatusInternalServerError
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	c.JSON(http.StatusOK, response.PortDetails)
}

// ChangeActive открывает или закрывает порт
func (h *Handler) ChangeActive(c *gin.Context) {
	errUse := models.UsesMsgError{}

	idSmStr := c.Param("id_sm")
	idSm, err := strconv.Atoi(idSmStr)
	if err != nil {
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Ошибка id сервиса"
		errUse.HttpCode = http.StatusBadRequest
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	idRecStr := c.Param("id_rec")
	idRec, err := strconv.Atoi(idRecStr)
	if err != nil {
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Ошибка id порта"
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
		errUse.ErrorTitle = "Метод сервиса не доступен"
		errUse.HttpCode = http.StatusInternalServerError
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	// Проверяем, что сервис является RECEIVER
	if manager.TypeSm != "RECEIVER" {
		errUse.ErrorMsg = "Service is not a RECEIVER"
		errUse.ErrorTitle = "Не является получателем"
		errUse.HttpCode = http.StatusBadRequest
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	port, err := h.GetPortNumberByID(int32(idRec), client)
	if err != nil {
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Ошибка при определении номера порта"
		errUse.HttpCode = http.StatusInternalServerError
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	// Создаем объект порта с измененным статусом
	portDefinition := &proto.PortDefinition{
		IdReceiver:       port.IdReceiver,
		IdSm:             port.IdSm,
		Name:             port.Name,
		PortReceiver:     port.PortReceiver,
		Protocol:         port.Protocol,
		Description:      port.Description,
		Active:           !port.Active,
		Status:           port.Status,
		ConnectionsCount: int32(port.ConnectionsCount),
	}

	// Выполняем открытие или закрытие порта в зависимости от текущего статуса
	// var response *proto.PortOperationResponse
	if port.Active {
		// Закрываем порт
		// response
		_, err = client.ReceiverClient().ClosePort(context.Background(), portDefinition)
	} else {
		// Открываем порт
		// response
		_, err = client.ReceiverClient().OpenPort(context.Background(), portDefinition)
	}

	if err != nil {
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Ошибка изменения состояния порта"
		errUse.HttpCode = http.StatusInternalServerError
		c.JSON(errUse.HttpCode, errUse)
		return
	}
	responseNew := models.ReceiverModule{
		IDReceiver:       int(portDefinition.IdReceiver),
		IDSm:             manager.IdSm,
		Name:             portDefinition.Name,
		PortReceiver:     int(portDefinition.PortReceiver),
		Protocol:         portDefinition.Protocol,
		Active:           portDefinition.Active,
		Status:           portDefinition.Status,
		Description:      portDefinition.Description,
		ErrorMsg:         "",
		ConnectionsCount: int(portDefinition.ConnectionsCount),
	}

	c.JSON(http.StatusOK, responseNew)
}

// получение номера порта по ID
func (h *Handler) GetPortNumberByID(id int32, client *restgrpc.Client) (*proto.PortDefinition, error) {
	var portDefinition *proto.PortDefinition

	// Получаем список портов
	listResponse, err := client.ReceiverClient().ListPorts(context.Background(), &emptypb.Empty{})
	if err != nil {
		return nil, err
	}

	for _, r := range listResponse.Ports {
		if r.IdReceiver == id {
			portDefinition = r
			break

		}
	}

	if portDefinition == nil {
		return nil, errors.New("port not found")
	}
	return portDefinition, nil
}

// DeletePort удаляет порт из конфигурации
func (h *Handler) DeletePort(c *gin.Context) {
	errUse := models.UsesMsgError{}

	idSmStr := c.Param("id_sm")
	idSm, err := strconv.Atoi(idSmStr)
	if err != nil {
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Ошибка в параметре id_sm"
		errUse.HttpCode = http.StatusBadRequest
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	idRecStr := c.Param("id_rec")
	idRec, err := strconv.Atoi(idRecStr)
	if err != nil {
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Ошибка в параметре id_rec"
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
		errUse.ErrorTitle = "Метод сервиса не доступен"
		errUse.HttpCode = http.StatusInternalServerError
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	// Проверяем, что сервис является RECEIVER
	if manager.TypeSm != "RECEIVER" {
		errUse.ErrorMsg = "Service is not a RECEIVER"
		errUse.ErrorTitle = "Сервис не является получателем данных"
		errUse.HttpCode = http.StatusBadRequest
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	portToDelete, err := h.GetPortNumberByID(int32(idRec), client)
	if err != nil {
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Ошибка при получении номера порта"
		errUse.HttpCode = http.StatusBadRequest
		c.JSON(errUse.HttpCode, errUse)
		return
	}
	// Вызываем gRPC метод для удаления порта
	_, err = client.ReceiverClient().DeletePort(context.Background(), portToDelete)
	if err != nil {
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Ошибка при удалении  порта"
		errUse.HttpCode = http.StatusInternalServerError
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	c.JSON(http.StatusOK, portToDelete)
}
