package main

import (
	"flag"
	"log"

	"github.com/rackov/NavControl/pkg/config"
	"github.com/rackov/NavControl/pkg/monitoring"
	"github.com/rackov/NavControl/services/receiver/internal/portmanager"
)

func main() {

	configPath := flag.String("config", "NavControl/cfg/receiver.toml", "путь к файлу конфигурации")
	flag.Parse()

	// Конфигурация портов
	config := config.NewServices()
	if err := config.LoadConfig(*configPath); err != nil {
		log.Fatalf("Failed to load config: %v\n", err)
	}
	// Создаем менеджер портов
	config.Filename = *configPath
	pm := portmanager.NewPortManager(config)
	defer pm.Stop()

	// Запускаем сервер для сбора метрик
	go func() {
		if err := monitoring.StartMetricsServer(config.MetricPort); err != nil {
			log.Printf("Failed to start metrics server: %v\n", err)
		}
	}()

	// Регистрируем наш сервис
	grpcService := portmanager.NewGRPCServer(pm)

	// Запускаем менеджер портов
	go func() {
		if err := pm.Start(); err != nil {
			log.Fatalf("Failed to start port manager: %v", err)
		}
	}()

	// Запускаем сервер gRPC
	if err := grpcService.StartGRPCServer(config.GrpcPort); err != nil {
		log.Fatalf("Failed to start gRPC server: %v", err)
	}

}

/*
	// Получаем путь к исполняемому файлу
	exePath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}

	// Получаем директорию исполняемого файла
	exeDir := filepath.Dir(exePath)

	// Строим путь к файлу конфигурации относительно директории исполняемого файла
	configPath := filepath.Join(exeDir, "receiver.toml")


*/
