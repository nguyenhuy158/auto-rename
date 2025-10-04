package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

func TestGenerateUUIDName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantExt string
	}{
		{
			name:    "file with extension",
			input:   "test.txt",
			wantExt: ".txt",
		},
		{
			name:    "file without extension",
			input:   "test",
			wantExt: "",
		},
		{
			name:    "file with multiple dots",
			input:   "test.backup.sql",
			wantExt: ".sql",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateUUIDName(tt.input)

			// Check if result ends with expected extension
			if filepath.Ext(result) != tt.wantExt {
				t.Errorf("generateUUIDName() extension = %v, want %v", filepath.Ext(result), tt.wantExt)
			}

			// Check if the name part (without extension) is a valid UUID
			nameWithoutExt := result[:len(result)-len(tt.wantExt)]
			if _, err := uuid.Parse(nameWithoutExt); err != nil {
				t.Errorf("generateUUIDName() produced invalid UUID: %v", err)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "test_validate_config")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				Dir:    tempDir,
				DryRun: false,
			},
			wantErr: false,
		},
		{
			name: "empty directory",
			config: Config{
				Dir:    "",
				DryRun: false,
			},
			wantErr: true,
		},
		{
			name: "non-existent directory",
			config: Config{
				Dir:    "/non/existent/path",
				DryRun: false,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
