// colorextractor.go
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

// ColorPalette represents a standardized color scheme
type ColorPalette struct {
	Background string            `json:"background"`
	Foreground string            `json:"foreground"`
	Cursor     string            `json:"cursor"`
	Colors     map[string]string `json:"colors"` // color0-color15
}

// ColorSyncConfig holds settings for color synchronization
type ColorSyncConfig struct {
	Enabled      bool              `json:"enabled"`
	AutoApply    bool              `json:"auto-apply"`
	Applications map[string]bool   `json:"applications"`
	LastTheme    string            `json:"last-theme"`
	LastColors   *ColorPalette     `json:"last-colors,omitempty"`
}

// ColorExtractor extracts colors from GTK themes
type ColorExtractor struct {
	themePaths []string
}

// NewColorExtractor creates a new color extractor
func NewColorExtractor() *ColorExtractor {
	paths := []string{
		filepath.Join(os.Getenv("HOME"), ".themes"),
		filepath.Join(os.Getenv("HOME"), ".local/share/themes"),
		"/usr/share/themes",
	}
	return &ColorExtractor{themePaths: paths}
}

// FindThemePath locates the GTK theme directory
func (ce *ColorExtractor) FindThemePath(themeName string) string {
	for _, basePath := range ce.themePaths {
		themePath := filepath.Join(basePath, themeName, "gtk-3.0")
		if pathExists(themePath) {
			return themePath
		}
	}
	return ""
}

// ExtractColors extracts color palette from GTK theme
func (ce *ColorExtractor) ExtractColors(themeName string) (*ColorPalette, error) {
	themePath := ce.FindThemePath(themeName)
	if themePath == "" {
		return nil, fmt.Errorf("theme %s not found", themeName)
	}

	cssFile := filepath.Join(themePath, "gtk.css")
	if !pathExists(cssFile) {
		return nil, fmt.Errorf("gtk.css not found in %s", themePath)
	}

	colors := make(map[string]string)
	
	content, err := os.ReadFile(cssFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read css file: %w", err)
	}

	// Extract @define-color declarations
	colorPattern := regexp.MustCompile(`@define-color\s+(\w+)\s+([#\w(),.\s]+);`)
	matches := colorPattern.FindAllStringSubmatch(string(content), -1)
	
	for _, match := range matches {
		if len(match) >= 3 {
			name := match[1]
			value := strings.TrimSpace(match[2])
			colors[name] = value
		}
	}

	// Extract CSS variables
	varPattern := regexp.MustCompile(`--(\w+-\w+(?:-\w+)*)\s*:\s*([#\w(),.\s]+);`)
	varMatches := varPattern.FindAllStringSubmatch(string(content), -1)
	
	for _, match := range varMatches {
		if len(match) >= 3 {
			name := match[1]
			value := strings.TrimSpace(match[2])
			colors[name] = value
		}
	}

	// Resolve color references
	colors = ce.resolveColorReferences(colors)

	// Generate standard palette
	palette := ce.generateStandardPalette(colors)

	return palette, nil
}

// resolveColorReferences resolves @color references in values
func (ce *ColorExtractor) resolveColorReferences(colors map[string]string) map[string]string {
	resolved := make(map[string]string)
	maxIterations := 10

	for i := 0; i < maxIterations; i++ {
		changed := false
		for name, value := range colors {
			if strings.Contains(value, "@") {
				refPattern := regexp.MustCompile(`@(\w+)`)
				matches := refPattern.FindStringSubmatch(value)
				if len(matches) >= 2 {
					refName := matches[1]
					if refValue, exists := colors[refName]; exists {
						newValue := strings.Replace(value, "@"+refName, refValue, -1)
						if newValue != value {
							colors[name] = newValue
							changed = true
						}
					}
				}
			}
			resolved[name] = colors[name]
		}
		if !changed {
			break
		}
	}

	return resolved
}

