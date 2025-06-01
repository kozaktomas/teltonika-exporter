package main

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Devices []struct {
		Name     string        `yaml:"name,omitempty"`
		Schema   string        `yaml:"schema,omitempty"`
		Host     string        `yaml:"host"`
		Timeout  time.Duration `yaml:"timeout,omitempty"`
		Username string        `yaml:"username"`
		Password string        `yaml:"password"`
		Collect  []string      `yaml:"collect"`
	} `yaml:"devices"`
	MacTranslations   map[string]string `yaml:"mac_translations,omitempty"`
	RadioTranslations map[string]string `yaml:"radio_translations,omitempty"`
}

func ParseConfig(file string) (*Config, error) {
	config := &Config{}
	configContent, err := os.ReadFile(file) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	err = yaml.Unmarshal(configContent, config)
	if err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	// reasonable defaults
	for key, device := range config.Devices {
		if device.Name == "" {
			config.Devices[key].Name = device.Host
		}

		if device.Schema == "" {
			config.Devices[key].Schema = "https"
		}

		if device.Timeout == 0 {
			config.Devices[key].Timeout = 10 * time.Second
		}
	}

	return config, nil
}
