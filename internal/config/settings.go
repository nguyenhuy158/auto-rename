package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

type Settings struct {
	Dir                string   `json:"dir"`
	DryRun             bool     `json:"dry_run"`
	WebPort            string   `json:"web_port"`
	WebOnly            bool     `json:"web_only"`
	QuarantineDir      string   `json:"quarantine_dir"`
	Extensions         []string `json:"extensions"`
	Cron               bool     `json:"cron"`
	CronExpr           string   `json:"cron_expr"`
	RenameSubfolder    bool     `json:"rename_subfolder"`
	FormatSize         string   `json:"format_size"`
	FormatDatetime     string   `json:"format_datetime"`
	FormatTimezone     string   `json:"format_timezone"`
	FormatSizeDecimals int      `json:"format_size_decimals"`
	UpdatedAt          string   `json:"updated_at"`
}

func GetDefaultSettings() Settings {
	return Settings{
		Dir:                "/app/data",
		DryRun:             true,
		WebPort:            "8080",
		WebOnly:            false,
		QuarantineDir:      "/app/quarantine",
		Extensions:         []string{".mp4", ".mov", ".mkv", ".avi", ".webm", ".m4v", ".wmv"},
		Cron:               false,
		CronExpr:           "*/30 * * * *",
		RenameSubfolder:    true,
		FormatSize:         "auto",
		FormatDatetime:     "2006-01-02 15:04:05",
		FormatTimezone:     "UTC+7",
		FormatSizeDecimals: 2,
		UpdatedAt:          time.Now().Format(time.RFC3339),
	}
}

func LoadSettings(filepath string) (Settings, error) {
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

	if settings.WebPort == "" {
		settings.WebPort = "8080"
	}
	if settings.QuarantineDir == "" {
		settings.QuarantineDir = "/app/quarantine"
	}
	if len(settings.Extensions) == 0 {
		settings.Extensions = GetDefaultSettings().Extensions
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

func SaveSettings(settings Settings, filePath string) error {
	settings.UpdatedAt = time.Now().Format(time.RFC3339)

	data, err := json.MarshalIndent(settings, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	tempFile := filePath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp settings file: %w", err)
	}

	if err := os.Rename(tempFile, filePath); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to save settings file: %w", err)
	}

	log.Printf("Saved settings to: %s", filePath)
	return nil
}

func ValidateSettings(settings Settings) error {
	if settings.WebPort == "" {
		return fmt.Errorf("web_port cannot be empty")
	}

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

	validTimezones := map[string]bool{
		"UTC":   true,
		"Local": true,
		"UTC+7": true,
	}
	if !validTimezones[settings.FormatTimezone] {
		return fmt.Errorf("invalid format_timezone: %s (must be: UTC, UTC+7, or Local)", settings.FormatTimezone)
	}

	if settings.FormatSizeDecimals < 0 || settings.FormatSizeDecimals > 5 {
		return fmt.Errorf("invalid format_size_decimals: %d (must be 0-5)", settings.FormatSizeDecimals)
	}

	if settings.Cron && settings.CronExpr == "" {
		return fmt.Errorf("cron_expr cannot be empty when cron is enabled")
	}

	if !settings.WebOnly && settings.Dir != "" {
		if _, err := os.Stat(settings.Dir); os.IsNotExist(err) {
			return fmt.Errorf("directory does not exist: %s", settings.Dir)
		}
	}

	return nil
}
