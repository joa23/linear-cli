package taskwriter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriter_WriteTasks_Success(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	writer := NewWriter()
	tasks := []ClaudeTask{
		{
			ID:          "TEST-1",
			Subject:     "Fix bug",
			Description: "This is a bug fix",
			ActiveForm:  "Fixing bug",
			Status:      "pending",
			Blocks:      []string{},
			BlockedBy:   []string{"TEST-2"},
		},
		{
			ID:          "TEST-2",
			Subject:     "Add feature",
			Description: "This is a new feature",
			ActiveForm:  "Adding feature",
			Status:      "pending",
			Blocks:      []string{},
			BlockedBy:   []string{},
		},
	}

	err := writer.WriteTasks(tmpDir, tasks)
	if err != nil {
		t.Fatalf("WriteTasks failed: %v", err)
	}

	// Verify files were created
	file1 := filepath.Join(tmpDir, "TEST-1.json")
	file2 := filepath.Join(tmpDir, "TEST-2.json")

	if _, err := os.Stat(file1); os.IsNotExist(err) {
		t.Errorf("Expected file %s to exist", file1)
	}
	if _, err := os.Stat(file2); os.IsNotExist(err) {
		t.Errorf("Expected file %s to exist", file2)
	}

	// Verify JSON content
	data, err := os.ReadFile(file1)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", file1, err)
	}

	var task ClaudeTask
	if err := json.Unmarshal(data, &task); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if task.ID != "TEST-1" {
		t.Errorf("Expected ID 'TEST-1', got '%s'", task.ID)
	}
	if task.Subject != "Fix bug" {
		t.Errorf("Expected subject 'Fix bug', got '%s'", task.Subject)
	}
	if len(task.BlockedBy) != 1 || task.BlockedBy[0] != "TEST-2" {
		t.Errorf("Expected blockedBy ['TEST-2'], got %v", task.BlockedBy)
	}
}

func TestWriter_WriteTasks_EmptyFolder(t *testing.T) {
	writer := NewWriter()
	tasks := []ClaudeTask{
		{ID: "TEST-1", Subject: "Test", Description: "Test", ActiveForm: "Testing", Status: "pending", Blocks: []string{}, BlockedBy: []string{}},
	}

	err := writer.WriteTasks("", tasks)
	if err == nil {
		t.Error("Expected error for empty folder, got nil")
	}
}

func TestWriter_WriteTasks_CreatesFolderIfNotExists(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "nested", "folder")

	writer := NewWriter()
	tasks := []ClaudeTask{
		{ID: "TEST-1", Subject: "Test", Description: "Test", ActiveForm: "Testing", Status: "pending", Blocks: []string{}, BlockedBy: []string{}},
	}

	err := writer.WriteTasks(subDir, tasks)
	if err != nil {
		t.Fatalf("WriteTasks failed: %v", err)
	}

	// Verify nested folder was created
	if _, err := os.Stat(subDir); os.IsNotExist(err) {
		t.Errorf("Expected folder %s to be created", subDir)
	}

	// Verify file exists
	file := filepath.Join(subDir, "TEST-1.json")
	if _, err := os.Stat(file); os.IsNotExist(err) {
		t.Errorf("Expected file %s to exist", file)
	}
}

func TestWriter_WriteTasks_ValidJSONFormat(t *testing.T) {
	tmpDir := t.TempDir()

	writer := NewWriter()
	tasks := []ClaudeTask{
		{
			ID:          "TEST-123",
			Subject:     "Implement OAuth",
			Description: "Add OAuth authentication support",
			ActiveForm:  "Implementing OAuth",
			Status:      "pending",
			Blocks:      []string{},
			BlockedBy:   []string{"TEST-124", "TEST-125"},
		},
	}

	err := writer.WriteTasks(tmpDir, tasks)
	if err != nil {
		t.Fatalf("WriteTasks failed: %v", err)
	}

	// Read and verify JSON structure
	data, err := os.ReadFile(filepath.Join(tmpDir, "TEST-123.json"))
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	var task ClaudeTask
	if err := json.Unmarshal(data, &task); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	// Verify all fields
	if task.ID != "TEST-123" {
		t.Errorf("ID mismatch")
	}
	if task.Subject != "Implement OAuth" {
		t.Errorf("Subject mismatch")
	}
	if task.Description != "Add OAuth authentication support" {
		t.Errorf("Description mismatch")
	}
	if task.ActiveForm != "Implementing OAuth" {
		t.Errorf("ActiveForm mismatch")
	}
	if task.Status != "pending" {
		t.Errorf("Status mismatch")
	}
	if len(task.BlockedBy) != 2 {
		t.Errorf("Expected 2 blockers, got %d", len(task.BlockedBy))
	}
}

func TestWriter_WriteTasks_OverwritesExistingFiles(t *testing.T) {
	tmpDir := t.TempDir()

	writer := NewWriter()

	// Write initial task
	tasks1 := []ClaudeTask{
		{ID: "TEST-1", Subject: "Original", Description: "Original desc", ActiveForm: "Working", Status: "pending", Blocks: []string{}, BlockedBy: []string{}},
	}
	err := writer.WriteTasks(tmpDir, tasks1)
	if err != nil {
		t.Fatalf("First write failed: %v", err)
	}

	// Overwrite with updated task
	tasks2 := []ClaudeTask{
		{ID: "TEST-1", Subject: "Updated", Description: "Updated desc", ActiveForm: "Working", Status: "pending", Blocks: []string{}, BlockedBy: []string{}},
	}
	err = writer.WriteTasks(tmpDir, tasks2)
	if err != nil {
		t.Fatalf("Second write failed: %v", err)
	}

	// Verify file was overwritten
	data, err := os.ReadFile(filepath.Join(tmpDir, "TEST-1.json"))
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	var task ClaudeTask
	if err := json.Unmarshal(data, &task); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if task.Subject != "Updated" {
		t.Errorf("Expected 'Updated', got '%s'", task.Subject)
	}
	if task.Description != "Updated desc" {
		t.Errorf("Expected 'Updated desc', got '%s'", task.Description)
	}
}

func TestClaudeTask_JSONMarshaling(t *testing.T) {
	task := ClaudeTask{
		ID:          "TEST-1",
		Subject:     "Test Task",
		Description: "Test Description",
		ActiveForm:  "Testing Task",
		Status:      "pending",
		Blocks:      []string{"TEST-2"},
		BlockedBy:   []string{"TEST-3", "TEST-4"},
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal back
	var unmarshaled ClaudeTask
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify round-trip
	if unmarshaled.ID != task.ID {
		t.Error("ID mismatch after round-trip")
	}
	if unmarshaled.Subject != task.Subject {
		t.Error("Subject mismatch after round-trip")
	}
	if len(unmarshaled.Blocks) != len(task.Blocks) {
		t.Error("Blocks length mismatch after round-trip")
	}
	if len(unmarshaled.BlockedBy) != len(task.BlockedBy) {
		t.Error("BlockedBy length mismatch after round-trip")
	}
}
