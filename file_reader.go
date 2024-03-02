package main

import "os"

type FileReader = func(string) ([]byte, error)

func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
