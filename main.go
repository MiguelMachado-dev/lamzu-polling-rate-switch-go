package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var (
	configFile   string
	daemon       bool
	verbose      bool
	dryRun       bool
	force        bool
	gameName     string
	gameExe      string
	gamePath     string
)

var rootCmd = &cobra.Command{
	Use:   "lamzu-automator",
	Short: "LAMZU Mouse Polling Rate Auto-Switch",
	Long:  "Automatically adjusts LAMZU mouse polling rate based on running applications",
	Run:   runAutomator,
}

var setCmd = &cobra.Command{
	Use:   "set [rate]",
	Short: "Set polling rate manually",
	Args:  cobra.ExactArgs(1),
	Run:   runSetRate,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available polling rates",
	Run:   runListRates,
}

var debugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Debug and list LAMZU devices",
	Run:   runDebug,
}

var scanSteamCmd = &cobra.Command{
	Use:   "scan-steam",
	Short: "Scan for Steam games and update config",
	Run:   runScanSteam,
}

var addGameCmd = &cobra.Command{
	Use:   "add-game",
	Short: "Add a custom game manually",
	Run:   runAddGame,
}

var removeGameCmd = &cobra.Command{
	Use:   "remove-game",
	Short: "Remove a custom game",
	Run:   runRemoveGame,
}

