// Settings management - JSON-based configuration
package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

// Settings represents the application settings stored in JSON
type Settings struct {
	Dir                  string `json:"dir"`
	DryRun               bool   `json:"dry_run"`
	WebPort              string `json:"web_port"`
	WebOnly              bool   `json:"web_only"`
	DbPath               string `json:"db_path"`
	Cron                 bool   `json:"cron"`
	CronExpr             string `json:"cron_expr"`
	RenameSubfolder      bool   `json:"rename_subfolder"`
	FormatSize           string `json:"format_size"`
	FormatDatetime       string `json:"format_datetime"`
	FormatTimezone       string `json:"format_timezone"`
	FormatSizeDecimals   int    `json:"format_size_decimals"`
	UpdatedAt            string `json:"updated_at"`
}

// GetDefaultSettings returns factory default settings
func GetDefaultSettings() Settings {
	return Settings{
		Dir:                "/app/data",
		DryRun:             false,
		WebPort:            "8080",
		WebOnly:            true,
		DbPath:             "./file_renames.db",
		Cron:               true,
		CronExpr:           "*/30 * * * *",
		RenameSubfolder:    true,
		FormatSize:         "auto",
		FormatDatetime:     "2006-01-02 15:04:05",
		FormatTimezone:     "UTC+7",
		FormatSizeDecimals: 2,
		UpdatedAt:          time.Now().Format(time.RFC3339),
	}
}

// LoadSettings loads settings from JSON file
// Returns default settings if file doesn't exist
func LoadSettings(filepath string) (Settings, error) {
	// If file doesn't exist, return defaults
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		log.Printf("Settings file not found: %s, using defaults", filepath)
		return GetDefaultSettings(), nil
	}

	data, err := os.ReadFile(filepath)
	if err != nil {
		log.Printf("Error reading settings file: %v, using defaults", err)
		return GetDefaultSettings(), nil
	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		log.Printf("Error parsing settings file: %v, using defaults", err)
		return GetDefaultSettings(), nil
	}

	// Validate critical fields
	if settings.WebPort == "" {
		settings.WebPort = "8080"
	}
	if settings.DbPath == "" {
		settings.DbPath = "./file_renames.db"
	}
	if settings.CronExpr == "" {
		settings.CronExpr = "*/30 * * * *"
	}
	if settings.FormatSize == "" {
		settings.FormatSize = "auto"
	}
	if settings.FormatDatetime == "" {
		settings.FormatDatetime = "2006-01-02 15:04:05"
	}
	if settings.FormatTimezone == "" {
		settings.FormatTimezone = "UTC+7"
	}
	if settings.FormatSizeDecimals == 0 {
		settings.FormatSizeDecimals = 2
	}

	log.Printf("Loaded settings from: %s", filepath)
	return settings, nil
}

// SaveSettings saves settings to JSON file
func SaveSettings(settings Settings, filePath string) error {
	// Update timestamp
	settings.UpdatedAt = time.Now().Format(time.RFC3339)

	// Marshal to JSON with pretty printing
	data, err := json.MarshalIndent(settings, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	// Write to file with atomic operation (write to temp, then rename)
	tempFile := filePath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp settings file: %w", err)
	}

	// Rename temp file to actual file (atomic)
	if err := os.Rename(tempFile, filePath); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to save settings file: %w", err)
	}

	log.Printf("Saved settings to: %s", filePath)
	return nil
}

