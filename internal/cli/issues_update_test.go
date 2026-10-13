package cli

import "testing"

func TestValidateIssueLabelModeFlagsUsesFlagPresence(t *testing.T) {
	if err := validateIssueLabelModeFlags(true, true, false); err == nil {
		t.Fatal("expected empty --add-labels presence to conflict with --labels")
	}
	if err := validateIssueLabelModeFlags(true, false, true); err == nil {
		t.Fatal("expected empty --remove-labels presence to conflict with --labels")
	}
	if err := validateIssueLabelModeFlags(false, true, true); err != nil {
		t.Fatalf("add and remove modes should be compatible: %v", err)
	}
	if err := validateIssueLabelModeFlags(true, false, false); err != nil {
		t.Fatalf("replace-only mode returned %v", err)
	}
}
