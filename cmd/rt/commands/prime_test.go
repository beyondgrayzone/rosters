package commands

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"rosters/pkg/models"

	"github.com/spf13/cobra"
)

func TestPrimeCommand(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	t.Run("outputs full template by default", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().Bool("compact", false, "")
		cmd.Flags().Bool("export", false, "")
		cmd.Flags().Bool("json", false, "")

		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		runPrime(cmd, []string{})

		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = rescueStdout

		if !strings.Contains(string(out), "# Rosters Workflow Context") {
			t.Errorf("expected full title, got: %s", string(out))
		}
		if !strings.Contains(string(out), "Session Close Protocol") {
			t.Error("missing close protocol")
		}
	})

	t.Run("outputs compact template with flag", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().Bool("compact", true, "")
		cmd.Flags().Bool("export", false, "")
		cmd.Flags().Bool("json", false, "")

		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		runPrime(cmd, []string{})

		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = rescueStdout

		if !strings.Contains(string(out), "# Rosters Quick Reference") {
			t.Errorf("expected compact title, got: %s", string(out))
		}
	})

	t.Run("json mode returns structured data", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().Bool("compact", false, "")
		cmd.Flags().Bool("export", false, "")
		cmd.Flags().Bool("json", true, "")

		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		runPrime(cmd, []string{})

		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = rescueStdout

		var res map[string]any
		if err := json.Unmarshal(out, &res); err != nil {
			t.Fatalf("failed to parse json: %v", err)
		}
		if res["success"] != true || res["command"] != "prime" {
			t.Errorf("unexpected json structure: %v", res)
		}
		if res["sections"] == nil {
			t.Error("missing sections in json output")
		}
	})

	t.Run("uses custom PRIME.md if it exists", func(t *testing.T) {
		rostersDir := filepath.Join(tmpDir, models.SeedsDirName)
		os.MkdirAll(rostersDir, 0755)
		os.WriteFile(filepath.Join(rostersDir, models.ConfigFile), []byte("project: test"), 0644)
		os.WriteFile(filepath.Join(rostersDir, "PRIME.md"), []byte("# Custom Context"), 0644)

		cmd := &cobra.Command{}
		cmd.Flags().Bool("compact", false, "")
		cmd.Flags().Bool("export", false, "")
		cmd.Flags().Bool("json", false, "")

		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		runPrime(cmd, []string{})

		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = rescueStdout

		if string(out) != "# Custom Context" {
			t.Errorf("expected custom content, got: %q", string(out))
		}
	})

	t.Run("ignores custom PRIME.md with --export", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().Bool("compact", false, "")
		cmd.Flags().Bool("export", true, "")
		cmd.Flags().Bool("json", false, "")

		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		runPrime(cmd, []string{})

		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = rescueStdout

		if !strings.Contains(string(out), "# Rosters Workflow Context") {
			t.Error("export should return default template even if custom exists")
		}
	})
}
