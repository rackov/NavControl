package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/rackov/NavControl/pkg/models"
	"github.com/rackov/NavControl/services/receiver/internal/handler/arnavi"
	"github.com/rackov/NavControl/services/receiver/internal/portmanager"
)

func main() {
	// Создаем менеджер портов
	pm := portmanager.NewPortManager()

	// Создаем два экземпляра протокола Arnavi для разных портов
	arnaviProtocol1 := arnavi.NewArnaviProtocol()
	arnaviProtocol2 := arnavi.NewArnaviProtocol()

	// Добавляем протоколы на разные порты
	if err := pm.AddProtocol(8080, arnaviProtocol1); err != nil {
		log.Fatalf("Failed to add Arnavi protocol on port 8080: %v", err)
	}

	if err := pm.AddProtocol(8081, arnaviProtocol2); err != nil {
		log.Fatalf("Failed to add Arnavi protocol on port 8081: %v", err)
	}

	// Запускаем протоколы на их портах
	if err := pm.StartProtocol(8080); err != nil {
		log.Fatalf("Failed to start Arnavi protocol on port 8080: %v", err)
	}

	if err := pm.StartProtocol(8081); err != nil {
		log.Fatalf("Failed to start Arnavi protocol on port 8081: %v", err)
	}

	// Выводим информацию о запущенных протоколах
	info := pm.GetProtocolsInfo()
	for port, protocolName := range info {
		log.Printf("Protocol %s is running on port %d", protocolName, port)
	}

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