var listGamesCmd = &cobra.Command{
	Use:   "list-games",
	Short: "List all configured games",
	Run:   runListGames,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "config.yaml", "config file")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.Flags().BoolVarP(&daemon, "daemon", "d", false, "run as daemon")

	// Steam scan command flags
	scanSteamCmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be updated without saving")
	scanSteamCmd.Flags().BoolVar(&force, "force", false, "force rescan even if recent scan exists")

	// Add game command flags
	addGameCmd.Flags().StringVar(&gameName, "name", "", "game name (required)")
	addGameCmd.Flags().StringVar(&gameExe, "exe", "", "game executable (required)")
	addGameCmd.Flags().StringVar(&gamePath, "path", "", "game path (optional)")
	addGameCmd.MarkFlagRequired("name")
	addGameCmd.MarkFlagRequired("exe")

	// Remove game command flags
	removeGameCmd.Flags().StringVar(&gameName, "name", "", "game name to remove (required)")
	removeGameCmd.MarkFlagRequired("name")

	rootCmd.AddCommand(setCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(debugCmd)
	rootCmd.AddCommand(scanSteamCmd)
	rootCmd.AddCommand(addGameCmd)
	rootCmd.AddCommand(removeGameCmd)
	rootCmd.AddCommand(listGamesCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func initMouseController() (MouseControllerInterface, error) {
	// Use Windows native HID API
	controller, err := NewWindowsMouseController()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Windows HID controller: %w", err)
	}

	if verbose {
		fmt.Println("✅ Using Windows native HID API")
	}
	return controller, nil
}

func runAutomator(cmd *cobra.Command, args []string) {
	config, err := LoadConfig(configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	mouse, err := initMouseController()
	if err != nil {
		log.Fatalf("Failed to initialize mouse controller: %v", err)
	}
	defer mouse.Close()

	// Test mouse connection
	if err := mouse.TestConnection(); err != nil {
		log.Fatalf("Failed to connect to LAMZU mouse: %v", err)
	}

	fmt.Println("🎮 LAMZU Polling Rate Auto-Switch v1.0")
	fmt.Println("✅ Mouse connected successfully")

	// Initialize notification manager
	notificationManager := NewNotificationManager()

	// Set initial polling rate
	if err := mouse.SetPollingRate(config.DefaultPollingRate); err != nil {
		log.Fatalf("Failed to set initial polling rate: %v", err)
	}

	fmt.Printf("📊 Default polling rate: %dHz\n", config.DefaultPollingRate)
	fmt.Printf("🎯 Game polling rate: %dHz\n", config.GamePollingRate)
	
	totalGames := len(config.Games) + len(config.DetectedGames) + len(config.CustomGames)
	fmt.Printf("🔍 Monitoring %d games\n", totalGames)

	// Show app started notification
	notificationManager.ShowAppStarted()

	watcher := NewGameWatcher(config, mouse, notificationManager)

	if daemon {
		fmt.Println("🚀 Starting in daemon mode...")
		runDaemon(watcher)
	} else {
		fmt.Println("🎮 Starting in interactive mode (Ctrl+C to stop)...")
		runInteractive(watcher)
	}
}

func runSetRate(cmd *cobra.Command, args []string) {
	rate := parsePollingRate(args[0])
	if rate == 0 {
		fmt.Fprintf(os.Stderr, "Invalid polling rate: %s\n", args[0])
		fmt.Fprintf(os.Stderr, "Valid rates: 500, 1000, 2000, 4000, 8000\n")
		os.Exit(1)
	}

	mouse, err := initMouseController()
	if err != nil {
		log.Fatalf("Failed to initialize mouse controller: %v", err)
	}
	defer mouse.Close()

	if err := mouse.SetPollingRate(rate); err != nil {
		log.Fatalf("Failed to set polling rate: %v", err)
	}

	fmt.Printf("✅ Polling rate set to %dHz\n", rate)
}

func runListRates(cmd *cobra.Command, args []string) {
	fmt.Println("Available polling rates:")
	for rate := range pollingRateMap {
		fmt.Printf("  %dHz\n", rate)
	}
}

func runDebug(cmd *cobra.Command, args []string) {
	fmt.Println("🔧 LAMZU Device Debug Mode")
	fmt.Println("==========================")

	// Try to connect
	fmt.Println("🔌 Testing connection...")
	mouse, err := initMouseController()
	if err != nil {
		fmt.Printf("❌ Failed to connect: %v\n", err)
		return
	}
	defer mouse.Close()

	// Test connection
	if err := mouse.TestConnection(); err != nil {
		fmt.Printf("❌ Connection test failed: %v\n", err)
		return
	}

	fmt.Println("✅ Connection successful!")

	// Test setting polling rates
	fmt.Println("\n🎯 Testing polling rate changes...")
	testRates := []int{1000, 2000, 1000}

	for _, rate := range testRates {
		fmt.Printf("Setting %dHz... ", rate)
		if err := mouse.SetPollingRate(rate); err != nil {
			fmt.Printf("❌ Failed: %v\n", err)
		} else {
			fmt.Printf("✅ Success!\n")
		}
	}

	fmt.Println("\n🎉 All tests completed!")
}

func runInteractive(watcher *GameWatcher) {
	watcher.Start()
	defer watcher.Stop()

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Println("\n👋 Shutting down...")
}

func runDaemon(watcher *GameWatcher) {
	// In a real implementation, you'd use Windows service APIs
	// For now, this runs as a background process
	watcher.Start()
	defer watcher.Stop()

	// Keep running until system signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
}

// Steam scanning command implementations
func runScanSteam(cmd *cobra.Command, args []string) {
	fmt.Println("🔍 Scanning for Steam games...")

	// Initialize Steam detector
	config, err := LoadConfig(configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	detector := NewSteamDetector(config)

	// Find Steam installation
	steamPath, err := detector.FindSteamInstallation()
	if err != nil {
		fmt.Printf("❌ Steam installation not found: %v\n", err)
		fmt.Println("💡 Make sure Steam is installed or use --config to specify a custom config file")
		os.Exit(1)
	}

	if verbose {
		fmt.Printf("✅ Steam found at: %s\n", steamPath)
	}

	// Discover libraries
	libraries, err := detector.DiscoverLibraries(steamPath)
	if err != nil {
		log.Fatalf("Failed to discover Steam libraries: %v", err)
	}

	// Check if we should skip scan due to recent scan
	if !force && config.Steam != nil {
		timeSinceLastScan := time.Since(config.Steam.LastScan)
		if timeSinceLastScan < 24*time.Hour {
			fmt.Printf("⏰ Recent scan found (%.1f hours ago)\n", timeSinceLastScan.Hours())
			fmt.Println("Use --force to rescan anyway")
			return
		}
	}

	// Scan for games
	scanner := NewGameScanner(libraries)
	games, err := scanner.ScanAllLibraries()
	if err != nil && verbose {
		fmt.Printf("⚠️ Scan completed with warnings: %v\n", err)
	}

	fmt.Printf("🎮 Found %d games across %d libraries\n", len(games), len(libraries))

	if dryRun {
		fmt.Println("\n📋 Dry run - no changes saved:")
		fmt.Println("Steam libraries:")
		for _, lib := range libraries {
			fmt.Printf("  - %s: %s\n", lib.Label, lib.Path)
		}
		fmt.Println("\nDetected games:")
		for _, game := range games {
			sizeMB := game.SizeMB
			if sizeMB == 0 {
				fmt.Printf("  - %s (%s)\n", game.Name, game.Executable)
			} else {
				fmt.Printf("  - %s (%s, %.1f GB)\n", game.Name, game.Executable, float64(sizeMB)/1024)
			}
		}
		return
	}

	// Update config
	updater := NewConfigUpdater(configFile)
	if err := updater.UpdateWithSteamData(steamPath, libraries, games); err != nil {
		log.Fatalf("Failed to update config: %v", err)
	}

	fmt.Printf("✅ Config updated with %d games\n", len(games))
	
	// Display summary
	detected, custom, legacy, err := updater.GetGameCounts()
	if err == nil {
		fmt.Printf("📊 Games: %d detected, %d custom", detected, custom)
		if legacy > 0 {
			fmt.Printf(", %d legacy", legacy)
		}
		fmt.Println()
	}
}

func runAddGame(cmd *cobra.Command, args []string) {
	updater := NewConfigUpdater(configFile)
	
	if err := updater.AddCustomGame(gameName, gameExe, gamePath); err != nil {
		fmt.Printf("❌ Failed to add game: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("✅ Added custom game: %s (%s)\n", gameName, gameExe)
}

func runRemoveGame(cmd *cobra.Command, args []string) {
	updater := NewConfigUpdater(configFile)
	
	if err := updater.RemoveCustomGame(gameName); err != nil {
		fmt.Printf("❌ Failed to remove game: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("✅ Removed custom game: %s\n", gameName)
}

func runListGames(cmd *cobra.Command, args []string) {
	config, err := LoadConfig(configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	fmt.Println("🎮 Configured Games:")
	
	// Show detected Steam games
	if len(config.DetectedGames) > 0 {
		fmt.Println("\n📚 Steam Games:")
		for _, game := range config.DetectedGames {
			if game.SizeMB > 0 {
				fmt.Printf("  - %s (%s, %.1f GB)\n", game.Name, game.Executable, float64(game.SizeMB)/1024)
			} else {
				fmt.Printf("  - %s (%s)\n", game.Name, game.Executable)
			}
		}
	}
	
	// Show custom games
	if len(config.CustomGames) > 0 {
		fmt.Println("\n🛠️ Custom Games:")
		for _, game := range config.CustomGames {
			if game.Path != "" {
				fmt.Printf("  - %s (%s) [%s]\n", game.Name, game.Executable, game.Path)
			} else {
				fmt.Printf("  - %s (%s)\n", game.Name, game.Executable)
			}
		}
	}
	
	// Show legacy games
	if len(config.Games) > 0 {
		fmt.Println("\n⚠️ Legacy Games (consider migrating):")
		for _, game := range config.Games {
			fmt.Printf("  - %s\n", game)
		}
	}

	// Summary
	total := len(config.DetectedGames) + len(config.CustomGames) + len(config.Games)
	fmt.Printf("\n📊 Total: %d games configured\n", total)
	
	if config.Steam != nil && !config.Steam.LastScan.IsZero() {
		fmt.Printf("🕐 Last Steam scan: %s\n", config.Steam.LastScan.Format("2006-01-02 15:04:05"))
	}
}
