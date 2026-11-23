package service

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/joa23/linear-cli/internal/format"
	"github.com/joa23/linear-cli/pkg/linear"
)

const (
	testInitiativeUUID = "965bab1e-b1a0-4353-943e-ec14571cc1d2"
	testProjectUUID    = "0b7e2c3a-1111-4222-8333-944455556666"
	testProject2UUID   = "0b7e2c3a-1111-4222-8333-944455557777"
)

// recordedOp is a GraphQL operation the fake API received.
type recordedOp struct {
	Name      string
	Variables map[string]interface{}
}

var opNamePattern = regexp.MustCompile(`(?:query|mutation)\s+(\w+)`)

// fakeLinearAPI answers GraphQL operations by name with canned "data" payloads.
type fakeLinearAPI struct {
	t         *testing.T
	responses map[string]string // operation name -> JSON for "data"
	ops       []recordedOp
}

func (f *fakeLinearAPI) RoundTrip(req *http.Request) (*http.Response, error) {
	body, _ := io.ReadAll(req.Body)
	var payload struct {
		Query     string                 `json:"query"`
		Variables map[string]interface{} `json:"variables"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		f.t.Fatalf("bad request: %v", err)
	}
	name := ""
	if m := opNamePattern.FindStringSubmatch(payload.Query); m != nil {
		name = m[1]
	}
	f.ops = append(f.ops, recordedOp{Name: name, Variables: payload.Variables})

	data, ok := f.responses[name]
	if !ok {
		f.t.Errorf("unexpected operation %q", name)
		data = "null"
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewBufferString(`{"data": ` + data + `}`)),
		Request:    req,
	}, nil
}

func (f *fakeLinearAPI) opNames() []string {
	names := make([]string, len(f.ops))
	for i, op := range f.ops {
		names[i] = op.Name
	}
	return names
}

func newFakeInitiativeService(t *testing.T, responses map[string]string) (*InitiativeService, *fakeLinearAPI) {
	t.Helper()
	api := &fakeLinearAPI{t: t, responses: responses}
	client := linear.NewClient("test-token")
	client.GetBase().SetHTTPClient(&http.Client{Transport: api})
	return NewInitiativeService(client, format.New()), api
}

const fullInitiativeJSON = `{
	"id": "` + testInitiativeUUID + `", "slugId": "3c7a2b1d9e0f", "name": "Q1 Platform",
	"url": "https://linear.app/acme/initiative/q1-platform-3c7a2b1d9e0f", "status": "Active", "health": "onTrack",
	"createdAt": "2026-01-01T00:00:00Z", "updatedAt": "2026-01-02T00:00:00Z",
	"projects": {"nodes": [{"id": "` + testProjectUUID + `", "name": "Auth Revamp", "progress": 0.5, "status": {"name": "In Progress", "type": "started"}}]}
}`

func TestParseInitiativeRef(t *testing.T) {
	tests := []struct {
		name      string
		ref       string
		wantKind  initiativeRefKind
		wantValue string
	}{
		{"empty", "  ", initiativeRefInvalid, ""},
		{"uuid", testInitiativeUUID, initiativeRefUUID, testInitiativeUUID},
		{"bare slug", "3c7a2b1d9e0f", initiativeRefSlug, "3c7a2b1d9e0f"},
		{"title slug", "q1-platform-3c7a2b1d9e0f", initiativeRefSlug, "3c7a2b1d9e0f"},
		{"url", "https://linear.app/acme/initiative/q1-platform-3c7a2b1d9e0f", initiativeRefSlug, "3c7a2b1d9e0f"},
		{"url with tab and query", "https://linear.app/acme/initiative/q1-platform-3c7a2b1d9e0f/projects?x=1", initiativeRefSlug, "3c7a2b1d9e0f"},
		{"url fragment", "https://linear.app/acme/initiative/q1-3c7a2b1d9e0f#updates", initiativeRefSlug, "3c7a2b1d9e0f"},
		{"non-initiative url", "https://linear.app/acme/project/auth-3c7a2b1d9e0f", initiativeRefInvalid, ""},
		{"name", "Q1 Platform", initiativeRefName, "Q1 Platform"},
		{"name with dash", "Q1 - Platform", initiativeRefName, "Q1 - Platform"},
		{"uppercase hex is a name", "Release-3C7A2B1D9E0F", initiativeRefName, "Release-3C7A2B1D9E0F"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kind, value := parseInitiativeRef(tt.ref)
			if kind != tt.wantKind || value != tt.wantValue {
				t.Errorf("parseInitiativeRef(%q) = (%v, %q), want (%v, %q)", tt.ref, kind, value, tt.wantKind, tt.wantValue)
			}
		})
	}
}

func TestNormalizeInitiativeStatus(t *testing.T) {
	for in, want := range map[string]string{"": "", "active": "Active", "PLANNED": "Planned", " Completed ": "Completed", "cancelled": "Canceled"} {
		got, err := normalizeInitiativeStatus(in)
		if err != nil || got != want {
			t.Errorf("normalizeInitiativeStatus(%q) = (%q, %v), want %q", in, got, err, want)
		}
	}
	if _, err := normalizeInitiativeStatus("started"); err == nil {
		t.Error("expected error for invalid status")
	}
}

func TestNormalizeHealth(t *testing.T) {
	for in, want := range map[string]string{"": "", "onTrack": "onTrack", "on-track": "onTrack", "AT_RISK": "atRisk", "off track": "offTrack"} {
		got, err := normalizeHealth(in)
		if err != nil || got != want {
			t.Errorf("normalizeHealth(%q) = (%q, %v), want %q", in, got, err, want)
		}
	}
	if _, err := normalizeHealth("great"); err == nil {
		t.Error("expected error for invalid health")
	}
}

func TestValidateTargetDate(t *testing.T) {
	if err := validateTargetDate(""); err != nil {
		t.Errorf("empty date should be valid: %v", err)
	}
	if err := validateTargetDate("2026-03-31"); err != nil {
		t.Errorf("valid date rejected: %v", err)
	}
	for _, bad := range []string{"2026-13-01", "03/31/2026", "Q1"} {
		if err := validateTargetDate(bad); err == nil {
			t.Errorf("expected error for %q", bad)
		}
	}
}

func TestInitiativeService_Validation(t *testing.T) {
	str := func(s string) *string { return &s }
	svc := NewInitiativeService(nil, format.New()) // must fail before touching the client

	createTests := []struct {
		name    string
		params  InitiativeCreateParams
		wantErr string
	}{
		{"missing name", InitiativeCreateParams{}, "name is required"},
		{"bad status", InitiativeCreateParams{Name: "Q1", Status: "started"}, "invalid status"},
		{"bad date", InitiativeCreateParams{Name: "Q1", TargetDate: "tomorrow"}, "invalid target date"},
	}
	for _, tt := range createTests {
		t.Run("create "+tt.name, func(t *testing.T) {
			_, err := svc.Create(&tt.params)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}

	updateTests := []struct {
		name    string
		params  InitiativeUpdateParams
		wantErr string
	}{
		{"nothing set", InitiativeUpdateParams{}, "nothing to update"},
		{"empty name", InitiativeUpdateParams{Name: str(" ")}, "--name cannot be empty"},
		{"bad status", InitiativeUpdateParams{Status: str("done")}, "invalid status"},
		{"empty status", InitiativeUpdateParams{Status: str("")}, "--status cannot be empty"},
		{"bad date", InitiativeUpdateParams{TargetDate: str("2026/01/01")}, "invalid target date"},
	}
	for _, tt := range updateTests {
		t.Run("update "+tt.name, func(t *testing.T) {
			_, err := svc.Update("Q1 Platform", &tt.params)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}

	t.Run("status update needs body", func(t *testing.T) {
		_, err := svc.CreateUpdate(&InitiativeStatusUpdateParams{Initiative: "Q1", Body: "  "})
		if err == nil || !strings.Contains(err.Error(), "body is required") {
			t.Errorf("error = %v", err)
		}
	})
	t.Run("status update bad health", func(t *testing.T) {
		_, err := svc.CreateUpdate(&InitiativeStatusUpdateParams{Initiative: "Q1", Body: "x", Health: "fine"})
		if err == nil || !strings.Contains(err.Error(), "invalid health") {
			t.Errorf("error = %v", err)
		}
	})
	t.Run("list bad status", func(t *testing.T) {
		_, err := svc.List(&InitiativeListParams{Status: "doing"})
		if err == nil || !strings.Contains(err.Error(), "invalid status") {
			t.Errorf("error = %v", err)
		}
	})
}

func TestInitiativeService_GetByName(t *testing.T) {
	svc, api := newFakeInitiativeService(t, map[string]string{
		"InitiativesByName": `{"initiatives": {"nodes": [{"id": "` + testInitiativeUUID + `", "slugId": "3c7a2b1d9e0f", "name": "Q1 Platform", "status": "Active"}]}}`,
		"Initiative":        `{"initiative": ` + fullInitiativeJSON + `}`,
	})

	out, err := svc.Get("q1 platform", format.VerbosityDetailed, format.OutputText)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !strings.Contains(out, "Q1 Platform") || !strings.Contains(out, "Auth Revamp [In Progress] 50%") {
		t.Errorf("unexpected output:\n%s", out)
	}
	if got := strings.Join(api.opNames(), ","); got != "InitiativesByName,Initiative" {
		t.Errorf("ops = %s", got)
	}
	if api.ops[0].Variables["name"] != "q1 platform" {
		t.Errorf("name variable = %v", api.ops[0].Variables["name"])
	}
}

func TestInitiativeService_GetSlugFallsBackToName(t *testing.T) {
	svc, api := newFakeInitiativeService(t, map[string]string{
		"InitiativeBySlug":  `{"initiatives": {"nodes": []}}`,
		"InitiativesByName": `{"initiatives": {"nodes": [{"id": "` + testInitiativeUUID + `", "name": "release-aaaaaaaaaaaa"}]}}`,
		"Initiative":        `{"initiative": ` + fullInitiativeJSON + `}`,
	})

	if _, err := svc.Get("release-aaaaaaaaaaaa", format.VerbosityCompact, format.OutputJSON); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got := strings.Join(api.opNames(), ","); got != "InitiativeBySlug,InitiativesByName,Initiative" {
		t.Errorf("ops = %s", got)
	}
}

func TestInitiativeService_URLNotFoundDoesNotFallBack(t *testing.T) {
	svc, api := newFakeInitiativeService(t, map[string]string{
		"InitiativeBySlug": `{"initiatives": {"nodes": []}}`,
	})
	_, err := svc.Get("https://linear.app/acme/initiative/q1-3c7a2b1d9e0f", format.VerbosityCompact, format.OutputText)
	if err == nil || !strings.Contains(err.Error(), "initiative not found") {
		t.Errorf("error = %v", err)
	}
	if len(api.ops) != 1 {
		t.Errorf("expected only the slug lookup, got %v", api.opNames())
	}
}

func TestInitiativeService_AmbiguousName(t *testing.T) {
	svc, _ := newFakeInitiativeService(t, map[string]string{
		"InitiativesByName": `{"initiatives": {"nodes": [
			{"id": "a", "slugId": "aaaaaaaaaaaa", "name": "Growth"},
			{"id": "b", "slugId": "bbbbbbbbbbbb", "name": "growth"}
		]}}`,
	})
	_, err := svc.Get("Growth", format.VerbosityCompact, format.OutputText)
	if err == nil || !strings.Contains(err.Error(), "multiple initiatives") || !strings.Contains(err.Error(), "bbbbbbbbbbbb") {
		t.Errorf("error = %v", err)
	}
}

func TestInitiativeService_RemoveProject(t *testing.T) {
	svc, api := newFakeInitiativeService(t, map[string]string{
		"InitiativesByName": `{"initiatives": {"nodes": [{"id": "` + testInitiativeUUID + `", "name": "Q1 Platform"}]}}`,
		"ProjectInitiativeLinks": `{"project": {"initiativeToProjects": {"nodes": [
			{"id": "link-other", "initiative": {"id": "someone-else"}},
			{"id": "link-1", "initiative": {"id": "` + testInitiativeUUID + `"}}
		]}}}`,
		"InitiativeToProjectDelete": `{"initiativeToProjectDelete": {"success": true}}`,
	})

	out, err := svc.RemoveProject("Q1 Platform", testProjectUUID)
	if err != nil {
		t.Fatalf("RemoveProject: %v", err)
	}
	if !strings.Contains(out, "Removed project") {
		t.Errorf("unexpected output: %s", out)
	}
	last := api.ops[len(api.ops)-1]
	if last.Name != "InitiativeToProjectDelete" || last.Variables["id"] != "link-1" {
		t.Errorf("unexpected delete op: %+v", last)
	}
}

func TestInitiativeService_RemoveProjectNotLinked(t *testing.T) {
	svc, api := newFakeInitiativeService(t, map[string]string{
		"InitiativesByName":      `{"initiatives": {"nodes": [{"id": "` + testInitiativeUUID + `", "name": "Q1 Platform"}]}}`,
		"ProjectInitiativeLinks": `{"project": {"initiativeToProjects": {"nodes": []}}}`,
	})

	_, err := svc.RemoveProject("Q1 Platform", testProjectUUID)
	if err == nil || !strings.Contains(err.Error(), "is not linked") {
		t.Errorf("error = %v", err)
	}
	for _, name := range api.opNames() {
		if name == "InitiativeToProjectDelete" {
			t.Error("must not delete when no link exists")
		}
	}
}

func TestInitiativeService_CreateLinksProjects(t *testing.T) {
	svc, api := newFakeInitiativeService(t, map[string]string{
		"InitiativeCreate":          `{"initiativeCreate": {"success": true, "initiative": ` + fullInitiativeJSON + `}}`,
		"InitiativeToProjectCreate": `{"initiativeToProjectCreate": {"success": true, "initiativeToProject": {"id": "link"}}}`,
		"Initiative":                `{"initiative": ` + fullInitiativeJSON + `}`,
	})

	out, err := svc.Create(&InitiativeCreateParams{
		Name:       "Q1 Platform",
		Status:     "active",
		TargetDate: "2026-03-31",
		Projects:   []string{testProjectUUID, testProject2UUID, testProjectUUID}, // duplicate is dropped
		Verbosity:  format.VerbosityCompact,
		OutputType: format.OutputJSON,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	want := "InitiativeCreate,InitiativeToProjectCreate,InitiativeToProjectCreate,Initiative"
	if got := strings.Join(api.opNames(), ","); got != want {
		t.Errorf("ops = %s, want %s", got, want)
	}
	input, _ := api.ops[0].Variables["input"].(map[string]interface{})
	if input["status"] != "Active" || input["targetDate"] != "2026-03-31" {
		t.Errorf("unexpected create input: %v", input)
	}

	var dto format.InitiativeDTO
	if err := json.Unmarshal([]byte(out), &dto); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out)
	}
	if dto.ProjectCount != 1 || len(dto.Projects) != 1 {
		t.Errorf("unexpected projects in output: %+v", dto)
	}
}

func TestInitiativeService_UpdateClearsOwner(t *testing.T) {
	svc, api := newFakeInitiativeService(t, map[string]string{
		"InitiativeUpdate": `{"initiativeUpdate": {"success": true, "initiative": ` + fullInitiativeJSON + `}}`,
	})
	empty := ""
	if _, err := svc.Update(testInitiativeUUID, &InitiativeUpdateParams{Owner: &empty, TargetDate: &empty}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if len(api.ops) != 1 {
		t.Fatalf("UUID refs should not need a lookup, ops = %v", api.opNames())
	}
	input, _ := api.ops[0].Variables["input"].(map[string]interface{})
	for _, k := range []string{"ownerId", "targetDate"} {
		if v, ok := input[k]; !ok || v != nil {
			t.Errorf("%s should be sent as null, got %v (present=%v)", k, v, ok)
		}
	}
}

func TestInitiativeService_CreateUpdatePost(t *testing.T) {
	svc, api := newFakeInitiativeService(t, map[string]string{
		"InitiativeUpdateCreate": `{"initiativeUpdateCreate": {"success": true, "initiativeUpdate": {
			"id": "up-1", "body": "Auth shipped", "health": "atRisk", "createdAt": "2026-01-15T10:00:00Z",
			"user": {"id": "u", "name": "Alice", "displayName": "alice"}, "initiative": {"id": "x", "name": "Q1 Platform"}}}}`,
	})

	out, err := svc.CreateUpdate(&InitiativeStatusUpdateParams{
		Initiative: testInitiativeUUID, Body: "Auth shipped", Health: "at-risk",
		Verbosity: format.VerbosityFull, OutputType: format.OutputText,
	})
	if err != nil {
		t.Fatalf("CreateUpdate: %v", err)
	}
	if !strings.Contains(out, "[At risk]") || !strings.Contains(out, "Auth shipped") {
		t.Errorf("unexpected output:\n%s", out)
	}
	input, _ := api.ops[0].Variables["input"].(map[string]interface{})
	if input["health"] != "atRisk" || input["initiativeId"] != testInitiativeUUID {
		t.Errorf("unexpected input: %v", input)
	}
}
