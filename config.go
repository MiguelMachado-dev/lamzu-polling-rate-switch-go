package main

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	DefaultPollingRate int           `yaml:"default_polling_rate"`
	GamePollingRate    int           `yaml:"game_polling_rate"`
	CheckInterval      time.Duration `yaml:"check_interval"`
	Games              []string      `yaml:"games"`
}

func LoadConfig(filename string) (*Config, error) {
	// Default configuration
	config := &Config{
		DefaultPollingRate: 1000,
		GamePollingRate:    2000,
		CheckInterval:      2 * time.Second,
		Games: []string{
			"HuntGame.exe",
			"DuneSandbox-Wi.exe",
			"eldenring.exe",
			"cs2.exe",
			"valorant.exe",
			"ApexLegends.exe",
		},
	}

	// Try to load from file
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			// Create default config file
			if err := SaveConfig(config, filename); err != nil {
				return nil, fmt.Errorf("failed to create default config: %w", err)
			}
			fmt.Printf("📄 Created default config file: %s\n", filename)
			return config, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

func SaveConfig(config *Config, filename string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(filename, data, 0644)
}
