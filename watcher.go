package main

import (
	"encoding/csv"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

type GameWatcher struct {
	config        *Config
	mouse         MouseControllerInterface
	isGameRunning bool
	ticker        *time.Ticker
	stopCh        chan struct{}
}

func NewGameWatcher(config *Config, mouse MouseControllerInterface) *GameWatcher {
	return &GameWatcher{
		config: config,
		mouse:  mouse,
		stopCh: make(chan struct{}),
	}
}

func (gw *GameWatcher) Start() {
	gw.ticker = time.NewTicker(gw.config.CheckInterval)

	go func() {
		for {
			select {
			case <-gw.ticker.C:
				gw.checkProcesses()
			case <-gw.stopCh:
				return
			}
		}
	}()

	// Initial check
	gw.checkProcesses()
}

func (gw *GameWatcher) Stop() {
	if gw.ticker != nil {
		gw.ticker.Stop()
	}
	close(gw.stopCh)
}

func (gw *GameWatcher) checkProcesses() {
	runningProcesses, err := gw.getRunningProcesses()
	if err != nil {
		if verbose {
			fmt.Printf("❌ Error getting processes: %v\n", err)
		}
		return
	}

	gameRunning := gw.isAnyGameRunning(runningProcesses)

	if gameRunning && !gw.isGameRunning {
		fmt.Printf("🎮 Game detected! Switching to %dHz\n", gw.config.GamePollingRate)
		gw.isGameRunning = true
		if err := gw.mouse.SetPollingRate(gw.config.GamePollingRate); err != nil {
			fmt.Printf("❌ Failed to set game polling rate: %v\n", err)
		}
	} else if !gameRunning && gw.isGameRunning {
		fmt.Printf("🏠 No game detected. Switching to %dHz\n", gw.config.DefaultPollingRate)
		gw.isGameRunning = false
		if err := gw.mouse.SetPollingRate(gw.config.DefaultPollingRate); err != nil {
			fmt.Printf("❌ Failed to set default polling rate: %v\n", err)
		}
	}
}

func (gw *GameWatcher) getRunningProcesses() ([]string, error) {
	cmd := exec.Command("tasklist", "/fo", "csv", "/nh")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute tasklist: %w", err)
	}

	// Parse CSV output
	reader := csv.NewReader(strings.NewReader(string(output)))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse tasklist output: %w", err)
	}

	processes := make([]string, 0, len(records))
	for _, record := range records {
		if len(record) > 0 {
			processes = append(processes, record[0])
		}
	}

	return processes, nil
}

func (gw *GameWatcher) isAnyGameRunning(processes []string) bool {
	processSet := make(map[string]bool)
	for _, process := range processes {
		processSet[strings.ToLower(process)] = true
	}

	for _, game := range gw.config.Games {
		if processSet[strings.ToLower(game)] {
			if verbose {
				fmt.Printf("🎯 Detected game: %s\n", game)
			}
			return true
		}
	}

	return false
}

func (gw *GameWatcher) GetStatus() (bool, int) {
	if gw.isGameRunning {
		return true, gw.config.GamePollingRate
	}
	return false, gw.config.DefaultPollingRate
}
