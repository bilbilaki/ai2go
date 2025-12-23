package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	APIKey      string `json:"api_key"`
	BaseURL     string `json:"base_url"`
	ProxyURL    string `json:"proxy_url"`
	AutoAccept  bool   `json:"auto_accept"`
	CurrentModel string `json:"current_model"`
}

const (
	configFile   = "config.json"
	defaultModel = "grok-4-1-fast-reasoning"
)

func Load() *Config {
	cfg := &Config{
		BaseURL:     "https://api.x.ai",
		CurrentModel: defaultModel,
	}

	file, err := os.ReadFile(configFile)
	if err != nil {
		return cfg
	}

	json.Unmarshal(file, cfg)
	
	// Ensure current model is set
	if cfg.CurrentModel == "" {
		cfg.CurrentModel = defaultModel
	}
	
	return cfg
}

func (c *Config) Save() error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configFile, data, 0644)
}

func (c *Config) SetAPIKey(key string) {
	c.APIKey = key
	c.Save()
}

func (c *Config) SetBaseURL(url string) {
	c.BaseURL = url
	c.Save()
}

func (c *Config) SetProxyURL(url string) {
	c.ProxyURL = url
	c.Save()
}

func (c *Config) SetCurrentModel(model string) {
	c.CurrentModel = model
	c.Save()
}

func (c *Config) ToggleAutoAccept() {
	c.AutoAccept = !c.AutoAccept
	c.Save()
}
