package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tristendillon/conduit/core/logger"
	"gopkg.in/yaml.v3"
)

type Config struct {
	AppName string  `yaml:"app_name"`
	Server  Server  `yaml:"server"`
	Codegen Codegen `yaml:"codegen"`
}

type Server struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type Codegen struct {
	Go struct {
		Output string `yaml:"output"`
	} `yaml:"go"`
	Typescript struct {
		Output string `yaml:"output"`
	} `yaml:"typescript"`
}

func Default() *Config {
	return &Config{
		AppName: "conduit",
		Server: Server{
			Host: "localhost",
			Port: 8080,
		},
	}
}

func Load() (*Config, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("cannot determine working dir: %w", err)
	}

	paths := []string{
		filepath.Join(wd, "conduit.yaml"),
	}

	var filePath string
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			filePath = p
			break
		}
	}

	if filePath == "" {
		logger.Debug("No config file found, using default config")
		config := Default()
		return config, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", filePath, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse yaml: %w", err)
	}
	logger.Debug("Config file found: %s", filePath)
	logger.Debug("Config: %+v", cfg)

	return &cfg, nil
}
