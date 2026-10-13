package cli

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestDescriptionFromFlagOrStdin(t *testing.T) {
	tests := []struct {
		name      string
		flagValue string
		input     string
		want      string
	}{
		{name: "literal value bypasses stdin", flagValue: "description", input: "unused", want: "description"},
		{name: "explicit stdin trims whitespace", flagValue: "-", input: " \n  description\n\n", want: "description"},
		{name: "explicit empty stdin remains empty", flagValue: "-", input: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &trackingReader{Reader: strings.NewReader(tt.input)}
			got, err := getDescriptionFromFlagOrStdinWithReader(tt.flagValue, reader)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
			if tt.flagValue != "-" && reader.read {
				t.Fatal("literal flag value consumed stdin")
			}
			if tt.flagValue == "-" && !reader.read {
				t.Fatal("explicit stdin flag did not read stdin")
			}
		})
	}
}

func TestDescriptionFromFlagOrStdinRejectsNilReader(t *testing.T) {
	_, err := getDescriptionFromFlagOrStdinWithReader("-", nil)
	if err == nil || !strings.Contains(err.Error(), "stdin reader is not configured") {
		t.Fatalf("got error %v, want deterministic nil-reader error", err)
	}
}

func TestReadStdinFromReturnsReaderError(t *testing.T) {
	wantErr := errors.New("read failed")
	_, err := readStdinFrom(errorReader{err: wantErr})
	if !errors.Is(err, wantErr) {
		t.Fatalf("got error %v, want %v", err, wantErr)
	}
}

func TestReadStdinFromReadsFinalLineWithoutNewline(t *testing.T) {
	got, err := readStdinFrom(strings.NewReader("first\nsecond"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "first\nsecond" {
		t.Fatalf("got %q, want %q", got, "first\nsecond")
	}
}

type trackingReader struct {
	io.Reader
	read bool
}

func (r *trackingReader) Read(p []byte) (int, error) {
	r.read = true
	return r.Reader.Read(p)
}

type errorReader struct {
	err error
}

func (r errorReader) Read([]byte) (int, error) {
	return 0, r.err
}
