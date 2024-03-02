package main

import "net/http"

type HttpAccessor = func(url string) (int, error)

// Returns the status code.
func httpHead(url string) (int, error) {
	resp, err := http.Head(url)
	if err != nil {
		return 0, err
	}
	return resp.StatusCode, nil
}
