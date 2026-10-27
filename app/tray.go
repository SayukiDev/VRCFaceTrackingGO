package app

import (
	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/SayukiDev/VRCFaceTrackingGO/internal/metadata"
)

// newTray adds the system tray icon. Clicking it shows the main window; its
// menu is the only way to quit once the main window has been closed.
func newTray(wails *application.App, icon []byte) {
	menu := application.NewMenu()
	menu.Add("Show").OnClick(func(*application.Context) { showMainWindow(wails) })
	menu.AddSeparator()
	menu.Add("Quit").OnClick(func(*application.Context) { wails.Quit() })

	tray := wails.SystemTray.New().
		SetIcon(icon).
		SetMenu(menu).
		OnClick(func() { showMainWindow(wails) })
	tray.SetTooltip(metadata.Info.Name)
}
