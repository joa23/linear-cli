package projects

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/joa23/linear-cli/pkg/linear/core"
	"github.com/joa23/linear-cli/pkg/linear/testutil"
)

func TestNormalizeStatusNames(t *testing.T) {
	got, err := NormalizeStatusNames([]string{" In Progress ", "on hold", "IN PROGRESS"})
	if err != nil {
		t.Fatalf("NormalizeStatusNames() error = %v", err)
	}
	want := []string{"In Progress", "on hold"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("NormalizeStatusNames() = %#v, want %#v", got, want)
	}

	_, err = NormalizeStatusNames([]string{"In Progress", ""})
	if err == nil || !strings.Contains(err.Error(), "empty value") {
		t.Fatalf("expected empty-value error, got %v", err)
	}
}

func TestProjectUnmarshalStatusPreservesStateAlias(t *testing.T) {
	var project core.Project
	err := json.Unmarshal([]byte(`{"id":"p1","state":"started","status":{"id":"s1","name":"On Hold","type":"started"}}`), &project)
	if err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if project.Status == nil || project.Status.Name != "On Hold" {
		t.Fatalf("status = %#v, want named status", project.Status)
	}
	if project.State != "started" {
		t.Fatalf("state = %q, want started", project.State)
	}
	if project.StatusName() != "On Hold" {
		t.Fatalf("StatusName() = %q, want On Hold", project.StatusName())
	}
}

func TestResolveProjectStatusNamesIgnoresArchivedAndRejectsUnknown(t *testing.T) {
	archived := "2026-01-01T00:00:00Z"
	body := testutil.NewGraphQLDataResponse(map[string]interface{}{
		"organization": map[string]interface{}{
			"projectStatuses": []map[string]interface{}{
				{"id": "active", "name": "In Progress", "type": "started"},
				{"id": "archived", "name": "Old", "type": "started", "archivedAt": archived},
			},
		},
	})
	base := core.NewBaseClient("test-token")
	base.SetHTTPClient(testHTTPClient(body))
	client := NewClient(base)

	ids, err := client.ResolveProjectStatusNames([]string{" in progress "})
	if err != nil || len(ids) != 1 || ids[0] != "active" {
		t.Fatalf("ResolveProjectStatusNames() = %#v, %v", ids, err)
	}
	base = core.NewBaseClient("test-token")
	base.SetHTTPClient(testHTTPClient(body))
	client = NewClient(base)
	_, err = client.ResolveProjectStatusNames([]string{"Old"})
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected archived status to be unavailable, got %v", err)
	}
}

func TestListAllProjectsWithStatusSendsORStatusFilter(t *testing.T) {
	transport := &sequentialTransport{responses: []interface{}{testutil.NewGraphQLDataResponse(map[string]interface{}{
		"projects": map[string]interface{}{"nodes": []map[string]interface{}{
			{"id": "p1", "state": "started", "status": map[string]interface{}{"id": "s1", "name": "In Progress", "type": "started"}},
		}},
	})}}
	base := core.NewBaseClient("test-token")
	base.SetHTTPClient(&http.Client{Transport: transport})
	client := NewClient(base)

	if _, err := client.ListAllProjectsWithStatus(3, []string{"s1", "s2"}); err != nil {
		t.Fatalf("ListAllProjectsWithStatus() error = %v", err)
	}
	var request struct {
		Variables map[string]interface{} `json:"variables"`
	}
	if err := json.Unmarshal([]byte(transport.requests[0]), &request); err != nil {
		t.Fatalf("request JSON error = %v", err)
	}
	filter := request.Variables["filter"].(map[string]interface{})
	status := filter["status"].(map[string]interface{})
	ids := status["id"].(map[string]interface{})["in"].([]interface{})
	if len(ids) != 2 || ids[0] != "s1" || ids[1] != "s2" {
		t.Fatalf("status IDs = %#v, want s1,s2", ids)
	}
}

func TestResolveProjectStatusNamesRejectsAmbiguousActiveNames(t *testing.T) {
	body := testutil.NewGraphQLDataResponse(map[string]interface{}{
		"organization": map[string]interface{}{
			"projectStatuses": []map[string]interface{}{
				{"id": "s1", "name": "Started", "type": "started"},
				{"id": "s2", "name": " started ", "type": "started"},
			},
		},
	})
	base := core.NewBaseClient("test-token")
	base.SetHTTPClient(withResponses(body))
	client := NewClient(base)
	_, err := client.ResolveProjectStatusNames([]string{"started"})
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected ambiguous status error, got %v", err)
	}
}

