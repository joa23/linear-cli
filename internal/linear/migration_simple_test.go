package linear

import (
	"testing"
)

func TestMigrateCommentMetadataToDescriptions_Integration(t *testing.T) {
	t.Skip("Migration functionality has been removed - metadata is now stored in descriptions by default")
	/*
	// This test verifies that the migration function compiles and can be called
	// The full integration test would require actual Linear API access
	
	client := NewClient("test-token")
	
	// This function should exist and be callable (though it will fail with test token)
	err := client.MigrateCommentMetadataToDescriptions()
	
	// We expect an error since we're using a test token, but the function should exist
	if err == nil {
		t.Log("Migration function executed successfully (unexpected with test token)")
	} else {
		t.Logf("Migration function exists and returned expected error: %v", err)
	}
	*/
}

func TestMigrationHelperFunctions(t *testing.T) {
	t.Skip("Migration functionality has been removed - metadata is now stored in descriptions by default")
	/*
	client := NewClient("test-token")
	
	// Test that the helper functions exist and are callable
	t.Run("migrateIssueCommentMetadata", func(t *testing.T) {
		// This will fail but proves the function exists
		err := client.migrateIssueCommentMetadata()
		if err != nil {
			t.Logf("migrateIssueCommentMetadata exists and returned error: %v", err)
		}
	})
	
	t.Run("migrateProjectCommentMetadata", func(t *testing.T) {
		// This will fail but proves the function exists
		err := client.migrateProjectCommentMetadata()
		if err != nil {
			t.Logf("migrateProjectCommentMetadata exists and returned error: %v", err)
		}
	})
	
	t.Run("ListAllProjects", func(t *testing.T) {
		// This will fail but proves the function exists
		_, err := client.ListAllProjects()
		if err != nil {
			t.Logf("ListAllProjects exists and returned error: %v", err)
		}
	})
	
	t.Run("getProjectCommentMetadata", func(t *testing.T) {
		// This will fail but proves the function exists
		_, err := client.getProjectCommentMetadata("test-project-id")
		if err != nil {
			t.Logf("getProjectCommentMetadata exists and returned error: %v", err)
		}
	})
	
	t.Run("getIssueCommentMetadata", func(t *testing.T) {
		// This will fail but proves the function exists
		_, err := client.getIssueCommentMetadata("test-issue-id")
		if err != nil {
			t.Logf("getIssueCommentMetadata exists and returned error: %v", err)
		}
	})
	*/
}

func TestMigrationLogic_Metadata_Merging(t *testing.T) {
	// Test the core metadata merging logic without API calls
	
	tests := []struct {
		name                string
		existingMetadata    map[string]interface{}
		commentMetadata     map[string]interface{}
		expectedMerged      map[string]interface{}
	}{
		{
			name: "comment metadata takes precedence over existing",
			existingMetadata: map[string]interface{}{
				"priority": "low",
				"team":     "frontend",
			},
			commentMetadata: map[string]interface{}{
				"priority": "high",
				"status":   "urgent",
			},
			expectedMerged: map[string]interface{}{
				"priority": "high", // Comment metadata wins
				"team":     "frontend", // Existing preserved
				"status":   "urgent", // New from comment
			},
		},
		{
			name: "empty existing metadata",
			existingMetadata: map[string]interface{}{},
			commentMetadata: map[string]interface{}{
				"priority": "medium",
				"assignee": "john",
			},
			expectedMerged: map[string]interface{}{
				"priority": "medium",
				"assignee": "john",
			},
		},
		{
			name: "empty comment metadata",
			existingMetadata: map[string]interface{}{
				"priority": "low",
			},
			commentMetadata: map[string]interface{}{},
			expectedMerged: map[string]interface{}{
				"priority": "low",
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the merging logic from the migration function
			mergedMetadata := make(map[string]interface{})
			for k, v := range tt.existingMetadata {
				mergedMetadata[k] = v
			}
			for k, v := range tt.commentMetadata {
				mergedMetadata[k] = v
			}
			
			// Check that all expected keys are present with correct values
			for expectedKey, expectedValue := range tt.expectedMerged {
				if actualValue, exists := mergedMetadata[expectedKey]; !exists {
					t.Errorf("Expected key %s not found in merged metadata", expectedKey)
				} else if actualValue != expectedValue {
					t.Errorf("Key %s: expected %v, got %v", expectedKey, expectedValue, actualValue)
				}
			}
			
			// Check that no unexpected keys are present
			if len(mergedMetadata) != len(tt.expectedMerged) {
				t.Errorf("Merged metadata has %d keys, expected %d", len(mergedMetadata), len(tt.expectedMerged))
			}
		})
	}
}