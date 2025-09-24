package config

import (
	"os"

	"github.com/naoina/toml"
)

// представляет  конфигурацию записи данных
type ControlWriter struct {
	Description string    `toml:"description"`
	Name        string    `toml:"name"`
	IdSm        int       `toml:"id_sm"`
	IpSm        string    `toml:"ip_sm"`
	GrpcPort    int       `toml:"grpc_port"`
	MetricPort  int       `toml:"metric_port"`
	NatsAddress string    `toml:"nats_address"`
	NatsTopic   string    `toml:"nats_topic"`
	NatsTimeOut int       `toml:"nats_timeout"` // в секундах
	IdWriter    int32     `toml:"id_writer"`
	DbIp        string    `toml:"db_ip"`
	DbPort      int32     `toml:"db_port"`
	DbName      string    `toml:"db_name"`
	DbUser      string    `toml:"db_user"`
	DbPass      string    `toml:"db_pass"`
	DbTable     string    `toml:"db_table"`
	LogConfig   ConfigLog `toml:"log_config"`
	Filename    string    `toml:"-"`
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
