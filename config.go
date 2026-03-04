package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"syscall"
)

type Config struct {
	Locale           Locale `json:"locale"`
	Enabled          bool   `json:"enabled"`
	Hotkey           string `json:"hotkey"`
	LaunchAtLogin    *bool  `json:"launch_at_login"`
	ClipboardFallback *bool `json:"clipboard_fallback"` // pointer so nil = not yet set
}

func defaultConfig() *Config {
	return &Config{
		Locale:  EnGB,
		Enabled: true,
		Hotkey:  "Ctrl+.",
	}
}

func lockPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "tyop", "tyop.lock")
}

// acquireLock creates an exclusive flock on a lock file.
// Returns false if another instance already holds it.
func acquireLock() bool {
	path := lockPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return true // can't create dir — let it run anyway
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return true
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return false // another instance holds the lock
	}
	// Keep the file open for the lifetime of the process — lock is released on exit.
	return true
}

func configPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "tyop", "config.json")
}

func loadConfig() *Config {
	cfg := defaultConfig()
	data, err := os.ReadFile(configPath())
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal(data, cfg)
	// Validate
	if cfg.Locale != EnGB && cfg.Locale != EnUS {
		cfg.Locale = EnGB
	}
	if cfg.Hotkey == "" {
		cfg.Hotkey = "Ctrl+."
	}
	return cfg
}

func saveConfig(cfg *Config) {
	path := configPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o644)
}
