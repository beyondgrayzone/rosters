package commands

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"rosters/pkg/format"
	"rosters/pkg/models"
	"rosters/pkg/store"
)

func TestDepCommand(t *testing.T) {
	tmpDir := t.TempDir()
	rostersDir := filepath.Join(tmpDir, ".rosters")
	os.MkdirAll(rostersDir, 0755)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	os.WriteFile(filepath.Join(rostersDir, "config.yaml"), []byte("project: test\nversion: \"1\""), 0644)

	issue1 := models.Issue{ID: "iss-a", Title: "Issue A", Status: "open", CreatedAt: "now", UpdatedAt: "now"}
	issue2 := models.Issue{ID: "iss-b", Title: "Issue B", Status: "open", CreatedAt: "now", UpdatedAt: "now"}
	store.WriteIssues(rostersDir, []models.Issue{issue1, issue2})

	t.Run("adds dependency", func(t *testing.T) {
		err := runDepAddRemove("iss-b", "iss-a", true)
		if err != nil {
			t.Fatal(err)
		}

		issues, _ := store.ReadIssues(rostersDir)
		var b models.Issue
		for _, i := range issues {
			if i.ID == "iss-b" {
				b = i
			}
		}

		found := false
		for _, dep := range b.BlockedBy {
			if dep == "iss-a" {
				found = true
			}
		}
		if !found {
			t.Error("dependency not added to BlockedBy")
		}
	})

	t.Run("lists dependencies", func(t *testing.T) {
		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := runDepList("iss-b")

		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = rescueStdout

		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(out), "Blocked by:") || !strings.Contains(string(out), "iss-a") {
			t.Errorf("output missing dependency info: %s", string(out))
		}
	})

	t.Run("removes dependency", func(t *testing.T) {
		err := runDepAddRemove("iss-b", "iss-a", false)
		if err != nil {
			t.Fatal(err)
		}

		issues, _ := store.ReadIssues(rostersDir)
		for _, i := range issues {
			if i.ID == "iss-b" {
				if len(i.BlockedBy) != 0 {
					t.Error("dependency not removed")
				}
			}
		}
	})

	t.Run("json mode returns valid json", func(t *testing.T) {
		format.SetFormat("json")
		defer format.SetFormat("markdown")

		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		runDepAddRemove("iss-b", "iss-a", true)

		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = rescueStdout

		var res map[string]any
		if err := json.Unmarshal(out, &res); err != nil {
			t.Fatalf("failed to parse json: %v", err)
		}
		if res["success"] != true || res["command"] != "dep add" {
			t.Errorf("unexpected json result: %v", res)
		}
	})
}
