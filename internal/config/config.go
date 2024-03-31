package config

import (
	"os"

	yaml "sigs.k8s.io/yaml/goyaml.v3"
)

type ServerConfig struct {
	LogLevel string `yaml:"log_level"`
	Addr     string `yaml:"addr"`
	DataDir  string `yaml:"data_dir"`
	Auth     string `yaml:"auth"`
}

// TODO do this with tag?
var defaultServerConfig = ServerConfig{
	LogLevel: "info",
	Addr:     "127.0.0.1:3000",
	DataDir:  "data",
	Auth:     "",
}

func LoadFromYaml(p string) (*ServerConfig, error) {
	f, err := os.Open(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &defaultServerConfig, nil
		} else {
			return nil, err
		}
	}
	defer f.Close()

	var s ServerConfig
	d := yaml.NewDecoder(f)
	err = d.Decode(&s)
	if err != nil {
		return nil, err
	}

	if s.Addr == "" {
		s.Addr = defaultServerConfig.Addr
	}

	if s.DataDir == "" {
		s.DataDir = defaultServerConfig.DataDir
	}

	if s.LogLevel == "" {
		s.LogLevel = defaultServerConfig.LogLevel
	}

	return &s, nil
}
