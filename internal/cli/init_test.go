package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetDefaultProject_WithProject(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, ".linear.yaml"), []byte("team: TEC\nproject: project-eat\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	result := GetDefaultProject()
	if result != "project-eat" {
		t.Errorf("expected 'project-eat', got '%s'", result)
	}
}

func TestGetDefaultProject_WithoutProject(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, ".linear.yaml"), []byte("team: TEC\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	result := GetDefaultProject()
	if result != "" {
		t.Errorf("expected empty string, got '%s'", result)
	}
}

func TestGetDefaultProject_NoConfig(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	result := GetDefaultProject()
	if result != "" {
		t.Errorf("expected empty string, got '%s'", result)
	}
}

func TestLoadProjectConfig_WithProject(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, ".linear.yaml"), []byte("team: TEC\nproject: my-project\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	config, err := LoadProjectConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if config.Team != "TEC" {
		t.Errorf("expected team 'TEC', got '%s'", config.Team)
	}
	if config.Project != "my-project" {
		t.Errorf("expected project 'my-project', got '%s'", config.Project)
	}
}
