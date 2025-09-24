package main

import (
	"flag"
	"fmt"

	"github.com/rackov/NavControl/pkg/config"
	"github.com/rackov/NavControl/pkg/logger"
	"github.com/rackov/NavControl/pkg/monitoring"
	"github.com/rackov/NavControl/services/retranslator/internal/retgrpc"
	"github.com/rackov/NavControl/services/retranslator/internal/sendegts"
)

func main() {

	configPath := flag.String("config", "NavControl/cfg/retranslator.toml", "путь к файлу конфигурации")
	flag.Parse()

	cfg := config.NewRetranslator()
	if err := cfg.LoadConfig(*configPath); err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		return
	}

	log, err := logger.NewLogger(cfg.LogConfig)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)

	}
	log.Info("Starting retranslator service")
	retran := sendegts.NewRetranslator(cfg, log)
	go retran.Run()
	defer retran.Stop()

	// Запускаем сервер для сбора метрик
	go func() {
		log.Infof("Starting metrics server, port: %d", cfg.MetricPort)
		if err := monitoring.StartMetricsServer(cfg.MetricPort); err != nil {
			log.Printf("Failed to start metrics server: %v\n", err)
		}
	}()

	// Регистрируем наш сервис
	grpcService := retgrpc.NewGRPCServer(log, retran)

	// Запускаем сервер gRPC
	log.Infof("Starting gRPC server, port: %d", cfg.GrpcPort)
	if err := grpcService.StartGRPCServer(cfg.GrpcPort); err != nil {
		log.Fatalf("Failed to start gRPC server: %v", err)
	}

}
