package testutil

import (
	"testing"
)

func TestNewLinearMockServer(t *testing.T) {
	ts := NewLinearMockServer(t)
	if ts == nil {
		t.Fatal("Expected non-nil mock server")
	}
	if ts.URL == "" {
		t.Fatal("Expected non-empty URL")
	}
}

func TestNewSuccessTransport(t *testing.T) {
	// Test creating a success transport with string response
	response := `{"data": {"viewer": {"id": "123"}}}`
	transport := NewSuccessTransport(response)
	if transport == nil {
		t.Fatal("Expected non-nil transport")
	}
	if transport.Response.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", transport.Response.StatusCode)
	}
}

func TestGraphQLResponseHelpers(t *testing.T) {
	t.Run("data response", func(t *testing.T) {
		data := map[string]string{"foo": "bar"}
		resp := NewGraphQLDataResponse(data)
		if resp.Data == nil {
			t.Error("Expected non-nil data")
		}
		if len(resp.Errors) != 0 {
			t.Error("Expected no errors")
		}
	})

	t.Run("error response", func(t *testing.T) {
		resp := NewGraphQLErrorResponse("something went wrong", "TEST_ERROR")
		if resp.Data != nil {
			t.Error("Expected nil data")
		}
		if len(resp.Errors) != 1 {
			t.Error("Expected one error")
		}
		if resp.Errors[0].Message != "something went wrong" {
			t.Errorf("Expected error message 'something went wrong', got %s", resp.Errors[0].Message)
		}
	})
}

func TestTestIssueHelpers(t *testing.T) {
	issue := NewTestIssue("id-123", "TEST-456", "My Test Issue")
	if issue.ID != "id-123" {
		t.Errorf("Expected ID 'id-123', got %s", issue.ID)
	}
	if issue.Identifier != "TEST-456" {
		t.Errorf("Expected Identifier 'TEST-456', got %s", issue.Identifier)
	}
	if issue.Title != "My Test Issue" {
		t.Errorf("Expected Title 'My Test Issue', got %s", issue.Title)
	}
	if issue.Team == nil || issue.Team.Key != "TEST" {
		t.Error("Expected team with key 'TEST'")
	}
}

func TestNewTestTeam(t *testing.T) {
	team := NewTestTeam("team-id", "Engineering", "ENG")
	if team.ID != "team-id" {
		t.Errorf("Expected ID 'team-id', got %s", team.ID)
	}
	if team.Name != "Engineering" {
		t.Errorf("Expected Name 'Engineering', got %s", team.Name)
	}
	if team.Key != "ENG" {
		t.Errorf("Expected Key 'ENG', got %s", team.Key)
	}
}

func TestNewTestUser(t *testing.T) {
	user := NewTestUser("user-id", "John Doe", "john@example.com")
	if user.ID != "user-id" {
		t.Errorf("Expected ID 'user-id', got %s", user.ID)
	}
	if user.Name != "John Doe" {
		t.Errorf("Expected Name 'John Doe', got %s", user.Name)
	}
	if user.Email != "john@example.com" {
		t.Errorf("Expected Email 'john@example.com', got %s", user.Email)
	}
}
