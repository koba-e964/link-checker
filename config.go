package main

import (
	"errors"
	"os"

	"github.com/BurntSushi/toml"
)

const configFilePath = "./check_links_config.toml"

type Config struct {
	RetryCount int `toml:"retry_count"`
	// All text files' extensions
	TextFileExtensions []string `toml:"text_file_extensions"`
	Ignores            []Ignore `toml:"ignores"`
}

type Ignore struct {
	URL                    string   `toml:"url"`
	HasTLSError            bool     `toml:"has_tls_error"`
	Codes                  []int    `toml:"codes"`
	Reason                 string   `toml:"reason"`
	ConsideredAlternatives []string `toml:"considered_alternatives"`
}

func (c *Config) Validate() error {
	if len(c.TextFileExtensions) == 0 {
		return errors.New("text_file_extensions cannot be empty")
	}
	for _, ignore := range c.Ignores {
		if ignore.URL == "" {
			return errors.New("url cannot be empty")
		}
		if len(ignore.Codes) == 0 && !ignore.HasTLSError {
			return errors.New("codes cannot be empty when has_tls_error = false")
		}
		if ignore.Reason == "" {
			return errors.New("reason cannot be empty")
		}
		if len(ignore.ConsideredAlternatives) == 0 {
			return errors.New("considered_alternatives cannot be empty")
		}
	}
	return nil
}

func readConfig(configFilePath string) (*Config, error) {
	var config Config
	bytes, err := os.ReadFile(configFilePath)
	if err != nil {
		return nil, err
	}
	toml.Decode(string(bytes), &config)
	return &config, nil
}
