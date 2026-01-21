package linear

import (
	"reflect"
	"testing"
)

func TestExtractMetadataFromDescription(t *testing.T) {
	tests := []struct {
		name                 string
		description          string
		expectedMetadata     map[string]interface{}
		expectedCleanDesc    string
		expectedError        bool
	}{
		{
			name: "description with metadata section",
			description: "This is a regular description.\n\n<details><summary>🤖 Metadata</summary>\n\n```json\n{\n  \"priority\": \"high\",\n  \"category\": \"bug\",\n  \"estimate\": 4\n}\n```\n</details>",
			expectedMetadata: map[string]interface{}{
				"priority": "high",
				"category": "bug",
				"estimate": float64(4), // JSON numbers are parsed as float64
			},
			expectedCleanDesc: "This is a regular description.",
			expectedError:     false,
		},
		{
			name: "description without metadata",
			description: `This is a regular description without any metadata.

Some more content here.`,
			expectedMetadata:  map[string]interface{}{},
			expectedCleanDesc: `This is a regular description without any metadata.

Some more content here.`,
			expectedError: false,
		},
		{
			name: "description with empty metadata",
			description: "Description with empty metadata.\n\n<details><summary>🤖 Metadata</summary>\n\n```json\n{}\n```\n</details>",
			expectedMetadata:  map[string]interface{}{},
			expectedCleanDesc: "Description with empty metadata.",
			expectedError:     false,
		},
		{
			name: "description with invalid JSON in metadata",
			description: "Description with invalid JSON.\n\n<details><summary>🤖 Metadata</summary>\n\n```json\n{invalid json here\n```\n</details>",
			expectedMetadata:  map[string]interface{}{},
			expectedCleanDesc: "Description with invalid JSON.",
			expectedError:     true,
		},
		{
			name: "description with metadata in middle",
			description: "First part of description.\n\n<details><summary>🤖 Metadata</summary>\n\n```json\n{\"key\": \"value\"}\n```\n</details>\n\nSecond part of description.",
			expectedMetadata: map[string]interface{}{
				"key": "value",
			},
			expectedCleanDesc: "First part of description.\n\nSecond part of description.",
			expectedError: false,
		},
		{
			name:              "empty description",
			description:       "",
			expectedMetadata:  map[string]interface{}{},
			expectedCleanDesc: "",
			expectedError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata, cleanDesc := extractMetadataFromDescription(tt.description)
			
			// The new implementation doesn't return errors, it just returns nil metadata
			// if parsing fails, so we adjust the test accordingly
			
			if !reflect.DeepEqual(metadata, tt.expectedMetadata) {
				t.Errorf("metadata mismatch:\nexpected: %+v\ngot: %+v", tt.expectedMetadata, metadata)
			}
			
			if cleanDesc != tt.expectedCleanDesc {
				t.Errorf("clean description mismatch:\nexpected: %q\ngot: %q", tt.expectedCleanDesc, cleanDesc)
			}
		})
	}
}

func TestInjectMetadataIntoDescription(t *testing.T) {
	tests := []struct {
		name               string
		description        string
		metadata           map[string]interface{}
		expectedResult     string
	}{
		{
			name:        "inject metadata into empty description",
			description: "",
			metadata: map[string]interface{}{
				"priority": "high",
				"category": "bug",
			},
			expectedResult: "<details><summary>🤖 Metadata</summary>\n\n```json\n{\n  \"category\": \"bug\",\n  \"priority\": \"high\"\n}\n```\n</details>",
		},
		{
			name:        "inject metadata into description with content",
			description: "This is a task description.",
			metadata: map[string]interface{}{
				"status": "in-progress",
				"assignee": "john.doe",
			},
			expectedResult: "This is a task description.\n\n<details><summary>🤖 Metadata</summary>\n\n```json\n{\n  \"assignee\": \"john.doe\",\n  \"status\": \"in-progress\"\n}\n```\n</details>",
		},
		{
			name:        "inject empty metadata",
			description: "Task with no metadata.",
			metadata:    map[string]interface{}{},
			expectedResult: "Task with no metadata.",
		},
		{
			name:        "inject complex metadata",
			description: "Complex task.",
			metadata: map[string]interface{}{
				"tags":     []string{"urgent", "backend"},
				"estimate": 8,
				"details": map[string]interface{}{
					"component": "api",
					"version":   "2.0",
				},
			},
			expectedResult: "Complex task.\n\n<details><summary>🤖 Metadata</summary>\n\n```json\n{\n  \"details\": {\n    \"component\": \"api\",\n    \"version\": \"2.0\"\n  },\n  \"estimate\": 8,\n  \"tags\": [\n    \"urgent\",\n    \"backend\"\n  ]\n}\n```\n</details>",
		},
		{
			name:        "replace existing metadata",
			description: "Task description.\n\n<details><summary>🤖 Metadata</summary>\n\n```json\n{\"old\": \"data\"}\n```\n</details>",
			metadata: map[string]interface{}{
				"new": "data",
			},
			expectedResult: "Task description.\n\n<details><summary>🤖 Metadata</summary>\n\n```json\n{\n  \"new\": \"data\"\n}\n```\n</details>",
		},
		{
			name:        "inject metadata with special characters",
			description: "Task with special chars.",
			metadata: map[string]interface{}{
				"note": "This has \"quotes\" and \nnewlines",
				"path": "/home/user/file.txt",
			},
			expectedResult: "Task with special chars.\n\n<details><summary>🤖 Metadata</summary>\n\n```json\n{\n  \"note\": \"This has \\\"quotes\\\" and \\nnewlines\",\n  \"path\": \"/home/user/file.txt\"\n}\n```\n</details>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := injectMetadataIntoDescription(tt.description, tt.metadata)
			
			if result != tt.expectedResult {
				t.Errorf("result mismatch:\nexpected:\n%s\ngot:\n%s", tt.expectedResult, result)
			}
		})
	}
}

