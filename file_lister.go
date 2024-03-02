package main

import (
	"errors"
	"log"
	"os/exec"
	"strings"
)

// listFiles lists files using git ls-files.
func listFiles() ([]string, error) {
	cmd := exec.Command("git", "ls-files")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("git ls-files failed")
		return nil, errors.New("git ls-files failed")
	}
	paths := strings.Split(strings.ReplaceAll(string(output), "\r\n", "\n"), "\n")
	paths = paths[:len(paths)-1] // excludes the last element after the last newline
	return paths, nil
}
