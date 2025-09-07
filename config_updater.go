package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ConfigUpdater handles updating the config.yaml file with Steam games
type ConfigUpdater struct {
	configPath string
}

// NewConfigUpdater creates a new config updater instance
func NewConfigUpdater(configPath string) *ConfigUpdater {
	return &ConfigUpdater{
		configPath: configPath,
	}
}

// UpdateWithSteamData updates the config with Steam installation and game data
func (cu *ConfigUpdater) UpdateWithSteamData(steamPath string, libraries []Library, games []Game) error {
	// Load existing config
	config, err := cu.loadExistingConfig()
	if err != nil {
		return fmt.Errorf("failed to load existing config: %w", err)
	}

	// Update Steam configuration
	config.Steam = &SteamConfig{
		InstallPath: steamPath,
		Libraries:   libraries,
		LastScan:    time.Now(),
	}

	// Update detected games (preserve custom games)
	oldCustomGames := config.CustomGames
	config.DetectedGames = games

	// Merge with existing custom games or convert legacy games
	if config.CustomGames == nil && len(config.Games) > 0 {
		// Convert legacy games to custom games
		config.CustomGames = cu.convertLegacyGames(config.Games)
		if verbose {
			fmt.Printf("🔄 Converted %d legacy games to custom games\n", len(config.CustomGames))
		}
	} else {
		config.CustomGames = oldCustomGames
	}

	// Save updated config atomically
	if err := cu.saveConfigAtomic(config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// UpdateGamesSection updates only the games section while preserving other settings
func (cu *ConfigUpdater) UpdateGamesSection(games []Game) error {
	config, err := cu.loadExistingConfig()
	if err != nil {
		return fmt.Errorf("failed to load existing config: %w", err)
	}

	// Merge with existing games
	mergedGames := cu.MergeGameLists(config.DetectedGames, games)
	config.DetectedGames = mergedGames

	// Update last scan time
	if config.Steam != nil {
		config.Steam.LastScan = time.Now()
	}

	// Save updated config atomically
	if err := cu.saveConfigAtomic(config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// MergeGameLists intelligently merges existing and newly scanned games
func (cu *ConfigUpdater) MergeGameLists(existing, scanned []Game) []Game {
	// Create a map of scanned games by AppID for quick lookup
	scannedMap := make(map[string]Game)
	for _, game := range scanned {
		scannedMap[game.AppID] = game
	}

	// Start with scanned games as the base
	merged := make([]Game, 0, len(scanned))
	merged = append(merged, scanned...)

	// Add existing games that weren't found in the scan (might be from different libraries)
	existingMap := make(map[string]bool)
	for _, game := range scanned {
		existingMap[game.AppID] = true
	}

	for _, existingGame := range existing {
		if !existingMap[existingGame.AppID] {
			// This game wasn't found in the scan, check if it still exists
			if cu.verifyGameStillExists(existingGame) {
				merged = append(merged, existingGame)
			} else if verbose {
				fmt.Printf("🗑️  Removing uninstalled game: %s\n", existingGame.Name)
			}
		}
	}

	return merged
}

// verifyGameStillExists checks if a game directory still exists
func (cu *ConfigUpdater) verifyGameStillExists(game Game) bool {
	if game.InstallPath == "" {
		return false
	}

	stat, err := os.Stat(game.InstallPath)
	return err == nil && stat.IsDir()
}

// convertLegacyGames converts legacy game list to custom games
func (cu *ConfigUpdater) convertLegacyGames(legacyGames []string) []CustomGame {
	customGames := make([]CustomGame, 0, len(legacyGames))

	for _, executable := range legacyGames {
		if executable == "" {
			continue
		}

		// Generate a name from the executable
		name := strings.TrimSuffix(executable, ".exe")
		name = strings.ReplaceAll(name, "_", " ")
		name = strings.ReplaceAll(name, "-", " ")
		name = strings.Title(strings.ToLower(name))

		customGame := CustomGame{
			Name:       name,
			Executable: executable,
			Path:       "", // Path will be discovered by process monitoring
		}

		customGames = append(customGames, customGame)
	}

	return customGames
}

// loadExistingConfig loads the current config or creates a default one
func (cu *ConfigUpdater) loadExistingConfig() (*Config, error) {
	config, err := LoadConfig(cu.configPath)
	if err != nil {
		// If config doesn't exist, create a default one
		config = &Config{
			DefaultPollingRate: 1000,
			GamePollingRate:    2000,
			CheckInterval:      2 * time.Second,
			Games:              []string{}, // Start with empty legacy games
		}
	}

	return config, nil
}

// saveConfigAtomic saves the config file atomically using a temporary file
func (cu *ConfigUpdater) saveConfigAtomic(config *Config) error {
	// Marshal the config to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Create temporary file
	dir := filepath.Dir(cu.configPath)
	tempFile, err := os.CreateTemp(dir, ".config-*.yaml.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}

	tempPath := tempFile.Name()

	// Clean up temp file if something goes wrong
	defer func() {
		if tempFile != nil {
			tempFile.Close()
		}
		if _, err := os.Stat(tempPath); err == nil {
			os.Remove(tempPath)
		}
	}()

	// Write data to temp file
	if _, err := tempFile.Write(data); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Close temp file
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}
	tempFile = nil // Mark as closed

	// Atomic rename
	if err := os.Rename(tempPath, cu.configPath); err != nil {
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	if verbose {
		fmt.Printf("💾 Config saved to %s\n", cu.configPath)
	}

	return nil
}

// AddCustomGame adds a custom game to the config
func (cu *ConfigUpdater) AddCustomGame(name, executable, path string) error {
	config, err := cu.loadExistingConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize custom games if nil
	if config.CustomGames == nil {
		config.CustomGames = []CustomGame{}
	}

	// Check if game already exists
	for _, game := range config.CustomGames {
		if strings.EqualFold(game.Executable, executable) {
			return fmt.Errorf("game with executable '%s' already exists", executable)
		}
	}

	// Add new custom game
	newGame := CustomGame{
		Name:       name,
		Executable: executable,
		Path:       path,
	}

	config.CustomGames = append(config.CustomGames, newGame)

	// Save config
	return cu.saveConfigAtomic(config)
}

// RemoveCustomGame removes a custom game from the config
func (cu *ConfigUpdater) RemoveCustomGame(name string) error {
	config, err := cu.loadExistingConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if config.CustomGames == nil {
		return fmt.Errorf("no custom games found")
	}

	// Find and remove the game
	found := false
	for i, game := range config.CustomGames {
		if strings.EqualFold(game.Name, name) {
			// Remove the game at index i
			config.CustomGames = append(config.CustomGames[:i], config.CustomGames[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("custom game '%s' not found", name)
	}

	// Save config
	return cu.saveConfigAtomic(config)
}

// GetGameCounts returns counts of different game types
func (cu *ConfigUpdater) GetGameCounts() (detected int, custom int, legacy int, err error) {
	config, err := cu.loadExistingConfig()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to load config: %w", err)
	}

	return len(config.DetectedGames), len(config.CustomGames), len(config.Games), nil
}
