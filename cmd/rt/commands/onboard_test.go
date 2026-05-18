package commands

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"rosters/pkg/format"
)

func TestOnboardCommand(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	rostersDir := filepath.Join(tmpDir, ".rosters")
	os.MkdirAll(rostersDir, 0755)
	os.WriteFile(filepath.Join(rostersDir, "config.yaml"), []byte("project: test\nversion: \"1\""), 0644)

	t.Run("creates AGENTS.md if missing", func(t *testing.T) {
		opts := &onboardOptions{}
		err := runOnboard(opts)
		if err != nil {
			t.Fatal(err)
		}

		claudePath := filepath.Join(tmpDir, "AGENTS.md")
		if _, err := os.Stat(claudePath); err != nil {
			t.Error("AGENTS.md was not created")
		}

		content, _ := os.ReadFile(claudePath)
		if !strings.Contains(string(content), startMarker) {
			t.Error("content missing start marker")
		}
	})

	t.Run("updates outdated section", func(t *testing.T) {
		claudePath := filepath.Join(tmpDir, "AGENTS.md")
		oldContent := startMarker + "\n" + legacyVersionMarkerPrefix + "1 -->\nold content\n" + endMarker
		os.WriteFile(claudePath, []byte(oldContent), 0644)

		opts := &onboardOptions{}
		runOnboard(opts)

		content, _ := os.ReadFile(claudePath)
		if strings.Contains(string(content), "old content") {
			t.Error("old content should have been replaced")
		}
		if !strings.Contains(string(content), getSchemaMarker("")) {
			t.Error("missing new schema marker")
		}
	})

	t.Run("check mode reports status", func(t *testing.T) {
		format.SetFormat("json")
		defer format.SetFormat("markdown")

		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		opts := &onboardOptions{check: true, json: true}
		runOnboard(opts)

		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = rescueStdout

		var res map[string]any
		json.Unmarshal(out, &res)
		if res["status"] != "current" {
			t.Errorf("expected status current, got %v", res["status"])
		}
	})

	t.Run("detects pi variant", func(t *testing.T) {
		piDir := filepath.Join(tmpDir, ".pi")
		os.MkdirAll(piDir, 0755)
		settings := `{"packages": ["@bgz/rosters-cli"]}`
		os.WriteFile(filepath.Join(piDir, "settings.json"), []byte(settings), 0644)

		opts := &onboardOptions{stdout: true}
		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		runOnboard(opts)

		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = rescueStdout

		if !strings.Contains(string(out), getSchemaMarker("pi")) {
			t.Error("pi variant marker missing from stdout")
		}
		if !strings.Contains(string(out), "pi-coding-agent extension") {
			t.Error("pi variant content missing from stdout")
		}
	})
}
