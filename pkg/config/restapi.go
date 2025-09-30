package config

import (
	"os"

	"github.com/gin-contrib/cors"
	"github.com/naoina/toml"
)

type CfgCors struct {
	AllowAllOrigins           bool
	AllowOrigins              []string
	AllowMethods              []string
	AllowPrivateNetwork       bool
	AllowHeaders              []string
	AllowCredentials          bool
	ExposeHeaders             []string
	AllowWildcard             bool
	AllowBrowserExtensions    bool
	CustomSchemas             []string
	AllowWebSockets           bool
	AllowFiles                bool
	OptionsResponseStatusCode int
}

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
	Cors           CfgCors          `toml:"cors"`
	LogConfig      ConfigLog        `toml:"log_config"`
	ServiceList    []ServiceManager `toml:"service_list"`
	FileConfigPath string           `toml:"-"`
	FileLogPath    string           `toml:"-"`
}

func NewCfgRestApi(filePath string) (*CfgRestApi, error) {
	cfg := CfgRestApi{
		FileConfigPath: filePath,
	}
	err := cfg.LoadConfig()
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
func (s *CfgRestApi) RetCors() cors.Config {

	return cors.Config{
		AllowOrigins:              s.Cors.AllowOrigins,
		AllowMethods:              s.Cors.AllowMethods,
		AllowAllOrigins:           s.Cors.AllowAllOrigins,
		AllowHeaders:              s.Cors.AllowHeaders,
		ExposeHeaders:             s.Cors.ExposeHeaders,
		AllowCredentials:          s.Cors.AllowCredentials,
		AllowPrivateNetwork:       s.Cors.AllowPrivateNetwork,
		AllowWildcard:             s.Cors.AllowWildcard,
		AllowBrowserExtensions:    s.Cors.AllowBrowserExtensions,
		AllowWebSockets:           s.Cors.AllowWebSockets,
		AllowFiles:                s.Cors.AllowFiles,
		CustomSchemas:             s.Cors.CustomSchemas,
		OptionsResponseStatusCode: s.Cors.OptionsResponseStatusCode,
	}

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
