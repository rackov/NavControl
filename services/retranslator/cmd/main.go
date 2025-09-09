package main

import (
	"github.com/rackov/NavControl/pkg/config"
	"github.com/rackov/NavControl/pkg/logger"
	"github.com/rackov/NavControl/pkg/monitoring"
	"github.com/rackov/NavControl/services/retranslator/internal/retgrpc"
)

func main() {

	//	configPath := flag.String("config", "NavControl/cfg/receiver.toml", "путь к файлу конфигурации")
	//	flag.Parse()

	cfg := config.ConfigLog{
		LogLevel:    "info",
		LogFilePath: "logs/restapi.log",
		MaxSize:     100,
		MaxBackups:  3,
		MaxAge:      28,
		Compress:    true,
	}

	log, err := logger.NewLogger(cfg)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)

	}

	// Запускаем сервер для сбора метрик
	go func() {
		if err := monitoring.StartMetricsServer(9998); err != nil {
			log.Printf("Failed to start metrics server: %v\n", err)
		}
	}()

	// Регистрируем наш сервис
	grpcService := retgrpc.NewGRPCServer(log)

	// Запускаем сервер gRPC
	if err := grpcService.StartGRPCServer(50002); err != nil {
		log.Fatalf("Failed to start gRPC server: %v", err)
	}

}
