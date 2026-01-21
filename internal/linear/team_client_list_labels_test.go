package linear

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

// TestTeamClientListLabels tests the ListLabels functionality
func TestTeamClientListLabels(t *testing.T) {
	t.Run("list labels for a team", func(t *testing.T) {
		mockResponseBody := `{
			"data": {
				"team": {
					"labels": {
						"nodes": [
							{
								"id": "label-1",
								"name": "bug",
								"color": "#FF0000",
								"description": "Something isn't working"
							},
							{
								"id": "label-2",
								"name": "enhancement",
								"color": "#00FF00",
								"description": "New feature or request"
							},
							{
								"id": "label-3",
								"name": "documentation",
								"color": "#0000FF",
								"description": "Improvements or additions to documentation"
							}
						]
					}
				}
			}
		}`

		client := NewClient("test-token")
		client.base.httpClient = &http.Client{
			Transport: &mockTransport{
				response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(mockResponseBody)),
				},
			},
		}

		labels, err := client.Teams.ListLabels("team-123")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(labels) != 3 {
			t.Errorf("Expected 3 labels, got %d", len(labels))
		}

		// Check first label
		if labels[0].ID != "label-1" {
			t.Errorf("Expected first label ID to be 'label-1', got '%s'", labels[0].ID)
		}
		if labels[0].Name != "bug" {
			t.Errorf("Expected first label name to be 'bug', got '%s'", labels[0].Name)
		}
		if labels[0].Color != "#FF0000" {
			t.Errorf("Expected first label color to be '#FF0000', got '%s'", labels[0].Color)
		}
		if labels[0].Description != "Something isn't working" {
			t.Errorf("Expected first label description, got '%s'", labels[0].Description)
		}

		// Check second label
		if labels[1].Name != "enhancement" {
			t.Errorf("Expected second label name to be 'enhancement', got '%s'", labels[1].Name)
		}
		if labels[1].Color != "#00FF00" {
			t.Errorf("Expected second label color to be '#00FF00', got '%s'", labels[1].Color)
		}

		// Check third label
		if labels[2].Name != "documentation" {
			t.Errorf("Expected third label name to be 'documentation', got '%s'", labels[2].Name)
		}
	})

	t.Run("handle empty labels list", func(t *testing.T) {
		mockResponseBody := `{
			"data": {
				"team": {
					"labels": {
						"nodes": []
					}
				}
			}
		}`

		client := NewClient("test-token")
		client.base.httpClient = &http.Client{
			Transport: &mockTransport{
				response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(mockResponseBody)),
				},
			},
		}

		labels, err := client.Teams.ListLabels("team-123")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(labels) != 0 {
			t.Errorf("Expected 0 labels, got %d", len(labels))
		}
	})

	t.Run("handle team not found", func(t *testing.T) {
		mockResponseBody := `{
			"data": {
				"team": null
			}
		}`

		client := NewClient("test-token")
		client.base.httpClient = &http.Client{
			Transport: &mockTransport{
				response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(mockResponseBody)),
				},
			},
		}

		_, err := client.Teams.ListLabels("non-existent-team")
		if err == nil {
			t.Fatal("Expected error for non-existent team, got nil")
		}
		if err.Error() != "team not found" {
			t.Errorf("Expected 'team not found' error, got: %v", err)
		}
	})

	t.Run("handle GraphQL errors", func(t *testing.T) {
		mockResponseBody := `{
			"errors": [
				{
					"message": "Not authorized to access team"
				}
			]
		}`

		client := NewClient("test-token")
		client.base.httpClient = &http.Client{
			Transport: &mockTransport{
				response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(mockResponseBody)),
				},
			},
		}

		_, err := client.Teams.ListLabels("team-123")
		if err == nil {
			t.Fatal("Expected error, got nil")
		}

		if !contains(err.Error(), "Not authorized") {
			t.Errorf("Expected authorization error, got: %v", err)
		}
	})
}