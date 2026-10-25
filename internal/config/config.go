package config

import (
	"flag"
	"fmt"
	"log"
	"os"
)

type Config struct {
	Dir                string
	DryRun             bool
	WebPort            string
	WebOnly            bool
	QuarantineDir      string
	Extensions         []string
	Cron               bool
	CronExpr           string
	RenameSubfolder    bool
	FormatSize         string
	FormatDatetime     string
	FormatTimezone     string
	FormatSizeDecimals int
}

func ParseFlags() Config {
	configPath := "settings.json"
	for index, argument := range os.Args[1:] {
		if argument == "-config" && index+2 < len(os.Args) {
			configPath = os.Args[index+2]
		}
		if len(argument) > len("-config=") && argument[:len("-config=")] == "-config=" {
			configPath = argument[len("-config="):]
		}
	}

	log.Printf("Reading configuration...")

	settings, err := LoadSettings(configPath)
	if err != nil {
		log.Printf("Warning loading %s: %v", configPath, err)
		settings = GetDefaultSettings()
	}

	log.Printf("settingsDir=%v", settings.Dir)
	log.Printf("settingsDryRun=%v", settings.DryRun)
	log.Printf("settingsWebPort=%v", settings.WebPort)
	log.Printf("settingsWebOnly=%v", settings.WebOnly)
	log.Printf("settingsCron=%v", settings.Cron)
	log.Printf("settingsCronExpr=%v", settings.CronExpr)
	log.Printf("settingsRenameSubfolder=%v", settings.RenameSubfolder)
	log.Printf("settingsFormatSize=%v", settings.FormatSize)
	log.Printf("settingsFormatDatetime=%v", settings.FormatDatetime)
	log.Printf("settingsFormatTimezone=%v", settings.FormatTimezone)
	log.Printf("settingsFormatSizeDecimals=%v", settings.FormatSizeDecimals)

	var config Config
	flag.String("config", configPath, "Path to settings.json configuration file")
	flag.StringVar(&config.Dir, "dir", settings.Dir, "Directory containing files to rename")
	flag.BoolVar(&config.DryRun, "dry-run", settings.DryRun, "Preview changes without actually renaming files")
	flag.StringVar(&config.WebPort, "web-port", settings.WebPort, "Port for web interface")
	flag.BoolVar(&config.WebOnly, "web-only", settings.WebOnly, "Only start web server without renaming files")
	flag.StringVar(&config.QuarantineDir, "quarantine-dir", settings.QuarantineDir, "Directory for recoverable duplicate files")
	config.Extensions = settings.Extensions
	flag.BoolVar(&config.Cron, "cron", settings.Cron, "Continuously scan directory")
	flag.StringVar(&config.CronExpr, "cron-expr", settings.CronExpr, "Cron expression for scan schedule")
	flag.BoolVar(&config.RenameSubfolder, "rename-subfolder", settings.RenameSubfolder, "Allow renaming files in subfolders")
	flag.StringVar(&config.FormatSize, "format-size", settings.FormatSize, "Size format: auto, B, KB, MB, GB")
	flag.StringVar(&config.FormatDatetime, "format-datetime", settings.FormatDatetime, "DateTime format string")
	flag.StringVar(&config.FormatTimezone, "format-timezone", settings.FormatTimezone, "Timezone: UTC or Local")
	flag.IntVar(&config.FormatSizeDecimals, "format-size-decimals", settings.FormatSizeDecimals, "Size decimal places: 0-5")

	flag.Parse()
	return config
}

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
