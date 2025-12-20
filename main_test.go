package main

import (
	"reflect"
	"testing"
)

func TestStripTitleSuffix(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Basic :title suffix
		{"https://example.com:title", "https://example.com"},
		{"http://example.com:title", "http://example.com"},
		// :title=xxx suffix
		{"https://example.com:title=", "https://example.com"},
		{"https://example.com:title=Page Title", "https://example.com"},
		// URLs without :title suffix should be unchanged
		{"https://example.com", "https://example.com"},
		{"https://example.com:8080", "https://example.com:8080"},
		{"https://example.com:8080/path", "https://example.com:8080/path"},
		// Real examples from the issue
		{"https://www.ibjapan.jp/information/2023/09/22.html:title", "https://www.ibjapan.jp/information/2023/09/22.html"},
		{"https://www.reddit.com/r/PromptEngineering/comments/1okppqe/i_made_chatgpt_stop_being_nice_and_its_the_best/:title=", "https://www.reddit.com/r/PromptEngineering/comments/1okppqe/i_made_chatgpt_stop_being_nice_and_its_the_best/"},
	}

	for _, test := range tests {
		result := stripTitleSuffix(test.input)
		if result != test.expected {
			t.Errorf("stripTitleSuffix(%q) = %q, want %q", test.input, result, test.expected)
		}
	}
}

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

func TestCheckFileWithTitleSuffix(t *testing.T) {
	accessed := []string{}
	var httpHead HttpAccessor = func(url string) (int, error) {
		accessed = append(accessed, url)
		return 200, nil
	}
	seen := map[string]struct{}{}
	ignores := map[string]*Ignore{}
	readFile := getReadFileMock([]readFileEntry{
		{"dummy", "https://www.ibjapan.jp/information/2023/09/22.html:title\nhttp://example.com:title=Page Title\n"},
	})
	err := checkFile("dummy", 1, ignores, seen, readFile, httpHead)
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	expectedAccessed := []string{
		// HTTP URLs are processed first, then HTTPS
		"http://example.com",
		"https://www.ibjapan.jp/information/2023/09/22.html",
	}
	if !reflect.DeepEqual(accessed, expectedAccessed) {
		t.Errorf("accessed = %v, want %v", accessed, expectedAccessed)
	}
}