// ValidateSettings validates the settings structure
func ValidateSettings(settings Settings) error {
	// Validate web port
	if settings.WebPort == "" {
		return fmt.Errorf("web_port cannot be empty")
	}

	// Validate format_size
	validSizes := map[string]bool{
		"auto": true,
		"B":    true,
		"KB":   true,
		"MB":   true,
		"GB":   true,
	}
	if !validSizes[settings.FormatSize] {
		return fmt.Errorf("invalid format_size: %s (must be: auto, B, KB, MB, GB)", settings.FormatSize)
	}

	// Validate format_timezone
	validTimezones := map[string]bool{
		"UTC":   true,
		"Local": true,
		"UTC+7": true,
	}
	if !validTimezones[settings.FormatTimezone] {
		return fmt.Errorf("invalid format_timezone: %s (must be: UTC, UTC+7, or Local)", settings.FormatTimezone)
	}

	// Validate format_size_decimals
	if settings.FormatSizeDecimals < 0 || settings.FormatSizeDecimals > 5 {
		return fmt.Errorf("invalid format_size_decimals: %d (must be 0-5)", settings.FormatSizeDecimals)
	}

	// Validate cron expression if cron is enabled
	if settings.Cron && settings.CronExpr == "" {
		return fmt.Errorf("cron_expr cannot be empty when cron is enabled")
	}

	// Validate directory if not web_only
	if !settings.WebOnly && settings.Dir != "" {
		if _, err := os.Stat(settings.Dir); os.IsNotExist(err) {
			return fmt.Errorf("directory does not exist: %s", settings.Dir)
		}
	}

	return nil
}

// MigrateFromEnv loads settings from .env file if settings.json doesn't exist
// Returns true if migration was performed
func MigrateFromEnv(envFilepath, settingsFilepath string) bool {
	// Check if settings.json already exists
	if _, err := os.Stat(settingsFilepath); err == nil {
		log.Printf("Settings file already exists: %s", settingsFilepath)
		return false
	}

	// Check if .env exists
	if _, err := os.Stat(envFilepath); os.IsNotExist(err) {
		log.Printf("No .env file found for migration")
		return false
	}

	log.Printf("Migrating settings from .env to settings.json")

	// Load .env using godotenv
	_ = os.Setenv("DIR", "")
	envFile, err := os.ReadFile(envFilepath)
	if err != nil {
		log.Printf("Error reading .env file: %v", err)
		return false
	}

	// Parse .env manually (simple approach)
	envVars := make(map[string]string)
	for _, line := range splitLines(string(envFile)) {
		if key, value := parseEnvLine(line); key != "" {
			envVars[key] = value
		}
	}

	// Create settings from env vars
	settings := GetDefaultSettings()
	if v, ok := envVars["DIR"]; ok && v != "" {
		settings.Dir = v
	}
	if v, ok := envVars["DRY_RUN"]; ok && v != "" {
		settings.DryRun = v == "true"
	}
	if v, ok := envVars["WEB_PORT"]; ok && v != "" {
		settings.WebPort = v
	}
	if v, ok := envVars["WEB_ONLY"]; ok && v != "" {
		settings.WebOnly = v == "true"
	}
	if v, ok := envVars["DB_PATH"]; ok && v != "" {
		settings.DbPath = v
	}
	if v, ok := envVars["CRON"]; ok && v != "" {
		settings.Cron = v == "true"
	}
	if v, ok := envVars["CRON_EXPR"]; ok && v != "" {
		settings.CronExpr = v
	}
	if v, ok := envVars["RENAME_SUBFOLDER"]; ok && v != "" {
		settings.RenameSubfolder = v == "true"
	}

	// Save migrated settings
	if err := SaveSettings(settings, settingsFilepath); err != nil {
		log.Printf("Error saving migrated settings: %v", err)
		return false
	}

	log.Printf("Successfully migrated settings from .env to settings.json")
	return true
}

// Helper functions
func splitLines(s string) []string {
	var lines []string
	var current string
	for _, r := range s {
		if r == '\n' {
			if current != "" {
				lines = append(lines, current)
			}
			current = ""
		} else {
			current += string(r)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func parseEnvLine(line string) (string, string) {
	// Skip comments and empty lines
	if line == "" || line[0] == '#' {
		return "", ""
	}

	// Find the first '='
	for i, r := range line {
		if r == '=' {
			key := line[:i]
			value := line[i+1:]
			// Remove quotes if present
			if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
				value = value[1 : len(value)-1]
			}
			return key, value
		}
	}
	return "", ""
}
