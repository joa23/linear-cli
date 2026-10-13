package linear

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/joa23/linear-cli/pkg/linear/core"
)

func TestResolverResolveLabelMetadataSupportsNamesUUIDsAndCache(t *testing.T) {
	var requests atomic.Int32
	server, client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		var request struct {
			Variables struct {
				TeamID string `json:"teamId"`
			} `json:"variables"`
		}
		_ = json.NewDecoder(r.Body).Decode(&request)
		nodes := []core.Label{}
		if request.Variables.TeamID == "team-1" {
			nodes = []core.Label{
				{ID: "11111111-1111-1111-1111-111111111111", Name: "Tests", Parent: &core.LabelRef{ID: "22222222-2222-2222-2222-222222222222", Name: "Issue Type"}},
				{ID: "33333333-3333-3333-3333-333333333333", Name: "Standalone"},
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"team": map[string]interface{}{
					"labels": map[string]interface{}{"nodes": nodes},
				},
			},
		})
	})
	defer server.Close()

	resolver := NewResolver(client)
	label, err := resolver.ResolveLabelMetadata(" tests ", "team-1")
	if err != nil || label.ID != "11111111-1111-1111-1111-111111111111" || label.Parent == nil {
		t.Fatalf("name resolution = %#v, %v", label, err)
	}
	label.Parent.Name = "mutated"
	byID, err := resolver.ResolveLabelMetadata("11111111-1111-1111-1111-111111111111", "team-1")
	if err != nil || byID.Parent == nil || byID.Parent.Name != "Issue Type" {
		t.Fatalf("UUID resolution = %#v, %v", byID, err)
	}
	if requests.Load() != 1 {
		t.Fatalf("label endpoint requests = %d, want 1", requests.Load())
	}
	if _, err := resolver.ResolveLabel("11111111-1111-1111-1111-111111111111", "other-team"); err == nil {
		t.Fatal("cross-team UUID unexpectedly resolved")
	}
	if requests.Load() != 2 {
		t.Fatalf("cross-team lookup requests = %d, want 2", requests.Load())
	}
}
