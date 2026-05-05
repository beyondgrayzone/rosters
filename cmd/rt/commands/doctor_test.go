package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"rosters/pkg/models"
)

func TestDoctorChecks(t *testing.T) {
	tmpDir := t.TempDir()
	rostersDir := filepath.Join(tmpDir, models.SeedsDirName)
	os.MkdirAll(rostersDir, 0755)

	t.Run("checkConfig fails on missing config", func(t *testing.T) {
		res := checkConfig(rostersDir, nil, os.ErrNotExist)
		if res.Status != "fail" {
			t.Errorf("expected fail, got %s", res.Status)
		}
	})

	t.Run("checkJsonlIntegrity detects bad lines", func(t *testing.T) {
		path := filepath.Join(rostersDir, models.IssuesFile)
		os.WriteFile(path, []byte(`{"id":"ok"}`+"\n"+`{"id":bad`), 0644)
		res := checkJsonlIntegrity(rostersDir)
		if res.Status != "fail" {
			t.Errorf("expected fail, got %s", res.Status)
		}
		if len(res.Details) != 1 {
			t.Errorf("expected 1 detail, got %d", len(res.Details))
		}
	})

	t.Run("checkDuplicateIDs detects repeats", func(t *testing.T) {
		path := filepath.Join(rostersDir, models.IssuesFile)
		os.WriteFile(path, []byte(`{"id":"A"}`+"\n"+`{"id":"A"}`), 0644)
		res := checkDuplicateIDs(rostersDir)
		if res.Status != "warn" {
			t.Errorf("expected warn, got %s", res.Status)
		}
	})

	t.Run("checkStaleLocks identifies old locks", func(t *testing.T) {
		lockPath := filepath.Join(rostersDir, models.IssuesFile+".lock")
		os.WriteFile(lockPath, []byte(""), 0644)
		oldTime := time.Now().Add(-1 * time.Hour)
		os.Chtimes(lockPath, oldTime, oldTime)

		res := checkStaleLocks(rostersDir)
		if res.Status != "warn" {
			t.Errorf("expected warn, got %s", res.Status)
		}
	})
}

func TestDoctorFixes(t *testing.T) {
	tmpDir := t.TempDir()
	rostersDir := filepath.Join(tmpDir, models.SeedsDirName)
	os.MkdirAll(rostersDir, 0755)

	t.Run("fixes duplicate lines", func(t *testing.T) {
		path := filepath.Join(rostersDir, models.IssuesFile)
		os.WriteFile(path, []byte(`{"id":"A","title":"v1"}`+"\n"+`{"id":"A","title":"v2"}`), 0644)

		checks := []doctorCheck{{Name: "duplicate-ids", Status: "warn", Fixable: true}}
		fixed := applyDoctorFixes(rostersDir, checks)

		if len(fixed) == 0 {
			t.Fatal("expected items to be fixed")
		}

		content, _ := os.ReadFile(path)
		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		if len(lines) != 1 {
			t.Errorf("expected 1 line after dedup, got %d", len(lines))
		}
		if !strings.Contains(lines[0], "v2") {
			t.Error("expected last-write-wins (v2) to remain")
		}
	})
}
