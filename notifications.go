package main

import (
	"fmt"
	"github.com/go-toast/toast"
	"path/filepath"
)

// NotificationManager handles Windows toast notifications
type NotificationManager struct {
	appID string
	iconPath string
}

// NewNotificationManager creates a new notification manager
func NewNotificationManager() *NotificationManager {
	// Get the executable path for icon
	execPath, err := filepath.Abs(".")
	iconPath := ""
	if err == nil {
		iconPath = filepath.Join(execPath, "icon.png")
	}

	return &NotificationManager{
		appID: "LAMZU.MouseAutomator",
		iconPath: iconPath,
	}
}

// ShowAppStarted shows notification when app starts
func (nm *NotificationManager) ShowAppStarted() {
	notification := toast.Notification{
		AppID:   nm.appID,
		Title:   "LAMZU Automator",
		Message: "🚀 App iniciado com sucesso! Monitorando jogos...",
		Icon:    nm.iconPath,
	}

	if err := notification.Push(); err != nil && verbose {
		fmt.Printf("⚠️ Failed to show notification: %v\n", err)
	}
}

// ShowGameDetected shows notification when a game is detected
func (nm *NotificationManager) ShowGameDetected(pollingRate int) {
	notification := toast.Notification{
		AppID:   nm.appID,
		Title:   "Jogo Detectado!",
		Message: fmt.Sprintf("🎮 Alterando polling rate para %dHz", pollingRate),
		Icon:    nm.iconPath,
	}

	if err := notification.Push(); err != nil && verbose {
		fmt.Printf("⚠️ Failed to show notification: %v\n", err)
	}
}

// ShowGameClosed shows notification when no game is running
func (nm *NotificationManager) ShowGameClosed(pollingRate int) {
	notification := toast.Notification{
		AppID:   nm.appID,
		Title:   "Jogo Fechado",
		Message: fmt.Sprintf("🏠 Aplicando polling rate padrão: %dHz", pollingRate),
		Icon:    nm.iconPath,
	}

	if err := notification.Push(); err != nil && verbose {
		fmt.Printf("⚠️ Failed to show notification: %v\n", err)
	}
}

// ShowError shows error notification
func (nm *NotificationManager) ShowError(title, message string) {
	notification := toast.Notification{
		AppID:   nm.appID,
		Title:   title,
		Message: fmt.Sprintf("❌ %s", message),
		Icon:    nm.iconPath,
	}

	if err := notification.Push(); err != nil && verbose {
		fmt.Printf("⚠️ Failed to show notification: %v\n", err)
	}
}
