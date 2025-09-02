package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/rackov/NavControl/pkg/models"
	"github.com/rackov/NavControl/proto"
	"github.com/rackov/NavControl/services/receiver/internal/portmanager"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Создаем менеджер портов
	pm := portmanager.NewPortManager()

	// Добавляем порт с протоколом Arnavi
	err := pm.AddPort(8080, "Arnavi", true, "Main Arnavi Port")
	if err != nil {
		log.Fatalf("Failed to add Arnavi port: %v", err)
	}

	// Создаем и запускаем gRPC сервер
	// grpcServer := portmanager.NewGRPCServer(pm)

	// Создаем gRPC сервер ---------------  Регистрация для проверки
	grpcServer := grpc.NewServer()

	// Регистрируем наш сервис
	grpcService := portmanager.NewGRPCServer(pm)
	proto.RegisterReceiverControlServer(grpcServer, grpcService)

	// Включаем reflection API
	reflection.Register(grpcServer)
	// ---------------  Регистрация для проверки

	// Создаем слушателя для gRPC сервера
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	go func() {
		log.Printf("gRPC server started on port 50051")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
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
