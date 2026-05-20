package commands

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"rosters/pkg/format"
	"rosters/pkg/git"
	"rosters/pkg/models"

	"github.com/spf13/cobra"
)

func setupGitRepo(t *testing.T, dir string) {
	if err := git.Run(dir, "init"); err != nil {
		t.Fatalf("git init failed: %v", err)
	}
	git.Run(dir, "config", "user.email", "test@example.com")
	git.Run(dir, "config", "user.name", "Test User")
}

func commitRosters(t *testing.T, dir string) {
	if err := git.Run(dir, "add", models.SeedsDirName); err != nil {
		t.Fatalf("git add failed: %v", err)
	}
	if err := git.Run(dir, "commit", "-m", "infra"); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}
}

func TestSyncCommand(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	rostersDir := filepath.Join(tmpDir, models.SeedsDirName)
	os.MkdirAll(rostersDir, 0755)
	os.WriteFile(filepath.Join(rostersDir, models.ConfigFile), []byte("project: test\nversion: \"1\""), 0644)

	commitRosters(t, tmpDir)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	t.Run("shows status with no changes", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().Bool("status", true, "")

		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := runSync(cmd, []string{})

		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = rescueStdout

		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(out), "No uncommitted .rosters/ changes.") {
			t.Errorf("unexpected output: %s", string(out))
		}
	})

	t.Run("handles dry-run with changes", func(t *testing.T) {
		os.WriteFile(filepath.Join(rostersDir, "issues.jsonl"), []byte("{}\n"), 0644)

		cmd := &cobra.Command{}
		cmd.Flags().Bool("dry-run", true, "")

		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := runSync(cmd, []string{})

		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = rescueStdout

		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(out), "Dry run - would commit:") {
			t.Errorf("unexpected output: %s", string(out))
		}
	})

	t.Run("json mode returns valid json", func(t *testing.T) {
		format.SetFormat("json")
		defer format.SetFormat("markdown")

		cmd := &cobra.Command{}
		cmd.Flags().Bool("status", true, "")

		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		runSync(cmd, []string{})

		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = rescueStdout

		var res map[string]any
		if err := json.Unmarshal(out, &res); err != nil {
			t.Fatalf("failed to parse json: %v", err)
		}
		if res["command"] != "sync" {
			t.Errorf("unexpected json result: %v", res)
		}
	})
}
