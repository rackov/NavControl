package config

// Config содержит все необходимые настройки для инициализации логгера.
type ConfigLog struct {
	LogLevel    string `toml:"log_level"`
	LogFilePath string `toml:"log_file_path"`
	MaxSize     int    `toml:"max_size"`
	MaxBackups  int    `toml:"max_backups"`
	MaxAge      int    `toml:"max_age"`
	Compress    bool   `toml:"compress"`
}
