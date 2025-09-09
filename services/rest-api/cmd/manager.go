package main

import (
	"github.com/gin-gonic/gin"
	"github.com/rackov/NavControl/services/rest-api/internal/handlers"
)

func restMan(router *gin.Engine, h *handlers.Handler) {

	router.GET("/controller/sm", h.GetAllServices)
	// router.POST("/controller/sm", h.CreateService)
	// router.DELETE("/controller/sm/:id_sm", DeleteService)
	// router.PUT("/controller/sm", UpdateService)
}
