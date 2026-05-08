package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

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

func TestStore_WriteIssues(t *testing.T) {
	tmpDir := t.TempDir()
	issues := []models.Issue{{ID: "a", Title: "A"}, {ID: "b", Title: "B"}}

	if err := WriteIssues(tmpDir, issues); err != nil {
		t.Fatal(err)
	}

	read, _ := ReadIssues(tmpDir)
	if len(read) != 2 || read[1].ID != "b" {
		t.Errorf("WriteIssues failed, got %v", read)
	}
}

func TestStore_Plans(t *testing.T) {
	tmpDir := t.TempDir()
	plan := models.Plan{ID: "pl-1", Roster: "test-1", Status: models.PlanStatusApproved}

	path := PlansPath(tmpDir)
	data, _ := json.Marshal(plan)
	os.WriteFile(path, append(data, '\n'), 0644)

	plans, err := ReadPlans(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(plans) != 1 || plans[0].ID != "pl-1" {
		t.Errorf("unexpected plans: %v", plans)
	}
}

func TestStore_Deduplication(t *testing.T) {
	tmpDir := t.TempDir()
	path := IssuesPath(tmpDir)

	i1 := models.Issue{ID: "test-1", Title: "Version 1"}
	i2 := models.Issue{ID: "test-1", Title: "Version 2"}

	d1, _ := json.Marshal(i1)
	d2, _ := json.Marshal(i2)
	os.WriteFile(path, append(append(d1, '\n'), append(d2, '\n')...), 0644)

	issues, err := ReadIssues(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(issues) != 1 {
		t.Fatalf("expected 1 issue after deduplication, got %d", len(issues))
	}
	if issues[0].Title != "Version 2" {
		t.Errorf("expected last entry to win, got title: %s", issues[0].Title)
	}
}

func TestStore_StaleLockCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "stale.jsonl")
	lockPath := path + ".lock"

	f, _ := os.Create(lockPath)
	f.Close()

	staleTime := time.Now().Add(-time.Duration(models.LockStaleMS+1000) * time.Millisecond)
	os.Chtimes(lockPath, staleTime, staleTime)

	err := AcquireLock(path)
	if err != nil {
		t.Fatalf("AcquireLock failed to handle stale lock: %v", err)
	}
	defer ReleaseLock(path)

	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Error("lock file should exist after being acquired")
	}
}
