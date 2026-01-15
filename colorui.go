// colorui.go
package main

import (
	"fmt"

	"github.com/gotk3/gotk3/gtk"
	log "github.com/sirupsen/logrus"
)

var colorSyncManager *ColorSyncManager

// initColorSync initializes the color sync manager
func initColorSync() {
	colorSyncManager = NewColorSyncManager()
	log.Debug("Color sync manager initialized")
}

// onThemeChanged is called when the GTK theme changes
func onThemeChanged(themeName string) {
	if colorSyncManager == nil {
		return
	}

	if !colorSyncManager.IsEnabled() || !colorSyncManager.IsAutoApply() {
		log.Debug("Color sync auto-apply is disabled")
		return
	}

	go func() {
		if err := colorSyncManager.ApplyTheme(themeName); err != nil {
			log.Warnf("Failed to apply theme colors: %v", err)
		}
	}()
}

// setUpColorSyncForm creates the color sync settings UI
func setUpColorSyncForm() *gtk.Frame {
	frame, _ := gtk.FrameNew(fmt.Sprintf("  %s  ", "Color Synchronization"))
	frame.SetLabelAlign(0.5, 0.5)
	frame.SetProperty("margin", 6)

	mainBox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 12)
	mainBox.SetProperty("margin", 12)
	mainBox.SetProperty("hexpand", true)
	mainBox.SetProperty("vexpand", true)
	frame.Add(mainBox)

	// Description
	desc, _ := gtk.LabelNew("Automatically extract colors from GTK theme and apply to other applications")
	desc.SetLineWrap(true)
	desc.SetProperty("halign", gtk.ALIGN_START)
	desc.SetProperty("margin-bottom", 6)
	mainBox.PackStart(desc, false, false, 0)

	// Enable/Disable
	enableBox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 12)
	enableLabel, _ := gtk.LabelNew("Enable color synchronization:")
	enableLabel.SetProperty("halign", gtk.ALIGN_START)
	enableBox.PackStart(enableLabel, false, false, 0)

	enableSwitch, _ := gtk.SwitchNew()
	enableSwitch.SetActive(colorSyncManager.IsEnabled())
	enableSwitch.Connect("state-set", func(s *gtk.Switch, state bool) {
		colorSyncManager.SetEnabled(state)
		log.Infof("Color sync enabled: %v", state)
	})
	enableBox.PackStart(enableSwitch, false, false, 0)
	mainBox.PackStart(enableBox, false, false, 0)

	// Auto-apply
	autoBox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 12)
	autoLabel, _ := gtk.LabelNew("Auto-apply on theme change:")
	autoLabel.SetProperty("halign", gtk.ALIGN_START)
	autoBox.PackStart(autoLabel, false, false, 0)

	autoSwitch, _ := gtk.SwitchNew()
	autoSwitch.SetActive(colorSyncManager.IsAutoApply())
	autoSwitch.Connect("state-set", func(s *gtk.Switch, state bool) {
		colorSyncManager.SetAutoApply(state)
		log.Infof("Color sync auto-apply: %v", state)
	})
	autoBox.PackStart(autoSwitch, false, false, 0)
	mainBox.PackStart(autoBox, false, false, 0)

	// Applications frame
	appsFrame, _ := gtk.FrameNew("Applications")
	appsFrame.SetProperty("margin-top", 12)
	mainBox.PackStart(appsFrame, false, false, 0)

	appsGrid, _ := gtk.GridNew()
	appsGrid.SetRowSpacing(6)
	appsGrid.SetColumnSpacing(12)
	appsGrid.SetProperty("margin", 12)
	appsFrame.Add(appsGrid)

	// Application checkboxes
	apps := colorSyncManager.GetApplications()
	row := 0
	col := 0
	for _, app := range apps {
		appName := app
		cb, _ := gtk.CheckButtonNewWithLabel(capitalizeFirst(appName))
		cb.SetActive(colorSyncManager.IsAppEnabled(appName))
		cb.Connect("toggled", func() {
			enabled := cb.GetActive()
			colorSyncManager.SetAppEnabled(appName, enabled)
			log.Debugf("App %s sync: %v", appName, enabled)
		})
		appsGrid.Attach(cb, col, row, 1, 1)

		col++
		if col > 2 {
			col = 0
			row++
		}
	}

	// Manual apply button
	btnBox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 12)
	btnBox.SetProperty("margin-top", 12)
	
	applyBtn, _ := gtk.ButtonNew()
	applyBtn.SetLabel("Apply Colors Now")
	applyBtn.SetProperty("hexpand", true)
	
	statusLabel, _ := gtk.LabelNew("")
	statusLabel.SetProperty("halign", gtk.ALIGN_START)
	statusLabel.SetLineWrap(true)
	
	applyBtn.Connect("clicked", func() {
		themeName := gsettings.gtkTheme
		if themeName == "" {
			statusLabel.SetMarkup("<span foreground='red'>No theme selected</span>")
			return
		}
		
		statusLabel.SetMarkup(fmt.Sprintf("Applying colors from <b>%s</b>...", themeName))
		
		go func() {
			err := colorSyncManager.ApplyTheme(themeName)
			if err != nil {
				statusLabel.SetMarkup(fmt.Sprintf("<span foreground='red'>✗ Error: %s</span>", err.Error()))
			} else {
				statusLabel.SetMarkup("<span foreground='green'>✓ Colors applied successfully!</span>")
			}
		}()
	})
	
	btnBox.PackStart(applyBtn, true, true, 0)
	mainBox.PackStart(btnBox, false, false, 0)
	mainBox.PackStart(statusLabel, false, false, 6)

	// Current scheme info
	if colorSyncManager.config.LastTheme != "" {
		infoBox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 6)
		infoBox.SetProperty("margin-top", 12)
		
		sep, _ := gtk.SeparatorNew(gtk.ORIENTATION_HORIZONTAL)
		infoBox.PackStart(sep, false, false, 6)
		
		infoLabel, _ := gtk.LabelNew("")
		infoLabel.SetMarkup(fmt.Sprintf("<small>Last applied: <b>%s</b></small>", 
			colorSyncManager.config.LastTheme))
		infoLabel.SetProperty("halign", gtk.ALIGN_START)
		infoBox.PackStart(infoLabel, false, false, 0)
		
		if colorSyncManager.config.LastColors != nil {
			palette := colorSyncManager.config.LastColors
			colorBox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 6)
			colorBox.SetProperty("margin-top", 6)
			
			// Show a few sample colors
			samples := []struct{label, color string}{
				{"BG", palette.Background},
				{"FG", palette.Foreground},
				{"R", palette.Colors["color1"]},
				{"G", palette.Colors["color2"]},
				{"B", palette.Colors["color4"]},
			}
			
			for _, s := range samples {
				box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 2)
				
				lbl, _ := gtk.LabelNew(s.label)
				lbl.SetMarkup(fmt.Sprintf("<small>%s</small>", s.label))
				box.PackStart(lbl, false, false, 0)
				
				da, _ := gtk.DrawingAreaNew()
				da.SetSizeRequest(40, 20)
				da.Connect("draw", func(da *gtk.DrawingArea, cr *gtk.cairo.Context) {
					// Parse hex color
					r, g, b := parseHexColor(s.color)
					cr.SetSourceRGB(r, g, b)
					cr.Rectangle(0, 0, 40, 20)
					cr.Fill()
				})
				box.PackStart(da, false, false, 0)
				
				colorBox.PackStart(box, false, false, 6)
			}
			
			infoBox.PackStart(colorBox, false, false, 0)
		}
		
		mainBox.PackStart(infoBox, false, false, 0)
	}

	// Help text
	helpLabel, _ := gtk.LabelNew("")
	helpLabel.SetMarkup(`<small><i>Tip: Include generated config files in your application configs:
• Alacritty: import: - ~/.config/alacritty/colors.yml
• Kitty: include ./theme.conf
• Waybar: @import "colors.css"
• Rofi: @import "colors.rasi"</i></small>`)
	helpLabel.SetLineWrap(true)
	helpLabel.SetProperty("halign", gtk.ALIGN_START)
	helpLabel.SetProperty("margin-top", 12)
	mainBox.PackStart(helpLabel, false, false, 0)

	return frame
}

// parseHexColor converts hex color to RGB values (0.0-1.0)
func parseHexColor(hex string) (float64, float64, float64) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 0, 0, 0
	}
	
	var r, g, b int
	fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	
	return float64(r) / 255.0, float64(g) / 255.0, float64(b) / 255.0
}

// capitalizeFirst capitalizes the first letter of a string
func capitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// displayColorSyncForm shows the color sync settings
func displayColorSyncForm() {
	destroyContent()

	preview = setUpColorSyncForm()
	grid.Attach(preview, 0, 1, 2, 1)
	menuBar.Deactivate()
	grid.ShowAll()
	scrolledWindow.Hide()
}
