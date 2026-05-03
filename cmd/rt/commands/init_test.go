package commands

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"rosters/pkg/format"

	"github.com/spf13/cobra"
)

func TestInitCommand(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	cmd := &cobra.Command{}
	cmd.Flags().Bool("json", false, "")

	t.Run("creates .rosters directory and files", func(t *testing.T) {
		err := runInit(cmd, []string{})
		if err != nil {
			t.Fatalf("runInit failed: %v", err)
		}

		rostersDir := filepath.Join(tmpDir, ".rosters")
		paths := []string{
			filepath.Join(rostersDir, "config.yaml"),
			filepath.Join(rostersDir, "issues.jsonl"),
			filepath.Join(rostersDir, "templates.jsonl"),
			filepath.Join(rostersDir, "plans.jsonl"),
			filepath.Join(rostersDir, ".gitignore"),
			filepath.Join(tmpDir, ".gitattributes"),
		}

		for _, p := range paths {
			if _, err := os.Stat(p); os.IsNotExist(err) {
				t.Errorf("expected file %s to exist", p)
			}
		}

		configBody, _ := os.ReadFile(filepath.Join(rostersDir, "config.yaml"))
		expectedProject := filepath.Base(tmpDir)
		if !strings.Contains(string(configBody), "project: \""+expectedProject+"\"") {
			t.Errorf("config.yaml missing project name: %s", string(configBody))
		}
	})

	t.Run("is idempotent", func(t *testing.T) {
		err := runInit(cmd, []string{})
		if err != nil {
			t.Errorf("second runInit failed: %v", err)
		}
	})

	t.Run("appends to gitattributes", func(t *testing.T) {
		gitattrsPath := filepath.Join(tmpDir, ".gitattributes")
		os.WriteFile(gitattrsPath, []byte("*.txt text\n"), 0644)

		err := os.RemoveAll(filepath.Join(tmpDir, ".rosters"))
		if err != nil {
			t.Fatal(err)
		}

		runInit(cmd, []string{})

		content, _ := os.ReadFile(gitattrsPath)
		if !strings.Contains(string(content), "*.txt text") {
			t.Error("existing gitattributes content lost")
		}
		if !strings.Contains(string(content), ".rosters/issues.jsonl merge=union") {
			t.Error("new gitattributes entry not appended")
		}
	})

	t.Run("json flag returns success json", func(t *testing.T) {
		format.SetFormat("json")
		defer format.SetFormat("markdown")

		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		runInit(cmd, []string{})

		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = rescueStdout

		var res map[string]any
		if err := json.Unmarshal(out, &res); err != nil {
			t.Fatalf("failed to parse json output: %v", err)
		}
		if res["success"] != true || res["command"] != "init" {
			t.Errorf("unexpected json result: %v", res)
		}
	})
}
