package main

import (
	"log"

	"github.com/rackov/NavControl/pkg/config"
	"github.com/rackov/NavControl/pkg/monitoring"
	"github.com/rackov/NavControl/services/receiver/internal/portmanager"
)

func main() {
	// Создаем менеджер портов
	config := config.NewServices()

	pm := portmanager.NewPortManager(config)

	go func() {
		if err := monitoring.StartMetricsServer(9092); err != nil {
			log.Printf("Failed to start metrics server: %v", err)
		}
	}()

	// Регистрируем наш сервис
	grpcService := portmanager.NewGRPCServer(pm)

	defer pm.Stop()
	go func() {
		if err := pm.Start(); err != nil {
			log.Fatalf("Failed to start port manager: %v", err)
		}
	}()

	if err := grpcService.StartGRPCServer(50051); err != nil {
		log.Fatalf("Failed to start gRPC server: %v", err)
	}

}
