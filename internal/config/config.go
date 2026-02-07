package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type Config struct {
	APIKey         string `json:"api_key"`
	BaseURL        string `json:"base_url"`
	ProxyURL       string `json:"proxy_url"`
	AutoAccept     bool   `json:"auto_accept"`
	CurrentModel   string `json:"current_model"`
	FirstSetup     bool   `json:"first_setup"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

const (
	appConfigDir          = ".config/ai2go"
	configFile            = "config.json"
	defaultModel          = ""
	defaultTimeoutSeconds = 120
)

func getConfigPath() (string, error) {
	var dir string
	if runtime.GOOS == "windows" {
		dir = os.Getenv("USERPROFILE")
		if dir == "" {
			return "", fmt.Errorf("USERPROFILE env var not set on Windows")
		}
	} else {
		dir = os.Getenv("HOME")
		if dir == "" {
			return "", fmt.Errorf("HOME env var not set")
		}
	}
	configDir := filepath.Join(dir, appConfigDir)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config dir: %w", err)
	}
	return filepath.Join(configDir, configFile), nil
}

func ConfigDir() (string, error) {
	var dir string
	if runtime.GOOS == "windows" {
		dir = os.Getenv("USERPROFILE")
		if dir == "" {
			return "", fmt.Errorf("USERPROFILE env var not set on Windows")
		}
	} else {
		dir = os.Getenv("HOME")
		if dir == "" {
			return "", fmt.Errorf("HOME env var not set")
		}
	}
	configDir := filepath.Join(dir, appConfigDir)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config dir: %w", err)
	}
	return configDir, nil
}
func Load() *Config {
	cfg := &Config{
		FirstSetup: true,
	}

	configPath, err := getConfigPath()
	if err != nil {
		fmt.Printf("Error determining config path: %v\n", err)
		return cfg // Return defaults
	}

	file, err := os.ReadFile(configPath)
	if err != nil {
		return cfg
	}
	if err := json.Unmarshal(file, cfg); err != nil {
		fmt.Printf("Warning: Error parsing config: %v. Using defaults.\n", err)
		return cfg
	}

	// Ensure current model is set
	if cfg.CurrentModel == "" {
		cfg.CurrentModel = defaultModel
	}

	if cfg.CurrentModel == "" {
		cfg.FirstSetup = true
	}

	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = defaultTimeoutSeconds
	}

	return cfg
}

func (c *Config) Save() error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}

func (c *Config) SetAPIKey(key string) {
	c.APIKey = key
	if err := c.Save(); err != nil {
		fmt.Printf("Error saving config: %v\n", err)
	}
}

func (c *Config) SetBaseURL(url string) {
	c.BaseURL = url
	if err := c.Save(); err != nil {
		fmt.Printf("Error saving config: %v\n", err)
	}
}

func (c *Config) SetProxyURL(url string) {
	c.ProxyURL = url
	if err := c.Save(); err != nil {
		fmt.Printf("Error saving config: %v\n", err)
	}
}

func (c *Config) SetCurrentModel(model string) {
	c.CurrentModel = model
	if err := c.Save(); err != nil {
		fmt.Printf("Error saving config: %v\n", err)
	}
}

func (c *Config) ToggleAutoAccept() {
	c.AutoAccept = !c.AutoAccept
	if err := c.Save(); err != nil {
		fmt.Printf("Error saving config: %v\n", err)
	}
}
