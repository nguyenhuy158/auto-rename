// Config logic for auto-rename
package config

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config lưu thông tin cấu hình ứng dụng
type Config struct {
	Dir             string
	DryRun          bool
	WebPort         string
	WebOnly         bool
	DbPath          string
	Cron            bool
	RenameSubfolder bool
}

// parseFlags lấy config từ flag và env
func ParseFlags() Config {
	_ = godotenv.Load()
	envDir := os.Getenv("DIR")
	envDryRun := getBoolEnv("DRY_RUN", false)
	envWebPort := getEnv("WEB_PORT", "8080")
	envWebOnly := getBoolEnv("WEB_ONLY", false)
	envDbPath := getEnv("DB_PATH", "./file_renames.db")
	envCron := getBoolEnv("CRON", false)
	envRenameSubfolder := getBoolEnv("RENAME_SUBFOLDER", true)

	log.Printf("Reading configuration...")
	log.Printf("envDir=%v", envDir)
	log.Printf("envDryRun=%v", envDryRun)
	log.Printf("envWebPort=%v", envWebPort)
	log.Printf("envWebOnly=%v", envWebOnly)
	log.Printf("envDbPath=%v", envDbPath)
	log.Printf("envCron=%v", envCron)
	log.Printf("envRenameSubfolder=%v", envRenameSubfolder)
	log.Printf("Command line args: %v", os.Args)

	var config Config
	flag.StringVar(&config.Dir, "dir", envDir, "Directory containing files to rename (can also set DIR env var)")
	flag.BoolVar(&config.DryRun, "dry-run", envDryRun, "Preview changes without actually renaming files (can also set DRY_RUN env var)")
	flag.StringVar(&config.WebPort, "web-port", envWebPort, "Port for web interface (can also set WEB_PORT env var)")
	flag.BoolVar(&config.WebOnly, "web-only", envWebOnly, "Only start web server without renaming files (can also set WEB_ONLY env var)")
	flag.StringVar(&config.DbPath, "db", envDbPath, "SQLite database path (can also set DB_PATH env var)")
	flag.BoolVar(&config.Cron, "cron", envCron, "Continuously scan directory every minute (can also set CRON env var)")
	flag.BoolVar(&config.RenameSubfolder, "rename-subfolder", envRenameSubfolder, "Allow renaming files in subfolders (can also set RENAME_SUBFOLDER env var)")
	flag.Parse()

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