func TestUpdateDescriptionPreservingMetadata(t *testing.T) {
	tests := []struct {
		name           string
		oldDescription string
		newDescription string
		expectedResult string
	}{
		{
			name:           "preserve metadata when updating description",
			oldDescription: "Old task description.\n\n<details><summary>🤖 Metadata</summary>\n\n```json\n{\n  \"priority\": \"high\",\n  \"tags\": [\"bug\", \"urgent\"]\n}\n```\n</details>",
			newDescription: "Updated task description with more details.",
			expectedResult: "Updated task description with more details.\n\n<details><summary>🤖 Metadata</summary>\n\n```json\n{\n  \"priority\": \"high\",\n  \"tags\": [\n    \"bug\",\n    \"urgent\"\n  ]\n}\n```\n</details>",
		},
		{
			name:           "add metadata to new description when old has none",
			oldDescription: "Old description without metadata.",
			newDescription: "New description text.",
			expectedResult: "New description text.",
		},
		{
			name:           "handle empty new description with metadata",
			oldDescription: "Old text.\n\n<details><summary>🤖 Metadata</summary>\n\n```json\n{\"key\": \"value\"}\n```\n</details>",
			newDescription: "",
			expectedResult: "<details><summary>🤖 Metadata</summary>\n\n```json\n{\n  \"key\": \"value\"\n}\n```\n</details>",
		},
		{
			name:           "preserve complex metadata structure",
			oldDescription: "Task.\n\n<details><summary>🤖 Metadata</summary>\n\n```json\n{\n  \"assignee\": \"john@example.com\",\n  \"due_date\": \"2024-12-31\",\n  \"custom_fields\": {\n    \"department\": \"engineering\",\n    \"sprint\": 5\n  }\n}\n```\n</details>",
			newDescription: "Completely rewritten task description.",
			expectedResult: "Completely rewritten task description.\n\n<details><summary>🤖 Metadata</summary>\n\n```json\n{\n  \"assignee\": \"john@example.com\",\n  \"custom_fields\": {\n    \"department\": \"engineering\",\n    \"sprint\": 5\n  },\n  \"due_date\": \"2024-12-31\"\n}\n```\n</details>",
		},
		{
			name:           "handle new description that already has different metadata",
			oldDescription: "Old.\n\n<details><summary>🤖 Metadata</summary>\n\n```json\n{\"version\": 1}\n```\n</details>",
			newDescription: "New.\n\n<details><summary>🤖 Metadata</summary>\n\n```json\n{\"version\": 2}\n```\n</details>",
			expectedResult: "New.\n\n<details><summary>🤖 Metadata</summary>\n\n```json\n{\n  \"version\": 1\n}\n```\n</details>",
		},
		{
			name:           "both descriptions empty",
			oldDescription: "",
			newDescription: "",
			expectedResult: "",
		},
		{
			name:           "preserve metadata with special characters",
			oldDescription: "Text.\n\n<details><summary>🤖 Metadata</summary>\n\n```json\n{\n  \"note\": \"Has \\\"quotes\\\" and\\nnewlines\",\n  \"path\": \"/usr/local/bin\"\n}\n```\n</details>",
			newDescription: "New description content.",
			expectedResult: "New description content.\n\n<details><summary>🤖 Metadata</summary>\n\n```json\n{\n  \"note\": \"Has \\\"quotes\\\" and\\nnewlines\",\n  \"path\": \"/usr/local/bin\"\n}\n```\n</details>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := updateDescriptionPreservingMetadata(tt.oldDescription, tt.newDescription)
			
			if result != tt.expectedResult {
				t.Errorf("result mismatch:\nexpected:\n%s\ngot:\n%s", tt.expectedResult, result)
			}
		})
	}
}