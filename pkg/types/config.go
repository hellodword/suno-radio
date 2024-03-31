package types

type ServerConfig struct {
	LogLevel string `yaml:"log_level"`
	Addr     string `yaml:"addr"`
	DataDir  string `yaml:"data_dir"`
}