// generateStandardPalette creates a standardized color palette
func (ce *ColorExtractor) generateStandardPalette(colors map[string]string) *ColorPalette {
	palette := &ColorPalette{
		Background: "#1e1e1e",
		Foreground: "#d4d4d4",
		Cursor:     "#d4d4d4",
		Colors: map[string]string{
			"color0":  "#000000",
			"color1":  "#cd3131",
			"color2":  "#0dbc79",
			"color3":  "#e5e510",
			"color4":  "#2472c8",
			"color5":  "#bc3fbc",
			"color6":  "#11a8cd",
			"color7":  "#e5e5e5",
			"color8":  "#666666",
			"color9":  "#f14c4c",
			"color10": "#23d18b",
			"color11": "#f5f543",
			"color12": "#3b8eea",
			"color13": "#d670d6",
			"color14": "#29b8db",
			"color15": "#e5e5e5",
		},
	}

	// Map GTK theme colors to standard palette
	colorMapping := map[string]string{
		"theme_bg_color":          "background",
		"theme_fg_color":          "foreground",
		"theme_base_color":        "background",
		"theme_text_color":        "foreground",
		"theme_selected_bg_color": "color4",
		"warning_color":           "color3",
		"error_color":             "color1",
		"success_color":           "color2",
	}

	for gtkName, stdName := range colorMapping {
		if value, exists := colors[gtkName]; exists {
			normalized := ce.normalizeColor(value)
			if stdName == "background" || stdName == "foreground" || stdName == "cursor" {
				switch stdName {
				case "background":
					palette.Background = normalized
				case "foreground":
					palette.Foreground = normalized
				case "cursor":
					palette.Cursor = normalized
				}
			} else {
				palette.Colors[stdName] = normalized
			}
		}
	}

	return palette
}

// normalizeColor converts color to hex format
func (ce *ColorExtractor) normalizeColor(color string) string {
	color = strings.TrimSpace(color)

	// Already hex
	if strings.HasPrefix(color, "#") {
		return color
	}

	// RGB/RGBA format
	if strings.Contains(strings.ToLower(color), "rgb") {
		rgbPattern := regexp.MustCompile(`rgba?\((\d+),\s*(\d+),\s*(\d+)`)
		matches := rgbPattern.FindStringSubmatch(color)
		if len(matches) >= 4 {
			r, _ := strconv.Atoi(matches[1])
			g, _ := strconv.Atoi(matches[2])
			b, _ := strconv.Atoi(matches[3])
			return fmt.Sprintf("#%02x%02x%02x", r, g, b)
		}
	}

	return color
}

// TemplateManager manages color templates
type TemplateManager struct {
	configDir string
	templates map[string]string
}

// NewTemplateManager creates a new template manager
func NewTemplateManager() *TemplateManager {
	configDir := filepath.Join(configHome(), "nwg-look/color-templates")
	makeDir(configDir)

	tm := &TemplateManager{
		configDir: configDir,
		templates: make(map[string]string),
	}

	tm.createDefaultTemplates()
	return tm
}

// createDefaultTemplates creates default color templates
func (tm *TemplateManager) createDefaultTemplates() {
	templates := map[string]string{
		"alacritty.yml":      tm.alacrittyTemplate(),
		"waybar-colors.css":  tm.waybarTemplate(),
		"kitty.conf":         tm.kittyTemplate(),
		"rofi-colors.rasi":   tm.rofiTemplate(),
		"dunst-colors.conf":  tm.dunstTemplate(),
		"foot.ini":           tm.footTemplate(),
		"termite-colors.ini": tm.termiteTemplate(),
	}

	for filename, content := range templates {
		templateFile := filepath.Join(tm.configDir, filename)
		if !pathExists(templateFile) {
			if err := os.WriteFile(templateFile, []byte(content), 0644); err != nil {
				log.Warnf("Failed to create template %s: %v", filename, err)
			} else {
				log.Debugf("Created template: %s", templateFile)
			}
		}
	}
}

func (tm *TemplateManager) alacrittyTemplate() string {
	return `# Alacritty colors - Generated by nwg-look
colors:
  primary:
    background: '{background}'
    foreground: '{foreground}'
  cursor:
    text: '{background}'
    cursor: '{cursor}'
  normal:
    black:   '{color0}'
    red:     '{color1}'
    green:   '{color2}'
    yellow:  '{color3}'
    blue:    '{color4}'
    magenta: '{color5}'
    cyan:    '{color6}'
    white:   '{color7}'
  bright:
    black:   '{color8}'
    red:     '{color9}'
    green:   '{color10}'
    yellow:  '{color11}'
    blue:    '{color12}'
    magenta: '{color13}'
    cyan:    '{color14}'
    white:   '{color15}'
`
}

func (tm *TemplateManager) waybarTemplate() string {
	return `/* Waybar colors - Generated by nwg-look */
@define-color background {background};
@define-color foreground {foreground};
@define-color color0 {color0};
@define-color color1 {color1};
@define-color color2 {color2};
@define-color color3 {color3};
@define-color color4 {color4};
@define-color color5 {color5};
@define-color color6 {color6};
@define-color color7 {color7};
@define-color color8 {color8};

window#waybar {
    background-color: @background;
    color: @foreground;
}
`
}

