package main

import (
	"reflect"
	"testing"
)

func TestCheckURLLiveness(t *testing.T) {
	seen := map[string]struct{}{}
	httpHead := getHttpHeadMock([]httpHeadEntry{
		{"dummy-200", 200},
		{"dummy-404", 404},
	})
	err := checkURLLiveness("dummy-200", 1, nil, seen, httpHead)
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	err = checkURLLiveness("dummy-404", 1, nil, seen, httpHead)
	if err == nil {
		t.Errorf("err = nil, want non-nil")
	}
	expectedSeen := map[string]struct{}{
		"dummy-200": {},
		"dummy-404": {},
	}
	if !reflect.DeepEqual(seen, expectedSeen) {
		t.Errorf("seen = %v, want %v", seen, expectedSeen)
	}
}

func TestCheckFile(t *testing.T) {
	accessed := []string{}
	var httpHead HttpAccessor = func(url string) (int, error) {
		accessed = append(accessed, url)
		return 200, nil
	}
	seen := map[string]struct{}{}
	ignores := map[string]*Ignore{}
	readFile := getReadFileMock([]readFileEntry{
		{"dummy", "http://dummy-200\nhttps://dummy-404\n"},
		{"dummy2", "http://dummy-200\nhttps://dummy-404\n"},
	})
	err := checkFile("dummy", 1, ignores, seen, readFile, httpHead)
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	err = checkFile("dummy2", 1, ignores, seen, readFile, httpHead)
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	expectedAccessed := []string{
		// Only once for each URL
		"http://dummy-200",
		"https://dummy-404",
	}
	if !reflect.DeepEqual(accessed, expectedAccessed) {
		t.Errorf("accessed = %v, want %v", accessed, expectedAccessed)
	}
}
