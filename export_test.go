package main

import (
	"net/http"
	"os"
)

// file_reader.go
type readFileEntry struct {
	path string
	data string
}

func getReadFileMock(entries []readFileEntry) FileReader {
	return func(path string) ([]byte, error) {
		for _, entry := range entries {
			if entry.path == path {
				return []byte(entry.data), nil
			}
		}
		return nil, os.ErrNotExist
	}
}

// http_accessor.go
type httpHeadEntry struct {
	url        string
	statusCode int
}

func getHttpHeadMock(entries []httpHeadEntry) HttpAccessor {
	return func(url string) (int, error) {
		for _, entry := range entries {
			if entry.url == url {
				return entry.statusCode, nil
			}
		}
		return 0, http.ErrNotSupported
	}
}
