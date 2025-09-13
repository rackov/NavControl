package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rackov/NavControl/pkg/config"
	"github.com/rackov/NavControl/pkg/logger"
	"github.com/rackov/NavControl/pkg/monitoring"
	"github.com/rackov/NavControl/services/rest-api/internal/handlers"
)

func main() {

	configPath := flag.String("config", "NavControl/cfg/restapi.toml", "путь к файлу конфигурации")
	flag.Parse()

	cfg := config.ConfigLog{
		LogLevel:    "info",
		LogFilePath: "logs/writer.log",
		MaxSize:     100,
		MaxBackups:  3,
		MaxAge:      28,
		Compress:    true,
	}

	log, err := logger.NewLogger(cfg)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)

	}
	crest := config.CfgRestApi{}
	if err = crest.LoadConfig(*configPath); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Info("Starting monitorig service")
	// Запускаем сервер для сбора метрик
	go func() {
		if err := monitoring.StartMetricsServer(crest.MetricPort); err != nil {
			log.Printf("Failed to start metrics server: %v\n", err)
		}
	}()
	router := gin.New()
	router.Use(func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		log.Infof("%s %s %s %d %v",
			method,
			path,
			clientIP,
			statusCode,
			latency,
		)
	})
	// Инициализация обработчиков с передачей логгера
	h, err := handlers.NewHandler(&crest, log)
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
