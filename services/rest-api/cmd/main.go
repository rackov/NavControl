package main

import (
	"flag"
	"fmt"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/rackov/NavControl/pkg/config"
	"github.com/rackov/NavControl/pkg/logger"
	"github.com/rackov/NavControl/pkg/monitoring"
	"github.com/rackov/NavControl/services/rest-api/internal/handlers"
)

func restMan(router *gin.Engine, h *handlers.Handler) {
	controller := router.Group("/api/v1/controller")
	{
		// ServiceModule
		controller.GET("/sm/level", h.GetLogLevel)
		controller.POST("/sm/level", h.SetLogLevel)
		controller.GET("/sm/log", h.ReadLogs)
		controller.GET("/sm", h.GetServiceModules)
		controller.POST("/sm", h.CreateServiceModule)
		controller.DELETE("/sm/:id_sm", h.DeleteServiceModule)
		controller.GET("/sm/protocol", h.GetProtocols)
		// Receiver
		controller.GET("/receiver", h.ListAllReceiver)
		controller.GET("/receiver/:id_sm", h.ListPorts)
		controller.POST("/receiver", h.AddPort)
		controller.PATCH("/receiver/:id_sm/:id_rec", h.ChangeActive) // ClosePort, OpenPort
		controller.DELETE("/receiver/:id_sm/:id_rec", h.DeletePort)
		controller.GET("/receiver/client/:id_sm", h.GetConnectedClients)
		controller.POST("/receiver/client/:id_sm", h.DisconnectClient)
		// Retranslator
		controller.GET("/retranslator", h.ListAllRetranslator)
		controller.GET("/retranslator/:id_sm", h.ListClient)
		controller.POST("/retranslator", h.AddClient)
		controller.PUT("/retranslator", h.UpdateClient)
		controller.PATCH("/retranslator/:id_sm/:id_ret", h.ChangeActiveClient) //UpClient DownClient
		controller.DELETE("/retranslator/:id_sm/:id_ret", h.DeleteClient)
		controller.GET("/retranslator/devices/:id_sm", h.ListDevices)
		//writer
		// router.GET("/writer/:id_sm", h.ListWriters)
		// router.POST("/writer", h.AddWriter)
		// router.DELETE("/writer/:id_sm/:id_wr", h.DeleteWriter)
		// router.PATCH("/writer/:id_sm/:id_wr", h.ChangeActiveWr) //DownWrite UpWrite

	}
}

func main() {

	configPath := flag.String("config", "NavControl/cfg/restapi.toml", "путь к файлу конфигурации")
	flag.Parse()

	crest, err := config.NewCfgRestApi(*configPath)

	if err != nil {
		fmt.Printf("Failed to load config: %v", err)
		return
	}

	log, err := logger.NewLogger(crest.LogConfig)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)

	}

	log.Info("Starting monitorig service")
	// Запускаем сервер для сбора метрик
	go func() {
		if err := monitoring.StartMetricsServer(crest.MetricPort); err != nil {
			log.Printf("Failed to start metrics server: %v\n", err)
		}
	}()
	router := gin.New()
	// router.Use(func(c *gin.Context) {
	// 	start := time.Now()
	// 	path := c.Request.URL.Path

	// 	c.Next()

	// 	latency := time.Since(start)
	// 	clientIP := c.ClientIP()
	// 	method := c.Request.Method
	// 	statusCode := c.Writer.Status()

	// 	log.Infof("%s %s %s %d %v",
	// 		method,
	// 		path,
	// 		clientIP,
	// 		statusCode,
	// 		latency,
	// 	)
	// })
	con_cors := crest.RetCors()
	router.Use(cors.New(con_cors))

	// Инициализация обработчиков с передачей логгера
	h, err := handlers.NewHandler(crest, log)
	if err != nil {
		log.Fatalf("Failed to create handlers: %v", err)
	}
	defer h.Close()

	restMan(router, h)

	addr := fmt.Sprintf(":%d", crest.RestPort)
	log.Infof("Starting server on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

}
