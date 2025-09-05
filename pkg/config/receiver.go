package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/naoina/toml"
)

// Receiver представляет отдельный сервис приемника данных
type Receiver struct {
	IdReceiver       int    `tmol:"id_receiver" json:"id_receiver"`
	IdSm             int    `tmol:"id_sm" json:"id_sm"`
	Active           bool   `tmol:"active" json:"active"`
	Name             string `tmol:"name" json:"name"`
	PortReceiver     int    `tmol:"port_receiver" json:"port_receiver"`
	Protocol         string `tmol:"protocol" json:"protocol"`
	Status           string `tmol:"status" json:"status"`
	Description      string `tmol:"description" json:"description"`
	ConnectionsCount int    `tmol:"-" json:"connections_count"`
}

// Services представляет всю конфигурацию приемника данных
type ControlResiver struct {
	GrpcPort    int        `toml:"grpc_port"`
	MetricPort  int        `toml:"metric_port"`
	NatsAddress string     `toml:"nats_address"`
	NatsTopic   string     `toml:"nats_topic"`
	NatsTimeOut int        `toml:"nats_timeout"` // в секундах
	LogConfig   ConfigLog  `toml:"log_config"`
	Receivers   []Receiver `toml:"receivers"`
	Filename    string     `toml:"-"`
}

// NewServices создает новую конфигурацию с значениями по умолчанию
func NewServices() *ControlResiver {
	return &ControlResiver{
		GrpcPort:    50051,
		MetricPort:  9092,
		NatsAddress: "nats://192.168.194.242:4222",
		NatsTopic:   "nav.data",
		NatsTimeOut: 5,
		LogConfig: ConfigLog{
			LogLevel:    "info",
			LogFilePath: "logs/app.log",
			MaxSize:     100,
			MaxBackups:  3,
			MaxAge:      28,
			Compress:    true,
		},
		Receivers: []Receiver{
			{
				PortReceiver: 8080, // Инициализация поля PortReceiver значением 8080
				IdReceiver:   1,
				IdSm:         1,
				Active:       false,
				Name:         "",
				Protocol:     "EGTS",
				Status:       "",
				Description:  "",
			},
		},
	}
}
func (s *ControlResiver) LoadConfig(fname string) error {
	f, err := os.Open(fname)

	if err != nil {
		return err
	}
	defer f.Close()

	if err := toml.NewDecoder(f).Decode(s); err != nil {
		return err
	}
	return err
}
func (s *ControlResiver) SaveCfg(fname string) error {
	f, err := os.Create(fname)

	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(s)
}

func (s *ControlResiver) TrimAddress() (err error, ip string, port int) {
	natsAddress := s.NatsAddress

	// Удаляем префикс "nats://"
	address := strings.TrimPrefix(natsAddress, "nats://")

	// Разделяем строку на IP и порт
	host, portStr, err := net.SplitHostPort(address)
	if err != nil {
		fmt.Printf("Ошибка при разборе адреса: %v\n", err)
		return err, "", 0
	}

	// Преобразуем порт из строки в int
	port, err = strconv.Atoi(portStr)
	if err != nil {
		fmt.Printf("Ошибка при преобразовании порта: %v\n", err)
		return err, host, 0
	}

	return err, host, port

}
