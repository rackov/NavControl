package config

import (
	"os"

	"github.com/gin-contrib/cors"
	"github.com/naoina/toml"
)

// type CfgCors struct {
// 	AllowAllOrigins           bool
// 	AllowOrigins              []string
// 	AllowMethods              []string
// 	AllowPrivateNetwork       bool
// 	AllowHeaders              []string
// 	AllowCredentials          bool
// 	ExposeHeaders             []string
// 	AllowWildcard             bool
// 	AllowBrowserExtensions    bool
// 	CustomSchemas             []string
// 	AllowWebSockets           bool
// 	AllowFiles                bool
// 	OptionsResponseStatusCode int
// }

type ServiceManager struct {
	IdSm        int    `json:"id_sm"`
	IpSm        string `json:"ip_sm"`
	PortSm      int    `json:"port_sm"`
	Name        string `json:"name"`
	TypeSm      string `json:"type_sm"`
	IpBroker    string `json:"ip_broker"`    // адрес брокера
	PortBroker  int    `json:"port_broker"`  // порт брокера
	TopicBroker string `json:"topic_broker"` // топик брокера
	Active      bool   `json:"active"`
	Status      string `json:"status"`
	Description string `json:"description"` // какое-либо описание
	LogLevel    string `json:"log_level"`
	ErrorMsg    string `json:"error_msg" toml:"-"`
}

type CfgRestApi struct {
	RestPort       int              `toml:"rest_port"`
	MetricPort     int              `toml:"metric_port"`
	Cors           cors.Config      `toml:"cors"`
	ServiceList    []ServiceManager `toml:"service_list"`
	FileConfigPath string           `toml:"-"`
	FileLogPath    string           `toml:"-"`
}

func (s *CfgRestApi) LoadConfig() error {
	f, err := os.Open(s.FileConfigPath)

	if err != nil {
		return err
	}
	defer f.Close()

	if err := toml.NewDecoder(f).Decode(s); err != nil {
		return err
	}
	return err
}
func (s *CfgRestApi) SaveCfg() error {
	f, err := os.Create(s.FileConfigPath)

	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(s)
}
