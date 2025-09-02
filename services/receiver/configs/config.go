package configs

import (
	"sync"

	"github.com/rackov/NavControl/proto"
)

// Config описывает полную конфигурацию сервиса receiver
type Config struct {
	GRPCPort       uint32                 `toml:"grpc_port"`
	MonitoringPort uint32                 `toml:"monitoring_port"`
	Ports          []proto.PortDefinition `toml:"ports"` // Список всех сконфигурированных портов
	configPath     string                 // Путь к файлу для сохранения
	mu             sync.Mutex
}

// LoadConfig загружает конфигурацию из TOML-файла
