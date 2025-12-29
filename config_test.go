package main

import (
	"os"
	"path/filepath"
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
	// This test requires network access, so we skip it in CI
	if os.Getenv("CI") != "" {
		t.Skip("Skipping test that requires network access in CI")
	}

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
