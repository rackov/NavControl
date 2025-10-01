package config

import (
	"os"

	"github.com/naoina/toml"
)

type Writer struct {
	Description string `toml:"description"`
	Name        string `toml:"name"`
	IdWriter    int32  `toml:"id_writer"`
	DbIp        string `toml:"ip_db"`
	DbPort      int32  `toml:"port_db"`
	DbName      string `toml:"name_db"`
	DbUser      string `toml:"login"`
	DbPass      string `toml:"passw_db"`
	DbTable     string `toml:"table_db"`
}

// представляет  конфигурацию записи данных
type ControlWriter struct {
	IdSm        int    `toml:"id_sm"`
	Description string `toml:"description"`
	Name        string `toml:"name"`
	GrpcPort    int    `toml:"grpc_port"`
	MetricPort  int    `toml:"metric_port"`
	NatsAddress string `toml:"nats_address"`
	NatsTopic   string `toml:"nats_topic"`
	// NatsTimeOut int       `toml:"nats_timeout"` // в секундах
	LogConfig ConfigLog `toml:"log_config"`
	Writers   []Writer  `toml:"writers"`
	Filename  string    `toml:"-"`
}

func NewWriter() *ControlWriter {
	return &ControlWriter{}
}
func (s *ControlWriter) LoadConfig(fname string) error {
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
func (s *ControlWriter) SaveConfig() error {
	f, err := os.Create(s.Filename)

	if err != nil {
		return err
	}
	defer f.Close()

	if err := toml.NewEncoder(f).Encode(s); err != nil {
		return err
	}
	return err
}
