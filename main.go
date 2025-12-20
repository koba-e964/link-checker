package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var httpRegex = regexp.MustCompile("http://[-._%/[:alnum:]?:=+~@#&]+")
var httpsRegex = regexp.MustCompile("https://[-._%/[:alnum:]?:=+~@#&]+")
var titleRegex = regexp.MustCompile(`:title(=[^\s]*)?$`)

// stripTitleSuffix removes :title or :title=xxx suffix from URLs (Hatena notation)
func stripTitleSuffix(url string) string {
	return titleRegex.ReplaceAllString(url, "")
}

// shouldIgnoreByPrefix checks if a URL should be ignored based on prefix rules
func shouldIgnoreByPrefix(url string, prefixIgnores []PrefixIgnore) *PrefixIgnore {
	for i := range prefixIgnores {
		if strings.HasPrefix(url, prefixIgnores[i].Prefix) {
			return &prefixIgnores[i]
		}
	}
	return nil
}

// If ignore != nil, ignore.Codes will be used instead of the 2xx criterion.
// This function modifies seen.
func checkURLLiveness(url string, retryCount int, ignore *Ignore, seen map[string]struct{}, httpHead HttpAccessor) error {
	if _, ok := seen[url]; ok {
		// Already checked: not checking again
		return nil
	}
	seen[url] = struct{}{}
	for i := 0; i < retryCount; i++ {
		statusCode, err := httpHead(url)
		if err != nil {
			if ignore != nil && ignore.HasTLSError {
				// ok, but because ignore != nil, we need a log
				log.Printf("ok: url = %s, ignore = %v, err = %v\n", url, ignore, err)
				return nil
			}
			return err
		}
		if ignore != nil {
			ok := false
			for _, code := range ignore.Codes {
				if statusCode == code {
					ok = true
					break
				}
			}
			if ok {
				// ok, but because ignore != nil, we need a log
				log.Printf("ok: code = %d, url = %s , ignore = %v\n", statusCode, url, ignore)
				return nil
			}
		} else {
			if statusCode/100 == 2 {
				// ok
				return nil
			}
		}
		log.Printf("code = %d, url = %s, ignore = %v\n", statusCode, url, ignore)
		if i == retryCount-1 {
			return errors.New("invalid status code")
		} else {
			// exponential backoff
			time.Sleep((1 << i) * time.Second)
		}
	}
	return nil
}

// This function modifies seen.
func checkFile(path string, retryCount int, ignores map[string]*Ignore, prefixIgnores []PrefixIgnore, seen map[string]struct{}, readFile FileReader, httpHead HttpAccessor) (err error) {
	content, err := readFile(path)
	if err != nil {
		return err
	}

	all := httpRegex.FindAll(content, -1)
	var livenessErrors uint64 = 0
	for _, v := range all {
		url := string(v)
		url = stripTitleSuffix(url)

		// Check if URL matches any prefix ignore rules
		if prefixIgnore := shouldIgnoreByPrefix(url, prefixIgnores); prefixIgnore != nil {
			log.Printf("%s: HTTP link ignored by prefix: url = %s, prefix = %s, reason = %s\n",
				path, url, prefixIgnore.Prefix, prefixIgnore.Reason)
			continue
		}

		ignore := ignores[url]
		log.Printf("%s: HTTP link: url = %s\n", path, url)
		if thisError := checkURLLiveness(url, retryCount, ignore, seen, httpHead); thisError != nil {
			livenessErrors++
			log.Printf("%s: not alive: url = %s , thiserror = %v\n", path, url, thisError)
		}
	}

	all = httpsRegex.FindAll(content, -1)
	for _, v := range all {
		url := string(v)
		url = stripTitleSuffix(url)

		// Check if URL matches any prefix ignore rules
		if prefixIgnore := shouldIgnoreByPrefix(url, prefixIgnores); prefixIgnore != nil {
			log.Printf("%s: HTTPS link ignored by prefix: url = %s, prefix = %s, reason = %s\n",
				path, url, prefixIgnore.Prefix, prefixIgnore.Reason)
			continue
		}

		ignore := ignores[url]
		if thisError := checkURLLiveness(url, retryCount, ignore, seen, httpHead); thisError != nil {
			livenessErrors++
			log.Printf("%s: not alive: url = %s , thiserror = %v\n", path, url, thisError)
		}
	}
	if livenessErrors > 0 {
		err = fmt.Errorf("liveness check failed: path = %s , prev error = %w", path, err)
	}

	return err
}

func main() {
	// Check for add subcommand
	if len(os.Args) >= 3 && os.Args[1] == "add" {
		url := os.Args[2]
		if err := addLockEntry(lockFilePath, url); err != nil {
			log.Printf("Error adding lock entry: %v\n", err)
			os.Exit(1)
		}
		log.Printf("Successfully added %s to lock file\n", url)
		return
	}

	config, err := readConfig(configFilePath)
	if err != nil {
		panic(err)
	}
	if err := config.Validate(); err != nil {
		panic(err)
	}
	ignores := make(map[string]*Ignore)
	for _, ignore := range config.Ignores {
		// For handling of https://go.dev/blog/loopvar-preview
		ignoreCopied := ignore
		ignores[ignore.URL] = &ignoreCopied
	}

	numErrors := 0
	paths, err := listFiles()
	if err != nil {
		panic(err)
	}

	seen := make(map[string]struct{})
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			numErrors++
			log.Printf("path = %s, %v\n", path, err)
			continue
		}
		if info.IsDir() {
			continue
		}
		ext := filepath.Ext(path)
		ok := false
		for _, e := range config.TextFileExtensions {
			if ext == e {
				ok = true
				break
			}
		}
		if ok {
			if err := checkFile(path, config.RetryCount, ignores, config.PrefixIgnores, seen, readFile, httpHead); err != nil {
				numErrors++
				log.Printf("%v\n", err)
			}
		}
		continue
	}
	if numErrors > 0 {
		os.Exit(1)
	}
}
