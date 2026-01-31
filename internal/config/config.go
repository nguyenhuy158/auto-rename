// Config logic for auto-rename
package config

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
)

// Config lưu thông tin cấu hình ứng dụng
type Config struct {
	Dir                  string
	DryRun               bool
	WebPort              string
	WebOnly              bool
	DbPath               string
	Cron                 bool
	CronExpr             string
	RenameSubfolder      bool
	FormatSize           string
	FormatDatetime       string
	FormatTimezone       string
	FormatSizeDecimals   int
}

// parseFlags lấy config từ settings.json và CLI flags
func ParseFlags() Config {
	// Parse -config flag first to determine settings file path
	var configPath string
	flag.StringVar(&configPath, "config", "", "Path to settings.json configuration file")
	
	// Do a preliminary parse just to get the config flag
	flag.Parse()
	
	log.Printf("Reading configuration...")
	
	// Use provided config path or default to settings.json
	if configPath == "" {
		configPath = "settings.json"
	}
	
	// Try to load settings.json, fallback to default if not found
	settings, err := LoadSettings(configPath)
	if err != nil {
		log.Printf("Warning loading %s: %v", configPath, err)
		settings = GetDefaultSettings()
	}

	log.Printf("settingsDir=%v", settings.Dir)
	log.Printf("settingsDryRun=%v", settings.DryRun)
	log.Printf("settingsWebPort=%v", settings.WebPort)
	log.Printf("settingsWebOnly=%v", settings.WebOnly)
	log.Printf("settingsDbPath=%v", settings.DbPath)
	log.Printf("settingsCron=%v", settings.Cron)
	log.Printf("settingsCronExpr=%v", settings.CronExpr)
	log.Printf("settingsRenameSubfolder=%v", settings.RenameSubfolder)
	log.Printf("settingsFormatSize=%v", settings.FormatSize)
	log.Printf("settingsFormatDatetime=%v", settings.FormatDatetime)
	log.Printf("settingsFormatTimezone=%v", settings.FormatTimezone)
	log.Printf("settingsFormatSizeDecimals=%v", settings.FormatSizeDecimals)
	log.Printf("Command line args: %v", os.Args)

	var config Config
	flag.StringVar(&config.Dir, "dir", settings.Dir, "Directory containing files to rename")
	flag.BoolVar(&config.DryRun, "dry-run", settings.DryRun, "Preview changes without actually renaming files")
	flag.StringVar(&config.WebPort, "web-port", settings.WebPort, "Port for web interface")
	flag.BoolVar(&config.WebOnly, "web-only", settings.WebOnly, "Only start web server without renaming files")
	flag.StringVar(&config.DbPath, "db", settings.DbPath, "SQLite database path")
	flag.BoolVar(&config.Cron, "cron", settings.Cron, "Continuously scan directory")
	flag.StringVar(&config.CronExpr, "cron-expr", settings.CronExpr, "Cron expression for scan schedule")
	flag.BoolVar(&config.RenameSubfolder, "rename-subfolder", settings.RenameSubfolder, "Allow renaming files in subfolders")
	flag.StringVar(&config.FormatSize, "format-size", settings.FormatSize, "Size format: auto, B, KB, MB, GB")
	flag.StringVar(&config.FormatDatetime, "format-datetime", settings.FormatDatetime, "DateTime format string")
	flag.StringVar(&config.FormatTimezone, "format-timezone", settings.FormatTimezone, "Timezone: UTC or Local")
	flag.IntVar(&config.FormatSizeDecimals, "format-size-decimals", settings.FormatSizeDecimals, "Size decimal places: 0-5")

	return config
}

// getEnv lấy biến môi trường hoặc trả về giá trị mặc định
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getBoolEnv lấy biến môi trường kiểu bool hoặc trả về giá trị mặc định
func getBoolEnv(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return boolValue
}

// validateConfig kiểm tra tính hợp lệ của config
func ValidateConfig(config Config) error {
	if config.WebOnly {
		return nil
	}
	if config.Dir == "" {
		return fmt.Errorf("directory path is required. Use -dir flag")
	}
	if _, err := os.Stat(config.Dir); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", config.Dir)
	}
	return nil
}
