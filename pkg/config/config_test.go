package config

import (
	"os"
	"path/filepath"
	"testing"

	"rosters/pkg/models"
)

func TestProjectRootFromRostersDir(t *testing.T) {
	rostersDir := "/path/to/project/.rosters"
	root := ProjectRootFromRostersDir(rostersDir)
	expected := "/path/to/project"
	if root != expected {
		t.Errorf("ProjectRootFromRostersDir(%q) = %q, want %q", rostersDir, root, expected)
	}
}

func TestMaxPlanDepth(t *testing.T) {
	defaultDepth := 5
	tests := []struct {
		name   string
		config *models.Config
		want   int
	}{
		{
			name:   "config with nil depth",
			config: &models.Config{MaxPlanDepth: nil},
			want:   models.DefaultMaxPlanDepth,
		},
		{
			name:   "config with explicit depth",
			config: &models.Config{MaxPlanDepth: &defaultDepth},
			want:   5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MaxPlanDepth(tt.config); got != tt.want {
				t.Errorf("MaxPlanDepth() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestFindRostersDir_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	rostersDir := filepath.Join(tmpDir, models.SeedsDirName)
	configPath := filepath.Join(rostersDir, models.ConfigFile)
	if err := os.MkdirAll(rostersDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte("project: test"), 0600); err != nil {
		t.Fatal(err)
	}

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	found, err := FindRostersDir("")
	if err != nil {
		t.Fatalf("FindRostersDir() error: %v", err)
	}
	if found != rostersDir {
		t.Errorf("FindRostersDir() = %q, want %q", found, rostersDir)
	}
}

func TestFindRostersDir_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	_, err := FindRostersDir("")
	if err == nil {
		t.Error("FindRostersDir() expected error when no .rosters")
	}
	if !os.IsNotExist(err) {
		t.Errorf("expected not exist error, got %v", err)
	}
}

func TestReadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	rostersDir := filepath.Join(tmpDir, models.SeedsDirName)
	configPath := filepath.Join(rostersDir, models.ConfigFile)
	os.MkdirAll(rostersDir, 0755)
	content := `project: myproj
version: "1"
max_plan_depth: 10
`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := ReadConfig(rostersDir)
	if err != nil {
		t.Fatalf("ReadConfig error: %v", err)
	}
	if cfg.Project != "myproj" {
		t.Errorf("Project = %q, want myproj", cfg.Project)
	}
	if cfg.Version != "1" {
		t.Errorf("Version = %q, want 1", cfg.Version)
	}
	if cfg.MaxPlanDepth == nil || *cfg.MaxPlanDepth != 10 {
		t.Errorf("MaxPlanDepth = %v, want 10", cfg.MaxPlanDepth)
	}
}

func TestReadConfig_Missing(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := ReadConfig(tmpDir)
	if err == nil {
		t.Error("ReadConfig expected error for missing config")
	}
}

func TestLoadPlanTemplates_ReturnsBuiltinsOnMissing(t *testing.T) {
	tmpDir := t.TempDir()
	rostersDir := filepath.Join(tmpDir, models.SeedsDirName)
	os.MkdirAll(rostersDir, 0755)
	templates, err := LoadPlanTemplates(rostersDir)
	if err != nil {
		t.Fatalf("LoadPlanTemplates error: %v", err)
	}
	if len(templates) == 0 {
		t.Error("LoadPlanTemplates returned empty map")
	}
}
