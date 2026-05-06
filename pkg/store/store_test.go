package store

import (
	"os"
	"path/filepath"
	"testing"

	"rosters/pkg/models"
)

func TestStore_Locking(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.jsonl")

	t.Run("acquires and releases lock", func(t *testing.T) {
		err := AcquireLock(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(path + ".lock"); os.IsNotExist(err) {
			t.Error("lock file should exist")
		}
		ReleaseLock(path)
		if _, err := os.Stat(path + ".lock"); err == nil {
			t.Error("lock file should be removed")
		}
	})

	t.Run("WithLock executes function", func(t *testing.T) {
		val, err := WithLock(path, func() (int, error) {
			return 42, nil
		})
		if err != nil || val != 42 {
			t.Errorf("WithLock failed: %v, %v", err, val)
		}
	})
}

func TestStore_Issues(t *testing.T) {
	tmpDir := t.TempDir()
	issue := models.Issue{ID: "test-1", Title: "Hello"}

	err := AppendIssue(tmpDir, issue)
	if err != nil {
		t.Fatal(err)
	}

	issues, err := ReadIssues(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(issues) != 1 || issues[0].ID != "test-1" {
		t.Errorf("unexpected issues: %v", issues)
	}
}
