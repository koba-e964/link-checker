package main

import (
	"errors"
	"os"

	"github.com/BurntSushi/toml"
)

const configFilePath = "./check_links_config.toml"
const lockFilePath = "./check_links.lock"

type Config struct {
	RetryCount int `toml:"retry_count"`
	// All text files' extensions
	TextFileExtensions []string       `toml:"text_file_extensions"`
	Ignores            []Ignore       `toml:"ignores"`
	PrefixIgnores      []PrefixIgnore `toml:"prefix_ignores"`
}

type LockFile struct {
	Locks []Lock `toml:"locks"`
}

type Lock struct {
	URI  string     `toml:"uri"`
	Lock LockDetail `toml:"lock"`
}

type LockDetail struct {
	Include []string  `toml:"include"`
	Hash    *LockHash `toml:"hash,omitempty"`
}

type LockHash struct {
	SHA256 string `toml:"sha256,omitempty"`
	SHA384 string `toml:"sha384,omitempty"`
}

type Ignore struct {
	URL                    string   `toml:"url"`
	HasTLSError            bool     `toml:"has_tls_error"`
	Codes                  []int    `toml:"codes"`
	Reason                 string   `toml:"reason"`
	ConsideredAlternatives []string `toml:"considered_alternatives"`
}

type PrefixIgnore struct {
	Prefix string `toml:"prefix"`
	Reason string `toml:"reason"`
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
	for _, prefixIgnore := range c.PrefixIgnores {
		if prefixIgnore.Prefix == "" {
			return errors.New("prefix cannot be empty")
		}
		if prefixIgnore.Reason == "" {
			return errors.New("reason cannot be empty for prefix_ignores")
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

func readLockFile(lockFilePath string) (*LockFile, error) {
	var lockFile LockFile
	bytes, err := os.ReadFile(lockFilePath)
	if err != nil {
		// If lock file doesn't exist, return empty lockfile
		if os.IsNotExist(err) {
			return &LockFile{Locks: []Lock{}}, nil
		}
		return nil, err
	}
	_, err = toml.Decode(string(bytes), &lockFile)
	if err != nil {
		return nil, err
	}
	return &lockFile, nil
}

func writeLockFile(lockFilePath string, lockFile *LockFile) error {
	f, err := os.Create(lockFilePath)
	if err != nil {
		return err
	}
	defer f.Close()
	encoder := toml.NewEncoder(f)
	return encoder.Encode(lockFile)
}

func addLockEntry(lockFilePath string, uri string) error {
	lockFile, err := readLockFile(lockFilePath)
	if err != nil {
		return err
	}
	
	// Check if URI already exists
	for _, lock := range lockFile.Locks {
		if lock.URI == uri {
			return errors.New("URI already exists in lock file")
		}
	}
	
	// Add new lock entry with empty include array
	newLock := Lock{
		URI: uri,
		Lock: LockDetail{
			Include: []string{},
		},
	}
	lockFile.Locks = append(lockFile.Locks, newLock)
	
	return writeLockFile(lockFilePath, lockFile)
}
