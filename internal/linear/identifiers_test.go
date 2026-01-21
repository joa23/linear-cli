package linear

import (
	"testing"
)

func TestIsIssueIdentifier(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		// Valid identifiers
		{name: "standard format", input: "CEN-123", want: true},
		{name: "single digit", input: "ABC-1", want: true},
		{name: "multiple digits", input: "TEAM-9999", want: true},
		{name: "short prefix", input: "AB-12", want: true},
		{name: "long prefix", input: "ENGINEERING-123", want: true},

		// Invalid identifiers
		{name: "lowercase prefix", input: "cen-123", want: false},
		{name: "mixed case prefix", input: "Cen-123", want: false},
		{name: "underscore separator", input: "CEN_123", want: false},
		{name: "no separator", input: "CEN123", want: false},
		{name: "no number", input: "CEN-", want: false},
		{name: "no prefix", input: "-123", want: false},
		{name: "empty string", input: "", want: false},
		{name: "only prefix", input: "CEN", want: false},
		{name: "only number", input: "123", want: false},
		{name: "spaces", input: "CEN - 123", want: false},
		{name: "special chars in prefix", input: "CE@N-123", want: false},
		{name: "letters after number", input: "CEN-123A", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isIssueIdentifier(tt.input)
			if got != tt.want {
				t.Errorf("isIssueIdentifier(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseIssueIdentifier(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantTeamKey string
		wantNumber  string
		wantErr     bool
	}{
		{
			name:        "standard format",
			input:       "CEN-123",
			wantTeamKey: "CEN",
			wantNumber:  "123",
			wantErr:     false,
		},
		{
			name:        "single digit",
			input:       "ABC-1",
			wantTeamKey: "ABC",
			wantNumber:  "1",
			wantErr:     false,
		},
		{
			name:        "multiple digits",
			input:       "TEAM-9999",
			wantTeamKey: "TEAM",
			wantNumber:  "9999",
			wantErr:     false,
		},
		{
			name:        "invalid format",
			input:       "cen-123",
			wantTeamKey: "",
			wantNumber:  "",
			wantErr:     true,
		},
		{
			name:        "empty string",
			input:       "",
			wantTeamKey: "",
			wantNumber:  "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTeamKey, gotNumber, err := parseIssueIdentifier(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseIssueIdentifier(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}

			if gotTeamKey != tt.wantTeamKey {
				t.Errorf("parseIssueIdentifier(%q) teamKey = %v, want %v", tt.input, gotTeamKey, tt.wantTeamKey)
			}

			if gotNumber != tt.wantNumber {
				t.Errorf("parseIssueIdentifier(%q) number = %v, want %v", tt.input, gotNumber, tt.wantNumber)
			}
		})
	}
}

func TestIsEmail(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		// Valid emails
		{name: "standard email", input: "john@company.com", want: true},
		{name: "with subdomain", input: "john@mail.company.com", want: true},
		{name: "with plus", input: "john+test@company.com", want: true},
		{name: "with dots", input: "john.doe@company.com", want: true},
		{name: "with numbers", input: "john123@company.com", want: true},
		{name: "with hyphens", input: "john-doe@company.com", want: true},

		// Invalid emails
		{name: "no @ symbol", input: "johncompany.com", want: false},
		{name: "no domain", input: "john@", want: false},
		{name: "no local part", input: "@company.com", want: false},
		{name: "empty string", input: "", want: false},
		{name: "spaces", input: "john @company.com", want: false},
		{name: "multiple @", input: "john@@company.com", want: false},
		{name: "just name", input: "John Doe", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isEmail(tt.input)
			if got != tt.want {
				t.Errorf("isEmail(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