func (tm *TemplateManager) kittyTemplate() string {
	return `# Kitty colors - Generated by nwg-look
foreground {foreground}
background {background}
cursor {cursor}

color0 {color0}
color1 {color1}
color2 {color2}
color3 {color3}
color4 {color4}
color5 {color5}
color6 {color6}
color7 {color7}
color8 {color8}
color9 {color9}
color10 {color10}
color11 {color11}
color12 {color12}
color13 {color13}
color14 {color14}
color15 {color15}
`
}

func (tm *TemplateManager) rofiTemplate() string {
	return `/* Rofi colors - Generated by nwg-look */
* {
    background: {background};
    foreground: {foreground};
    selected: {color4};
    active: {color2};
    urgent: {color1};
}
`
}

func (tm *TemplateManager) dunstTemplate() string {
	return `# Dunst colors - Generated by nwg-look
[global]
    background = "{background}"
    foreground = "{foreground}"
    
[urgency_low]
    background = "{background}"
    foreground = "{foreground}"

[urgency_normal]
    background = "{color4}"
    foreground = "{foreground}"

[urgency_critical]
    background = "{color1}"
    foreground = "{foreground}"
`
}

func (tm *TemplateManager) footTemplate() string {
	return `# Foot terminal colors - Generated by nwg-look
[colors]
foreground={foreground}
background={background}

regular0={color0}
regular1={color1}
regular2={color2}
regular3={color3}
regular4={color4}
regular5={color5}
regular6={color6}
regular7={color7}

bright0={color8}
bright1={color9}
bright2={color10}
bright3={color11}
bright4={color12}
bright5={color13}
bright6={color14}
bright7={color15}
`
}

func (tm *TemplateManager) termiteTemplate() string {
	return `# Termite colors - Generated by nwg-look
[colors]
foreground = {foreground}
background = {background}
cursor = {cursor}

color0 = {color0}
color1 = {color1}
color2 = {color2}
color3 = {color3}
color4 = {color4}
color5 = {color5}
color6 = {color6}
color7 = {color7}
color8 = {color8}
color9 = {color9}
color10 = {color10}
color11 = {color11}
color12 = {color12}
color13 = {color13}
color14 = {color14}
color15 = {color15}
`
}

// ApplyColors applies colors to all templates
func (tm *TemplateManager) ApplyColors(palette *ColorPalette, enabledApps map[string]bool) error {
	destinations := map[string]string{
		"alacritty.yml":      filepath.Join(configHome(), "alacritty/colors.yml"),
		"waybar-colors.css":  filepath.Join(configHome(), "waybar/colors.css"),
		"kitty.conf":         filepath.Join(configHome(), "kitty/theme.conf"),
		"rofi-colors.rasi":   filepath.Join(configHome(), "rofi/colors.rasi"),
		"dunst-colors.conf":  filepath.Join(configHome(), "dunst/dunstrc-colors"),
		"foot.ini":           filepath.Join(configHome(), "foot/colors.ini"),
		"termite-colors.ini": filepath.Join(configHome(), "termite/colors"),
	}

	appNames := map[string]string{
		"alacritty.yml":      "alacritty",
		"waybar-colors.css":  "waybar",
		"kitty.conf":         "kitty",
		"rofi-colors.rasi":   "rofi",
		"dunst-colors.conf":  "dunst",
		"foot.ini":           "foot",
		"termite-colors.ini": "termite",
	}

	for templateName, destPath := range destinations {
		appName := appNames[templateName]
		
		// Skip if app is disabled
		if enabled, exists := enabledApps[appName]; exists && !enabled {
			log.Debugf("Skipping %s (disabled)", appName)
			continue
		}

		templatePath := filepath.Join(tm.configDir, templateName)
		if !pathExists(templatePath) {
			log.Debugf("Template not found: %s", templatePath)
			continue
		}

		// Read template
		content, err := os.ReadFile(templatePath)
		if err != nil {
			log.Warnf("Failed to read template %s: %v", templateName, err)
			continue
		}

		// Apply colors
		output := tm.fillTemplate(string(content), palette)

		// Create destination directory
		destDir := filepath.Dir(destPath)
		makeDir(destDir)

		// Write to destination
		if err := os.WriteFile(destPath, []byte(output), 0644); err != nil {
			log.Warnf("Failed to write %s: %v", destPath, err)
		} else {
			log.Infof("✓ Applied colors to %s", destPath)
		}
	}

	return nil
}

