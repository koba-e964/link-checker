package main

import (
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

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
	URI string `toml:"uri"`

	// h1: SHA-384
	HashVersion   string `toml:"hash_version,omitempty"`
	HashOfContent string `toml:"hash_of_content,omitempty"`
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
	encoder.Indent = ""
	return encoder.Encode(lockFile)
}

// fetchURLAndComputeSHA384 fetches the content of a URL and computes its SHA384 hash
func fetchURLAndComputeSHA384(url string) (string, error) {
	// TODO: move to http_accessor.go
	// TODO: add a function to perform http.NewRequest("GET", ...) to parameters for easy testing
	client := http.Client{
		Timeout: 30 * time.Second,
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "link-checker from https://github.com/koba-e964/link-checker")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP request to %s failed with status code: %d", url, resp.StatusCode)
	}

	// Limit response body to 100MB to prevent memory exhaustion
	limitedBody := io.LimitReader(resp.Body, 100*1024*1024)

	// Compute SHA384 hash
	// TODO: separate hashing logic into another file
	hasher := sha512.New384()
	if _, err := io.Copy(hasher, limitedBody); err != nil {
		return "", err
	}

	hashBytes := hasher.Sum(nil)
	return hex.EncodeToString(hashBytes), nil
}

func addLockEntry(lockFilePath string, uri string, allowUpdate bool) error {
	lockFile, err := readLockFile(lockFilePath)
	if err != nil {
		return err
	}
	updatedLockFile, err := addLockEntryPure(lockFile, uri, allowUpdate)
	if err != nil {
		return err
	}

	return writeLockFile(lockFilePath, updatedLockFile)
}

func addLockEntryPure(lockFile *LockFile, uri string, allowUpdate bool) (*LockFile, error) {
	// Check if URI already exists
	index := -1
	for i, lock := range lockFile.Locks {
		if lock.URI == uri {
			if !allowUpdate {
				return nil, fmt.Errorf("URI %s already exists in lock file", uri)
			} else {
				index = i
				break
			}
		}
	}

	// Fetch URL and compute SHA384 hash
	sha384Hash, err := fetchURLAndComputeSHA384(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL and compute hash: %w", err)
	}

	// Add new lock entry with computed hash
	newLock := Lock{
		URI:           uri,
		HashVersion:   "h1",
		HashOfContent: sha384Hash,
	}
	if allowUpdate && index != -1 {
		oldHash := lockFile.Locks[index].HashOfContent
		if oldHash == sha384Hash {
			// No change in hash, skip update
			log.Printf("No change in content for %s, skipping update\n", uri)
			return lockFile, nil
		} else {
			log.Printf("Content changed for %s, updating hash\n", uri)
		}
		lockFile.Locks[index] = newLock
	} else {
		lockFile.Locks = append(lockFile.Locks, newLock)
	}
	slices.SortFunc(lockFile.Locks, func(a, b Lock) int {
		return strings.Compare(a.URI, b.URI)
	})
	return lockFile, nil
}
