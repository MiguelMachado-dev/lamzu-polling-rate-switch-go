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
	Games              []string      `yaml:"games"` // Legacy support
	Steam              *SteamConfig  `yaml:"steam,omitempty"`
	DetectedGames      []Game        `yaml:"detected_games,omitempty"`
	CustomGames        []CustomGame  `yaml:"custom_games,omitempty"`
}

type SteamConfig struct {
	InstallPath string    `yaml:"install_path"`
	Libraries   []Library `yaml:"libraries"`
	LastScan    time.Time `yaml:"last_scan"`
}

type Library struct {
	Path  string `yaml:"path"`
	Label string `yaml:"label"`
}

type Game struct {
	Name        string `yaml:"name"`
	AppID       string `yaml:"app_id"`
	Executable  string `yaml:"executable"`
	InstallPath string `yaml:"install_path"`
	Library     string `yaml:"library"`
	SizeMB      int64  `yaml:"size_mb"`
}

type CustomGame struct {
	Name       string `yaml:"name"`
	Executable string `yaml:"executable"`
	Path       string `yaml:"path"`
}

func LoadConfig(filename string) (*Config, error) {
	// Default configuration
	config := &Config{
		DefaultPollingRate: 1000,
		GamePollingRate:    2000,
		CheckInterval:      2 * time.Second,
		Steam: &SteamConfig{
			InstallPath: "",
			Libraries:   []Library{},
			LastScan:    time.Time{},
		},
		DetectedGames: []Game{},
		CustomGames: []CustomGame{
			{Name: "Hunt: Showdown", Executable: "HuntGame.exe", Path: ""},
			{Name: "Dune: Awakening", Executable: "DuneSandbox-Wi.exe", Path: ""},
			{Name: "Elden Ring", Executable: "eldenring.exe", Path: ""},
			{Name: "Counter-Strike 2", Executable: "cs2.exe", Path: ""},
			{Name: "VALORANT", Executable: "valorant.exe", Path: ""},
			{Name: "Apex Legends", Executable: "ApexLegends.exe", Path: ""},
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
