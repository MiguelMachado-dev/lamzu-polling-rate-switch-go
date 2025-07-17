package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	configFile string
	daemon     bool
	verbose    bool
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

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "config.yaml", "config file")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.Flags().BoolVarP(&daemon, "daemon", "d", false, "run as daemon")

	rootCmd.AddCommand(setCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(debugCmd)
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
	fmt.Printf("🔍 Monitoring %d games\n", len(config.Games))

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