// fillTemplate replaces placeholders with actual colors
func (tm *TemplateManager) fillTemplate(template string, palette *ColorPalette) string {
	output := template
	
	// Replace main colors
	output = strings.ReplaceAll(output, "{background}", palette.Background)
	output = strings.ReplaceAll(output, "{foreground}", palette.Foreground)
	output = strings.ReplaceAll(output, "{cursor}", palette.Cursor)

	// Replace color0-color15
	for name, value := range palette.Colors {
		placeholder := "{" + name + "}"
		output = strings.ReplaceAll(output, placeholder, value)
	}

	return output
}

// ColorSyncManager manages the color synchronization feature
type ColorSyncManager struct {
	extractor *ColorExtractor
	templates *TemplateManager
	config    *ColorSyncConfig
	configFile string
}

// NewColorSyncManager creates a new color sync manager
func NewColorSyncManager() *ColorSyncManager {
	configFile := filepath.Join(configHome(), "nwg-look/color-sync.json")
	
	csm := &ColorSyncManager{
		extractor:  NewColorExtractor(),
		templates:  NewTemplateManager(),
		configFile: configFile,
	}

	csm.loadConfig()
	return csm
}

// loadConfig loads the color sync configuration
func (csm *ColorSyncManager) loadConfig() {
	if pathExists(csm.configFile) {
		data, err := os.ReadFile(csm.configFile)
		if err == nil {
			var config ColorSyncConfig
			if err := json.Unmarshal(data, &config); err == nil {
				csm.config = &config
				log.Debug("Loaded color sync config")
				return
			}
		}
	}

	// Default configuration
	csm.config = &ColorSyncConfig{
		Enabled:   true,
		AutoApply: true,
		Applications: map[string]bool{
			"alacritty": true,
			"waybar":    true,
			"kitty":     true,
			"rofi":      true,
			"dunst":     true,
			"foot":      true,
			"termite":   false,
		},
	}
	csm.saveConfig()
}

// saveConfig saves the color sync configuration
func (csm *ColorSyncManager) saveConfig() error {
	data, err := json.MarshalIndent(csm.config, "", "  ")
	if err != nil {
		return err
	}

	configDir := filepath.Dir(csm.configFile)
	makeDir(configDir)

	return os.WriteFile(csm.configFile, data, 0644)
}

// ApplyTheme extracts and applies colors from a GTK theme
func (csm *ColorSyncManager) ApplyTheme(themeName string) error {
	if !csm.config.Enabled {
		log.Debug("Color sync is disabled")
		return nil
	}

	log.Infof(">>> Extracting colors from GTK theme: %s", themeName)

	palette, err := csm.extractor.ExtractColors(themeName)
	if err != nil {
		return fmt.Errorf("failed to extract colors: %w", err)
	}

	log.Debugf("Extracted palette: bg=%s, fg=%s", palette.Background, palette.Foreground)

	// Apply to templates
	if err := csm.templates.ApplyColors(palette, csm.config.Applications); err != nil {
		return fmt.Errorf("failed to apply colors: %w", err)
	}

	// Save to config
	csm.config.LastTheme = themeName
	csm.config.LastColors = palette
	csm.saveConfig()

	log.Info("✓ Successfully applied colors!")
	return nil
}

// IsEnabled returns whether color sync is enabled
func (csm *ColorSyncManager) IsEnabled() bool {
	return csm.config.Enabled
}

// SetEnabled enables or disables color sync
func (csm *ColorSyncManager) SetEnabled(enabled bool) {
	csm.config.Enabled = enabled
	csm.saveConfig()
}

// IsAutoApply returns whether auto-apply is enabled
func (csm *ColorSyncManager) IsAutoApply() bool {
	return csm.config.AutoApply
}

// SetAutoApply sets auto-apply mode
func (csm *ColorSyncManager) SetAutoApply(autoApply bool) {
	csm.config.AutoApply = autoApply
	csm.saveConfig()
}

// IsAppEnabled returns whether an app is enabled for sync
func (csm *ColorSyncManager) IsAppEnabled(appName string) bool {
	enabled, exists := csm.config.Applications[appName]
	return exists && enabled
}

// SetAppEnabled enables or disables an app for sync
func (csm *ColorSyncManager) SetAppEnabled(appName string, enabled bool) {
	csm.config.Applications[appName] = enabled
	csm.saveConfig()
}

// GetApplications returns the list of supported applications
func (csm *ColorSyncManager) GetApplications() []string {
	apps := []string{"alacritty", "waybar", "kitty", "rofi", "dunst", "foot", "termite"}
	return apps
}

// ExportCurrentPalette exports the current palette to a file
func (csm *ColorSyncManager) ExportCurrentPalette(filename string) error {
	if csm.config.LastColors == nil {
		return fmt.Errorf("no palette to export")
	}

	data, err := json.MarshalIndent(csm.config.LastColors, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}
