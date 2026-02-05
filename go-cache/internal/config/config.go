package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/status-im/proxy-common/httpclient"
)

// APIKeysFile represents the structure of the API keys JSON file
type APIKeysFile struct {
	APITokens []string `json:"api_tokens"`
}

// AlchemyConfig represents Alchemy API configuration
type AlchemyConfig struct {
	APIKey   string                  `yaml:"api_key"`
	BaseURLs map[string]string       `yaml:"base_urls"`
	Retry    httpclient.RetryOptions `yaml:"retry"`
}

// ServerConfig represents HTTP server configuration
type ServerConfig struct {
	Port         string        `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

// Config represents the main configuration structure
type Config struct {
	Alchemy AlchemyConfig `yaml:"alchemy"`
	Server  ServerConfig  `yaml:"server"`
}

// LoadConfig loads configuration from file path
func LoadConfig(configPath string, logger *zap.Logger) (*Config, error) {
	logger.Info("Loading configuration", zap.String("path", configPath))

	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer func() { _ = file.Close() }()

	var config Config
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode YAML config: %w", err)
	}

	config.applyDefaults()
	config.expandEnvVars()

	return &config, nil
}

// applyDefaults sets default values for missing configuration
func (c *Config) applyDefaults() {
	// Alchemy defaults - use proxy-common's DefaultRetryOptions
	if c.Alchemy.Retry.MaxRetries == 0 {
		defaults := httpclient.DefaultRetryOptions()
		c.Alchemy.Retry.MaxRetries = defaults.MaxRetries
		c.Alchemy.Retry.BaseBackoff = defaults.BaseBackoff
		c.Alchemy.Retry.ConnectionTimeout = defaults.ConnectionTimeout
		c.Alchemy.Retry.RequestTimeout = defaults.RequestTimeout
		c.Alchemy.Retry.LogPrefix = "Alchemy"
	}

	if c.Server.Port == "" {
		c.Server.Port = "8080"
	}
	if c.Server.ReadTimeout == 0 {
		c.Server.ReadTimeout = 30 * time.Second
	}
	if c.Server.WriteTimeout == 0 {
		c.Server.WriteTimeout = 30 * time.Second
	}
}

// expandEnvVars expands environment variables in configuration
func (c *Config) expandEnvVars() {
	c.Alchemy.APIKey = os.ExpandEnv(c.Alchemy.APIKey)
}

// LoadAPIKeys loads API keys from JSON file
func LoadAPIKeys(keysPath string, logger *zap.Logger) ([]string, error) {
	logger.Info("Loading API keys", zap.String("path", keysPath))

	file, err := os.Open(keysPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open API keys file: %w", err)
	}
	defer func() { _ = file.Close() }()

	var keysFile APIKeysFile
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&keysFile); err != nil {
		return nil, fmt.Errorf("failed to decode JSON keys file: %w", err)
	}

	if len(keysFile.APITokens) == 0 {
		return nil, fmt.Errorf("no API tokens found in keys file")
	}

	logger.Info("API keys loaded successfully", zap.Int("count", len(keysFile.APITokens)))
	return keysFile.APITokens, nil
}
