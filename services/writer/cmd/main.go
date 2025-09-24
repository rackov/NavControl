package main

import (
	"flag"
	"fmt"

	"github.com/rackov/NavControl/pkg/config"
	"github.com/rackov/NavControl/pkg/logger"
	"github.com/rackov/NavControl/pkg/monitoring"
	"github.com/rackov/NavControl/services/writer/internal/mgrpc"
	"github.com/rackov/NavControl/services/writer/internal/wrdbnats"
)

func main() {

	configPath := flag.String("config", "NavControl/cfg/writer.toml", "путь к файлу конфигурации")
	flag.Parse()

	config := config.NewWriter()
	if err := config.LoadConfig(*configPath); err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	log, err := logger.NewLogger(config.LogConfig)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)

	}

	hub := wrdbnats.NewHubServer(config, log)
	go hub.Run()
	hub.AddService()

	// Запускаем сервер для сбора метрик
	go func() {
		if err := monitoring.StartMetricsServer(config.MetricPort); err != nil {
			log.Printf("Failed to start metrics server: %v\n", err)
		}
	}()

	// Регистрируем наш сервис
	grpcService := mgrpc.NewGRPCServer(hub)

	// Запускаем сервер gRPC
	if err := grpcService.StartGRPCServer(config.GrpcPort); err != nil {
		log.Fatalf("Failed to start gRPC server: %v", err)
	}

}
