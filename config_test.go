package main

import (
	"os"
	"path/filepath"
	"reflect"
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
  uri = "https://example.com"
  [locks.lock]
    include = ["test"]

[[locks]]
  uri = "https://example.org"
  [locks.lock]
    include = ["foo", "bar"]
    sha384 = "def456"
`
	
	err := os.WriteFile(lockPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	lockFile, err := readLockFile(lockPath)
	if err != nil {
		t.Errorf("readLockFile() error = %v, want nil", err)
	}
	
	if len(lockFile.Locks) != 2 {
		t.Fatalf("readLockFile() returned %d locks, want 2", len(lockFile.Locks))
	}
	
	// Check first lock
	if lockFile.Locks[0].URI != "https://example.com" {
		t.Errorf("Lock 0 URI = %s, want https://example.com", lockFile.Locks[0].URI)
	}
	expectedInclude := []string{"test"}
	if !reflect.DeepEqual(lockFile.Locks[0].Lock.Include, expectedInclude) {
		t.Errorf("Lock 0 Include = %v, want %v", lockFile.Locks[0].Lock.Include, expectedInclude)
	}
	if lockFile.Locks[0].Lock.SHA384 != "" {
		t.Errorf("Lock 0 SHA384 = %v, want empty", lockFile.Locks[0].Lock.SHA384)
	}
	
	// Check second lock
	if lockFile.Locks[1].URI != "https://example.org" {
		t.Errorf("Lock 1 URI = %s, want https://example.org", lockFile.Locks[1].URI)
	}
	expectedInclude2 := []string{"foo", "bar"}
	if !reflect.DeepEqual(lockFile.Locks[1].Lock.Include, expectedInclude2) {
		t.Errorf("Lock 1 Include = %v, want %v", lockFile.Locks[1].Lock.Include, expectedInclude2)
	}
	if lockFile.Locks[1].Lock.SHA384 != "def456" {
		t.Errorf("Lock 1 SHA384 = %s, want def456", lockFile.Locks[1].Lock.SHA384)
	}
}

func TestWriteLockFile(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "check_links.lock")
	
	lockFile := &LockFile{
		Locks: []Lock{
			{
				URI: "https://example.com",
				Lock: LockDetail{
					Include: []string{"test"},
				},
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
	err := addLockEntry(lockPath, "https://example.com")
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
	if lockFile.Locks[0].Lock.SHA384 == "" {
		t.Error("Lock SHA384 is empty, expected hash to be computed")
	}
	if len(lockFile.Locks[0].Lock.SHA384) != 96 { // SHA384 hex string is 96 characters
		t.Errorf("Lock SHA384 length = %d, want 96", len(lockFile.Locks[0].Lock.SHA384))
	}
	
	// Try to add duplicate
	err = addLockEntry(lockPath, "https://example.com")
	if err == nil {
		t.Error("addLockEntry() with duplicate should return error")
	}
}