func TestListUserProjectsPaginatesBeforeApplyingLimit(t *testing.T) {
	transport := &sequentialTransport{responses: []interface{}{
		testutil.NewGraphQLDataResponse(map[string]interface{}{
			"projects": map[string]interface{}{
				"nodes": []map[string]interface{}{
					{"id": "p1", "name": "First", "issues": map[string]interface{}{"nodes": []map[string]interface{}{{"id": "i1", "assignee": map[string]interface{}{"id": "u1"}}}}},
					{"id": "p2", "name": "Not Mine", "issues": map[string]interface{}{"nodes": []interface{}{}}},
				},
				"pageInfo": map[string]interface{}{"hasNextPage": true, "endCursor": "cursor-1"},
			},
		}),
		testutil.NewGraphQLDataResponse(map[string]interface{}{
			"projects": map[string]interface{}{
				"nodes": []map[string]interface{}{
					{"id": "p3", "name": "Third", "issues": map[string]interface{}{"nodes": []map[string]interface{}{{"id": "i3", "assignee": map[string]interface{}{"id": "u1"}}}}},
				},
				"pageInfo": map[string]interface{}{"hasNextPage": false},
			},
		}),
	}}
	base := core.NewBaseClient("test-token")
	base.SetHTTPClient(&http.Client{Transport: transport})
	client := NewClient(base)

	got, err := client.ListUserProjects("u1", 2)
	if err != nil {
		t.Fatalf("ListUserProjects() error = %v", err)
	}
	if len(got) != 2 || got[0].ID != "p1" || got[1].ID != "p3" {
		t.Fatalf("projects = %#v, want p1 then p3", got)
	}
	if len(transport.requests) != 2 {
		t.Fatalf("requests = %d, want 2", len(transport.requests))
	}
	if !strings.Contains(transport.requests[1], `"after":"cursor-1"`) {
		t.Fatalf("second request = %s, missing cursor", transport.requests[1])
	}
}

func TestUpdateProjectStateWithResultKeepsLegacyVariablesAndDecodesStatus(t *testing.T) {
	transport := &sequentialTransport{responses: []interface{}{testutil.NewGraphQLDataResponse(map[string]interface{}{
		"projectUpdate": map[string]interface{}{
			"success": true,
			"project": map[string]interface{}{
				"id": "p1", "state": "started",
				"status": map[string]interface{}{"id": "s1", "name": "In Progress", "type": "started"},
			},
		},
	})}}
	base := core.NewBaseClient("test-token")
	base.SetHTTPClient(&http.Client{Transport: transport})
	client := NewClient(base)

	project, err := client.UpdateProjectStateWithResult("p1", "started")
	if err != nil {
		t.Fatalf("UpdateProjectStateWithResult() error = %v", err)
	}
	if project.Status == nil || project.Status.Name != "In Progress" || project.State != "started" {
		t.Fatalf("project = %#v, want named status and legacy state", project)
	}
	var request struct {
		Query     string                 `json:"query"`
		Variables map[string]interface{} `json:"variables"`
	}
	if err := json.Unmarshal([]byte(transport.requests[0]), &request); err != nil {
		t.Fatalf("request JSON error = %v", err)
	}
	if !strings.Contains(request.Query, "input: { state: $state }") || request.Variables["state"] != "started" {
		t.Fatalf("request = %#v, want legacy state mutation", request)
	}
}

type sequentialTransport struct {
	responses []interface{}
	requests  []string
}

func (t *sequentialTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	t.requests = append(t.requests, string(body))
	response := t.responses[0]
	if len(t.responses) > 1 {
		t.responses = t.responses[1:]
	}
	encoded, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(encoded)), Header: make(http.Header)}, nil
}

func testHTTPClient(body interface{}) *http.Client {
	return &http.Client{Transport: testutil.NewSuccessTransport(body)}
}

func withResponses(body interface{}) *http.Client {
	return &http.Client{Transport: &sequentialTransport{responses: []interface{}{body}}}
}
