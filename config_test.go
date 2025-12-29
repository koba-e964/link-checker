package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadLockFile(t *testing.T) {
	// Test reading non-existent file
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "check_links.lock")

	lockFile, err := readLockFile(lockPath)
	if err != nil {
		t.Errorf("readLockFile() error = %v, want nil", err)
	}
	if len(lockFile.Locks) != 0 {
		t.Errorf("readLockFile() returned %d locks, want 0", len(lockFile.Locks))
	}
}

func TestReadLockFileWithContent(t *testing.T) {
	// Test reading file with content
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "check_links.lock")

	content := `[[locks]]
uri = "https://example.org"
hash_version = "h1"
hash_of_content = "def456"
`

	err := os.WriteFile(lockPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	lockFile, err := readLockFile(lockPath)
	if err != nil {
		t.Errorf("readLockFile() error = %v, want nil", err)
	}

	if len(lockFile.Locks) != 1 {
		t.Fatalf("readLockFile() returned %d locks, want 1", len(lockFile.Locks))
	}

	// Check first lock
	if lockFile.Locks[0].URI != "https://example.org" {
		t.Errorf("Lock 0 URI = %s, want https://example.org", lockFile.Locks[0].URI)
	}
	if lockFile.Locks[0].HashOfContent != "def456" {
		t.Errorf("Lock 0 HashOfContent = %s, want def456", lockFile.Locks[0].HashOfContent)
	}
}

func TestWriteLockFile(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "check_links.lock")

	lockFile := &LockFile{
		Locks: []Lock{
			{
				URI:           "https://example.com",
				HashVersion:   "h1",
				HashOfContent: "test",
			},
		},
	}

	err := writeLockFile(lockPath, lockFile)
	if err != nil {
		t.Errorf("writeLockFile() error = %v, want nil", err)
	}

	// Read back and verify
	readBack, err := readLockFile(lockPath)
	if err != nil {
		t.Errorf("readLockFile() error = %v, want nil", err)
	}

	if len(readBack.Locks) != 1 {
		t.Fatalf("Read back %d locks, want 1", len(readBack.Locks))
	}

	if readBack.Locks[0].URI != "https://example.com" {
		t.Errorf("Read back URI = %s, want https://example.com", readBack.Locks[0].URI)
	}
}

func TestAddLockEntry(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "check_links.lock")

	// Add first entry - using a real URL that should be stable
	err := addLockEntry(lockPath, "https://example.com", false)
	if err != nil {
		t.Errorf("addLockEntry() error = %v, want nil", err)
	}

	// Verify first entry
	lockFile, err := readLockFile(lockPath)
	if err != nil {
		t.Errorf("readLockFile() error = %v, want nil", err)
	}
	if len(lockFile.Locks) != 1 {
		t.Fatalf("Expected 1 lock, got %d", len(lockFile.Locks))
	}
	if lockFile.Locks[0].URI != "https://example.com" {
		t.Errorf("Lock URI = %s, want https://example.com", lockFile.Locks[0].URI)
	}
	// Verify hash was computed
	if lockFile.Locks[0].HashOfContent == "" {
		t.Error("Lock HashOfContent is empty, expected hash to be computed")
	}
	if len(lockFile.Locks[0].HashOfContent) != 96 { // SHA384 hex string is 96 characters
		t.Errorf("Lock HashOfContent length = %d, want 96", len(lockFile.Locks[0].HashOfContent))
	}

	// Try to add duplicate
	err = addLockEntry(lockPath, "https://example.com", false)
	if err == nil {
		t.Error("addLockEntry() with duplicate should return error")
	}
}

func TestVerifyLockEntry(t *testing.T) {
	// Test with a valid lock entry for example.com
	// First fetch and compute the hash
	hash, err := fetchURLAndComputeSHA384("https://example.com")
	if err != nil {
		t.Fatalf("Failed to fetch and hash URL: %v", err)
	}

	// Create a lock entry with the correct hash
	lock := Lock{
		URI:           "https://example.com",
		HashVersion:   "h1",
		HashOfContent: hash,
	}

	// Verify should succeed
	err = verifyLockEntry(lock)
	if err != nil {
		t.Errorf("verifyLockEntry() error = %v, want nil", err)
	}

	// Test with incorrect hash
	lockBadHash := Lock{
		URI:           "https://example.com",
		HashVersion:   "h1",
		HashOfContent: "incorrect_hash",
	}

	err = verifyLockEntry(lockBadHash)
	if err == nil {
		t.Error("verifyLockEntry() with bad hash should return error")
	}

	// Test with unsupported hash version
	lockBadVersion := Lock{
		URI:           "https://example.com",
		HashVersion:   "h2",
		HashOfContent: hash,
	}

	err = verifyLockEntry(lockBadVersion)
	if err == nil {
		t.Error("verifyLockEntry() with unsupported hash version should return error")
	}
}

func TestVerifyLockFile(t *testing.T) {
	// Test with empty lock file
	lockFile := &LockFile{Locks: []Lock{}}
	errors := verifyLockFile(lockFile)
	if len(errors) != 0 {
		t.Errorf("verifyLockFile() with empty lock file returned %d errors, want 0", len(errors))
	}

	// Test with unsupported hash version (no network required)
	lockFileWithBadVersion := &LockFile{
		Locks: []Lock{
			{
				URI:           "https://example.com",
				HashVersion:   "h2",
				HashOfContent: "some_hash",
			},
		},
	}

	errors = verifyLockFile(lockFileWithBadVersion)
	if len(errors) != 1 {
		t.Errorf("verifyLockFile() with unsupported hash version returned %d errors, want 1", len(errors))
	}
	if len(errors) > 0 && !strings.Contains(errors[0].Error(), "unsupported hash version") {
		t.Errorf("verifyLockFile() error message = %v, want to contain 'unsupported hash version'", errors[0])
	}

	// Test with lock file containing invalid entries (requires network)
	lockFileWithBadEntries := &LockFile{
		Locks: []Lock{
			{
				URI:           "https://example.com",
				HashVersion:   "h1",
				HashOfContent: "bad_hash",
			},
			{
				URI:           "https://example.org",
				HashVersion:   "h1",
				HashOfContent: "another_bad_hash",
			},
		},
	}

	errors = verifyLockFile(lockFileWithBadEntries)
	if len(errors) != 2 {
		t.Errorf("verifyLockFile() with 2 bad entries returned %d errors, want 2", len(errors))
	}
}
