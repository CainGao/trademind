package config

import (
	"os"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()
	if cfg == nil {
		t.Fatal("Default() should return non-nil config")
	}
	if cfg.Server.Port != 7789 {
		t.Errorf("Default port = %d, want 7789", cfg.Server.Port)
	}
	if cfg.App.Name != "TradeMind AI" {
		t.Errorf("Default app name = %q, want 'TradeMind AI'", cfg.App.Name)
	}
	if cfg.App.Version == "" {
		t.Error("Default version should not be empty")
	}
	if cfg.Database.Path == "" {
		t.Error("Default DB path should not be empty")
	}
	if cfg.Log.Level == "" {
		t.Error("Default log level should not be empty")
	}
}

func TestLoad_Defaults(t *testing.T) {
	// Clean env vars to ensure defaults
	os.Unsetenv("TRADEMIND_PORT")
	os.Unsetenv("TRADEMIND_DB_PATH")
	os.Unsetenv("TRADEMIND_LOG_LEVEL")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Server.Port != 7789 {
		t.Errorf("Default port = %d, want 7789", cfg.Server.Port)
	}
}

func TestLoad_EnvOverride_Port(t *testing.T) {
	os.Setenv("TRADEMIND_PORT", "9999")
	defer os.Unsetenv("TRADEMIND_PORT")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Server.Port != 9999 {
		t.Errorf("Overridden port = %d, want 9999", cfg.Server.Port)
	}
}

func TestLoad_EnvOverride_DBPath(t *testing.T) {
	os.Setenv("TRADEMIND_DB_PATH", "/tmp/test-trademind.db")
	defer os.Unsetenv("TRADEMIND_DB_PATH")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Database.Path != "/tmp/test-trademind.db" {
		t.Errorf("Overridden DB path = %q, want '/tmp/test-trademind.db'", cfg.Database.Path)
	}
}

func TestLoad_EnvOverride_LogLevel(t *testing.T) {
	os.Setenv("TRADEMIND_LOG_LEVEL", "debug")
	defer os.Unsetenv("TRADEMIND_LOG_LEVEL")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("Overridden log level = %q, want 'debug'", cfg.Log.Level)
	}
}

func TestLoad_CreatesDirectories(t *testing.T) {
	// Load should create runtime and logs directories
	os.Unsetenv("TRADEMIND_DB_PATH")
	_, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	// Directories should exist after Load
	if info, err := os.Stat("runtime"); err != nil || !info.IsDir() {
		t.Error("Load should create runtime/ directory")
	}
	if info, err := os.Stat("logs"); err != nil || !info.IsDir() {
		t.Error("Load should create logs/ directory")
	}
}
