package config

// Receiver представляет отдельный сервис (ранее Service)
type Receiver struct {
	ID       int    `toml:"id"`
	Name     string `toml:"name"`
	Port     int32  `toml:"port"`
	Protocol string `toml:"protocol"`
	Active   bool   `toml:"active"`
	Status   string `toml:"status"`
}

// Services представляет всю конфигурацию (ранее Config)
type Services struct {
	NatsAddress string     `toml:"nats_address"`
	NatsTopic   string     `toml:"nats_topic"`
	NatsTimeOut int        `toml:"nats_timeout"` // в секундах
	LogConfig   ConfigLog  `toml:"log_config"`
	Receivers   []Receiver `toml:"receivers"`
}

// NewServices создает новую конфигурацию с значениями по умолчанию
func NewServices() *Services {
	return &Services{
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
				ID:       1,
				Name:     "example-receiver",
				Port:     8080,
				Protocol: "Arnavi",
				Active:   true,
			},
		},
	}
}
