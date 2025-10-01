package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rackov/NavControl/pkg/models"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/rackov/NavControl/proto"
)

func (h *Handler) ListWriters(c *gin.Context) {
	errUse := models.UsesMsgError{}

	idSm, err := strconv.Atoi(c.Param("id_sm"))
	if err != nil {
		errUse.ErrorMsg = "invalid id_sm parameter"
		errUse.ErrorTitle = "Неверный параметр id_sm"
		errUse.HttpCode = http.StatusBadRequest
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	client, exists := h.services[idSm]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	if !exists {
		errUse.ErrorMsg = "Service not found"
		errUse.ErrorTitle = "Не удалось распознать запрос JSON"
		errUse.HttpCode = http.StatusNotFound
		c.JSON(errUse.HttpCode, errUse)
		return
	}
	manager, err := client.GetServiceManager(c.Request.Context())
	if err != nil {
		h.logger.Errorf("Не возможно подключится к сервису: %v", err)
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Не возможно подключится к сервису"
		errUse.HttpCode = http.StatusInternalServerError
		c.JSON(errUse.HttpCode, errUse)
		return
	}
	if manager.TypeSm != "WRITER" {
		h.logger.Errorf("Не верный тип сервиса %s", manager.TypeSm)
		errUse.ErrorMsg = fmt.Sprintf("Invalid service type WRITER: %s", manager.TypeSm)
		errUse.ErrorTitle = "Не верный тип сервиса на выбранном порту"
		errUse.HttpCode = http.StatusInternalServerError
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	list, err := client.WriterClient().ListWrites(context.Background(), &emptypb.Empty{})
	if err != nil {
		h.logger.Errorf("Не возможно подключится к сервису: %v", err)
		errUse.ErrorMsg = fmt.Sprintf("Не возможно подключится к сервису: %v", err)
		errUse.ErrorTitle = "Не возможно подключится к сервису"
		errUse.HttpCode = http.StatusInternalServerError
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	c.JSON(http.StatusOK, list)
}

func (h *Handler) AddWriter(c *gin.Context) {
	errUse := models.UsesMsgError{}

	var writeService pb.WriteService
	if err := c.ShouldBindJSON(&writeService); err != nil {
		errUse.ErrorMsg = "Invalid JSON format"
		errUse.ErrorTitle = "Не удалось распознать запрос JSON"
		errUse.HttpCode = http.StatusBadRequest
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	client, exists := h.services[int(writeService.IdSm)]
	if !exists {
		errUse.ErrorMsg = "Service not found"
		errUse.ErrorTitle = "Сервис не найден"
		errUse.HttpCode = http.StatusNotFound
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	manager, err := client.GetServiceManager(c.Request.Context())
	if err != nil {
		h.logger.Errorf("Не возможно подключится к сервису: %v", err)
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Не возможно подключится к сервису"
		errUse.HttpCode = http.StatusInternalServerError
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	if manager.TypeSm != "WRITER" {
		h.logger.Errorf("Не верный тип сервиса %s", manager.TypeSm)
		errUse.ErrorMsg = fmt.Sprintf("Invalid service type WRITER: %s", manager.TypeSm)
		errUse.ErrorTitle = "Не верный тип сервиса на выбранном порту"
		errUse.HttpCode = http.StatusInternalServerError
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	result, err := client.WriterClient().AddWrite(context.Background(), &writeService)
	if err != nil {
		h.logger.Errorf("Не возможно подключится к сервису: %v", err)
		errUse.ErrorMsg = fmt.Sprintf("Не возможно подключится к сервису: %v", err)
		errUse.ErrorTitle = "Не возможно подключится к сервису"
		errUse.HttpCode = http.StatusInternalServerError
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) DeleteWriter(c *gin.Context) {
	errUse := models.UsesMsgError{}

	idSm, err := strconv.Atoi(c.Param("id_sm"))
	if err != nil {
		errUse.ErrorMsg = "invalid id_sm parameter"
		errUse.ErrorTitle = "Неверный параметр id_sm"
		errUse.HttpCode = http.StatusBadRequest
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	idWr, err := strconv.Atoi(c.Param("id_wr"))
	if err != nil {
		errUse.ErrorMsg = "invalid id_wr parameter"
		errUse.ErrorTitle = "Неверный параметр id_wr"
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

	manager, err := client.GetServiceManager(c.Request.Context())
	if err != nil {
		h.logger.Errorf("Не возможно подключится к сервису: %v", err)
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Не возможно подключится к сервису"
		errUse.HttpCode = http.StatusInternalServerError
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	if manager.TypeSm != "WRITER" {
		h.logger.Errorf("Не верный тип сервиса %s", manager.TypeSm)
		errUse.ErrorMsg = fmt.Sprintf("Invalid service type WRITER: %s", manager.TypeSm)
		errUse.ErrorTitle = "Не верный тип сервиса на выбранном порту"
		errUse.HttpCode = http.StatusInternalServerError
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	writeService := &pb.WriteService{
		IdWriter: int32(idWr),
		IdSm:     int32(idSm),
	}

	result, err := client.WriterClient().DeleteWrite(context.Background(), writeService)
	if err != nil {
		h.logger.Errorf("Не возможно подключится к сервису: %v", err)
		errUse.ErrorMsg = fmt.Sprintf("Не возможно подключится к сервису: %v", err)
		errUse.ErrorTitle = "Не возможно подключится к сервису"
		errUse.HttpCode = http.StatusInternalServerError
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) ChangeActiveWr(c *gin.Context) {
	errUse := models.UsesMsgError{}

	idSm, err := strconv.Atoi(c.Param("id_sm"))
	if err != nil {
		errUse.ErrorMsg = "invalid id_sm parameter"
		errUse.ErrorTitle = "Неверный параметр id_sm"
		errUse.HttpCode = http.StatusBadRequest
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	idWr, err := strconv.Atoi(c.Param("id_wr"))
	if err != nil {
		errUse.ErrorMsg = "invalid id_wr parameter"
		errUse.ErrorTitle = "Неверный параметр id_wr"
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

	manager, err := client.GetServiceManager(c.Request.Context())
	if err != nil {
		h.logger.Errorf("Не возможно подключится к сервису: %v", err)
		errUse.ErrorMsg = err.Error()
		errUse.ErrorTitle = "Не возможно подключится к сервису"
		errUse.HttpCode = http.StatusInternalServerError
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	if manager.TypeSm != "WRITER" {
		h.logger.Errorf("Не верный тип сервиса %s", manager.TypeSm)
		errUse.ErrorMsg = fmt.Sprintf("Invalid service type WRITER: %s", manager.TypeSm)
		errUse.ErrorTitle = "Не верный тип сервиса на выбранном порту"
		errUse.HttpCode = http.StatusInternalServerError
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	setWrite := &pb.SetWrite{
		IdWriter: int32(idWr),
		IdSm:     int32(idSm),
	}

	// Проверяем текущий статус и вызываем соответствующий метод
	// Для простоты предположим, что PATCH переключает статус
	list, err := client.WriterClient().ListWrites(context.Background(), &emptypb.Empty{})
	if err != nil {
		h.logger.Errorf("Не возможно получить список записей: %v", err)
		errUse.ErrorMsg = fmt.Sprintf("Не возможно получить список записей: %v", err)
		errUse.ErrorTitle = "Не возможно получить список записей"
		errUse.HttpCode = http.StatusInternalServerError
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	found := false
	isActive := false
	for _, write := range list.ListWrite {
		if int(write.IdWriter) == idWr && int(write.IdSm) == idSm {
			found = true
			isActive = write.IsActive
			break
		}
	}

	if !found {
		errUse.ErrorMsg = "Writer not found"
		errUse.ErrorTitle = "Записывающее устройство не найдено"
		errUse.HttpCode = http.StatusNotFound
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	var result *pb.StateServ
	if isActive {
		result, err = client.WriterClient().DownWrite(context.Background(), setWrite)
	} else {
		result, err = client.WriterClient().UpWrite(context.Background(), setWrite)
	}

	if err != nil {
		h.logger.Errorf("Не возможно подключится к сервису: %v", err)
		errUse.ErrorMsg = fmt.Sprintf("Не возможно подключится к сервису: %v", err)
		errUse.ErrorTitle = "Не возможно подключится к сервису"
		errUse.HttpCode = http.StatusInternalServerError
		c.JSON(errUse.HttpCode, errUse)
		return
	}

	c.JSON(http.StatusOK, result)
}
