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

	// // Добавляем порт с протоколом Arnavi
	// err := pm.AddPort(8080, "Arnavi", true, "Main Arnavi Port")
	// if err != nil {
	// 	log.Fatalf("Failed to add Arnavi port: %v", err)
	// }
	// 4. Инициализация метрик Prometheus
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

	// Запускаем gRPC сервер в отдельной горутине
	// go func() {
	if err := grpcService.StartGRPCServer(50051); err != nil {
		log.Fatalf("Failed to start gRPC server: %v", err)
	}
	// }()

	// // Запускаем обработчик данных в отдельной горутине
	// go handleData(pm.GetDataChan())

	// // Настраиваем перехват сигналов для корректной остановки
	// sigChan := make(chan os.Signal, 1)
	// signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// // Ожидаем сигнал остановки
	// <-sigChan
	// log.Println("Received shutdown signal, stopping...")

	// // Останавливаем менеджер портов
	// pm.Stop()

	// log.Println("Service stopped")
}

// handleData обрабатывает навигационные данные
// func handleData(dataChan <-chan models.NavRecord) {
// 	for record := range dataChan {
// 		log.Printf("Received navigation record: %+v", record)
// 		// Здесь можно добавить дополнительную обработку данных
// 	}
// }
