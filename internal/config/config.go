package config

import (
	"os"

	yaml "sigs.k8s.io/yaml/goyaml.v3"
)

type ServerConfig struct {
	LogLevel    string    `yaml:"log_level"`
	Addr        string    `yaml:"addr"`
	DataDir     string    `yaml:"data_dir"`
	Auth        string    `yaml:"auth"`
	Cloudflared *bool     `yaml:"cloudflared"`
	RPC         string    `yaml:"rpc"`
	Playlist    *[]string `yaml:"playlist"`
}

func boolPtr(b bool) *bool {
	return &b
}

// TODO do this with tag?
var defaultServerConfig = ServerConfig{
	LogLevel:    "info",
	Addr:        "127.0.0.1:3000",
	DataDir:     "data",
	Auth:        "",
	Cloudflared: boolPtr(true),
	RPC:         "127.0.0.1:3001",
	Playlist: &[]string{
		"trending",
	},
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

	if s.Cloudflared == nil {
		s.Cloudflared = defaultServerConfig.Cloudflared
	}

	if s.RPC == "" {
		s.RPC = defaultServerConfig.RPC
	}

	if s.Playlist == nil {
		s.Playlist = defaultServerConfig.Playlist
	}

	return &s, nil
}
