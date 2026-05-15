package commands

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"rosters/pkg/format"
	"rosters/pkg/store"
)

func TestTplCommands(t *testing.T) {
	tmpDir := t.TempDir()
	rostersDir := filepath.Join(tmpDir, ".rosters")
	os.MkdirAll(rostersDir, 0755)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	os.WriteFile(filepath.Join(rostersDir, "config.yaml"), []byte("project: tst\nversion: \"1\""), 0644)

	t.Run("tpl create", func(t *testing.T) {
		cmd := tplCreateCmd()
		cmd.SetArgs([]string{"--name", "Test Template"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}

		templates, _ := store.ReadTemplates(rostersDir)
		if len(templates) != 1 || templates[0].Name != "Test Template" {
			t.Errorf("expected 1 template, got %d", len(templates))
		}
	})

	t.Run("tpl step add", func(t *testing.T) {
		templates, _ := store.ReadTemplates(rostersDir)
		id := templates[0].ID

		cmd := tplStepAddCmd()
		cmd.SetArgs([]string{id, "--title", "{prefix} step one"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}

		templates, _ = store.ReadTemplates(rostersDir)
		if len(templates[0].Steps) != 1 || templates[0].Steps[0].Title != "{prefix} step one" {
			t.Errorf("expected 1 step, got %d", len(templates[0].Steps))
		}
	})

	t.Run("tpl pour", func(t *testing.T) {
		templates, _ := store.ReadTemplates(rostersDir)
		id := templates[0].ID

		cmd := tplPourCmd()
		cmd.SetArgs([]string{id, "--prefix", "PROJ"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}

		issues, _ := store.ReadIssues(rostersDir)
		if len(issues) != 1 || issues[0].Title != "PROJ step one" {
			t.Errorf("expected 1 issue, got %d", len(issues))
		}
		if *issues[0].Convoy != id {
			t.Errorf("convoy id mismatch: %s", *issues[0].Convoy)
		}
	})

	t.Run("tpl list", func(t *testing.T) {
		format.SetFormat("json")
		defer format.SetFormat("markdown")

		cmd := tplListCmd()
		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		cmd.Execute()

		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = rescueStdout

		var res map[string]any
		json.Unmarshal(out, &res)
		if res["count"].(float64) != 1 {
			t.Errorf("expected 1 template in json, got %v", res["count"])
		}
	})

	t.Run("tpl status", func(t *testing.T) {
		templates, _ := store.ReadTemplates(rostersDir)
		id := templates[0].ID

		cmd := tplStatusCmd()
		cmd.SetArgs([]string{id})

		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		cmd.Execute()

		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = rescueStdout

		if !strings.Contains(string(out), "Total:       1") {
			t.Errorf("status output incorrect: %s", string(out))
		}
	})
}
