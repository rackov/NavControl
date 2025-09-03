package main

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/rackov/NavControl/pkg/logger"
	"github.com/rackov/NavControl/pkg/models"
	"github.com/rackov/NavControl/services/receiver/internal/portmanager"
)

func main() {
	// Создаем менеджер портов
	logDir := "/home/vladimir/go/project/NavControl/services/receiver/cmd/logs/"
	logPath := filepath.Join(logDir, "receiver.log")
	cfg := logger.Config{LogLevel: "debug",
		LogFilePath: logPath, MaxSize: 100, MaxBackups: 3, MaxAge: 30, Compress: true}
	log.Println(logDir)
	pm := portmanager.NewPortManager(cfg)

	// Добавляем порт с протоколом Arnavi
	err := pm.AddPort(8080, "Arnavi", true, "Main Arnavi Port")
	if err != nil {
		log.Fatalf("Failed to add Arnavi port: %v", err)
	}

	// Создаем и запускаем gRPC сервер
	// grpcServer := portmanager.NewGRPCServer(pm)

	// Регистрируем наш сервис
	grpcService := portmanager.NewGRPCServer(pm)

	// Запускаем gRPC сервер в отдельной горутине
	go func() {
		if err := grpcService.StartGRPCServer(50051); err != nil {
			log.Fatalf("Failed to start gRPC server: %v", err)
		}
	}()

	// Запускаем обработчик данных в отдельной горутине
	go handleData(pm.GetDataChan())

	// Настраиваем перехват сигналов для корректной остановки
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Ожидаем сигнал остановки
	<-sigChan
	log.Println("Received shutdown signal, stopping...")

	// Останавливаем менеджер портов
	pm.Stop()

	log.Println("Service stopped")
}

// handleData обрабатывает навигационные данные
func handleData(dataChan <-chan models.NavRecord) {
	for record := range dataChan {
		log.Printf("Received navigation record: %+v", record)
		// Здесь можно добавить дополнительную обработку данных
	}
}
