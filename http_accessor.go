package main

import "net/http"

type HttpAccessor = func(url string) (int, error)

// Returns the status code.
func httpHead(url string) (int, error) {
	client := http.Client{}
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "link-checker from https://github.com/koba-e964/link-checker")
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	return resp.StatusCode, nil
}
