package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// SteamDetector handles Steam installation detection
type SteamDetector struct {
	config *Config
}

// NewSteamDetector creates a new Steam detector instance
func NewSteamDetector(config *Config) *SteamDetector {
	return &SteamDetector{
		config: config,
	}
}

// FindSteamInstallation finds the Steam installation path
func (sd *SteamDetector) FindSteamInstallation() (string, error) {
	// Priority order as specified in CLAUDE.md:
	// 1. Check config.yaml for saved path
	// 2. Windows Registry: HKEY_CURRENT_USER\Software\Valve\Steam
	// 3. Default locations: C:\Program Files (x86)\Steam
	// 4. Environment variable: %STEAM_PATH%

	// 1. Check config.yaml for saved path
	if sd.config.Steam != nil && sd.config.Steam.InstallPath != "" {
		if sd.validateSteamPath(sd.config.Steam.InstallPath) {
			return sd.config.Steam.InstallPath, nil
		}
	}

	// 2. Check Windows Registry
	if regPath, err := sd.getSteamPathFromRegistry(); err == nil && regPath != "" {
		if sd.validateSteamPath(regPath) {
			return regPath, nil
		}
	}

	// 3. Check default locations
	defaultPaths := []string{
		`C:\Program Files (x86)\Steam`,
		`C:\Program Files\Steam`,
		`D:\Steam`,
		`E:\Steam`,
	}

	for _, path := range defaultPaths {
		if sd.validateSteamPath(path) {
			return path, nil
		}
	}

	// 4. Check environment variable
	if envPath := os.Getenv("STEAM_PATH"); envPath != "" {
		if sd.validateSteamPath(envPath) {
			return envPath, nil
		}
	}

	return "", fmt.Errorf("Steam installation not found")
}

// getSteamPathFromRegistry retrieves Steam path from Windows registry
func (sd *SteamDetector) getSteamPathFromRegistry() (string, error) {
	// Try HKEY_CURRENT_USER first
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Valve\Steam`, registry.QUERY_VALUE)
	if err == nil {
		defer key.Close()

		steamPath, _, err := key.GetStringValue("SteamPath")
		if err == nil && steamPath != "" {
			// Convert forward slashes to backslashes for Windows
			steamPath = strings.ReplaceAll(steamPath, "/", "\\")
			return steamPath, nil
		}

		// Also try InstallPath
		installPath, _, err := key.GetStringValue("InstallPath")
		if err == nil && installPath != "" {
			installPath = strings.ReplaceAll(installPath, "/", "\\")
			return installPath, nil
		}
	}

	// Try HKEY_LOCAL_MACHINE as fallback
	key, err = registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Valve\Steam`, registry.QUERY_VALUE)
	if err == nil {
		defer key.Close()

		installPath, _, err := key.GetStringValue("InstallPath")
		if err == nil && installPath != "" {
			installPath = strings.ReplaceAll(installPath, "/", "\\")
			return installPath, nil
		}
	}

	return "", fmt.Errorf("Steam path not found in registry")
}

// validateSteamPath checks if the given path is a valid Steam installation
func (sd *SteamDetector) validateSteamPath(path string) bool {
	if path == "" {
		return false
	}

	// Check if steam.exe exists
	steamExe := filepath.Join(path, "steam.exe")
	if _, err := os.Stat(steamExe); err != nil {
		return false
	}

	// Check if steamapps folder exists
	steamApps := filepath.Join(path, "steamapps")
	if stat, err := os.Stat(steamApps); err != nil || !stat.IsDir() {
		return false
	}

	return true
}

// DiscoverLibraries finds all Steam libraries including external drives
func (sd *SteamDetector) DiscoverLibraries(steamPath string) ([]Library, error) {
	libraries := []Library{}

	// Main Steam library
	mainLibrary := Library{
		Path:  steamPath,
		Label: "Main",
	}
	libraries = append(libraries, mainLibrary)

	// Parse libraryfolders.vdf for additional libraries
	libraryFoldersPath := filepath.Join(steamPath, "steamapps", "libraryfolders.vdf")


	content, err := os.ReadFile(libraryFoldersPath)
	if err != nil {
		// If libraryfolders.vdf doesn't exist, just return main library
		if verbose {
			fmt.Printf("⚠️ Could not read libraryfolders.vdf: %v\n", err)
		}
		return libraries, nil
	}


	parser := NewVDFParser()
	libraryData, err := parser.ParseLibraryFolders(content)
	if err != nil {
		if verbose {
			fmt.Printf("⚠️ Error parsing libraryfolders.vdf: %v\n", err)
		}
		return libraries, nil
	}


	// Process discovered libraries
	for _, libInfo := range libraryData {
		if libInfo.Path == "" {
			continue
		}

		// Skip the main library (already added) by normalizing paths for comparison
		normalizedLibPath := strings.ToLower(filepath.Clean(libInfo.Path))
		normalizedSteamPath := strings.ToLower(filepath.Clean(steamPath))
		if normalizedLibPath == normalizedSteamPath {
			continue
		}

		// Validate library path exists and is accessible
		if !sd.validateLibraryPath(libInfo.Path) {
			if verbose {
				fmt.Printf("⚠️ Skipping inaccessible library: %s\n", libInfo.Path)
			}
			continue
		}

		label := libInfo.Label
		if label == "" {
			// Generate a label based on drive letter or path
			if len(libInfo.Path) >= 2 && libInfo.Path[1] == ':' {
				label = fmt.Sprintf("Drive %c", libInfo.Path[0])
			} else {
				label = "External"
			}
		}

		library := Library{
			Path:  libInfo.Path,
			Label: label,
		}
		libraries = append(libraries, library)
	}

	if verbose {
		fmt.Printf("📚 Found %d Steam libraries\n", len(libraries))
		for _, lib := range libraries {
			fmt.Printf("   - %s: %s\n", lib.Label, lib.Path)
		}
	}

	return libraries, nil
}

// validateLibraryPath checks if a library path is valid and accessible
func (sd *SteamDetector) validateLibraryPath(path string) bool {
	if path == "" {
		return false
	}

	// Check if path exists and is a directory
	stat, err := os.Stat(path)
	if err != nil || !stat.IsDir() {
		return false
	}

	// Check if steamapps folder exists
	steamApps := filepath.Join(path, "steamapps")
	if stat, err := os.Stat(steamApps); err != nil || !stat.IsDir() {
		return false
	}

	// Check if we can read the directory
	_, err = os.ReadDir(steamApps)
	return err == nil
}
