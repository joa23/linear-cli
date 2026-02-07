package cli

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDescriptionFromFlagOrStdin_RegularValue(t *testing.T) {
	result, err := getDescriptionFromFlagOrStdin("hello world")
	require.NoError(t, err)
	assert.Equal(t, "hello world", result)
}

func TestGetDescriptionFromFlagOrStdin_EmptyString(t *testing.T) {
	result, err := getDescriptionFromFlagOrStdin("")
	require.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestGetDescriptionFromFlagOrStdin_DashWithoutPipe(t *testing.T) {
	// "-" without a stdin pipe should error, not hang
	_, err := getDescriptionFromFlagOrStdin("-")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires piped input")
}

func TestGetDescriptionFromFlagOrStdin_DashWithPipe(t *testing.T) {
	// Swap os.Stdin with a pipe containing test data
	r, w, err := os.Pipe()
	require.NoError(t, err)

	origStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = origStdin })

	_, err = w.WriteString("piped content\n")
	require.NoError(t, err)
	w.Close()

	result, err := getDescriptionFromFlagOrStdin("-")
	require.NoError(t, err)
	assert.Equal(t, "piped content", result)
}

func TestGetDescriptionFromFlagOrStdin_DashWithMultilinePipe(t *testing.T) {
	r, w, err := os.Pipe()
	require.NoError(t, err)

	origStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = origStdin })

	_, err = w.WriteString("line one\nline two\nline three\n")
	require.NoError(t, err)
	w.Close()

	result, err := getDescriptionFromFlagOrStdin("-")
	require.NoError(t, err)
	assert.Equal(t, "line one\nline two\nline three", result)
}

func TestGetDescriptionFromFlagOrStdin_FlagTakesPrecedenceOverStdin(t *testing.T) {
	// Even if stdin has a pipe, a non-"-" flag value should be returned as-is
	r, w, err := os.Pipe()
	require.NoError(t, err)

	origStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = origStdin
		w.Close()
	})

	result, err := getDescriptionFromFlagOrStdin("explicit value")
	require.NoError(t, err)
	assert.Equal(t, "explicit value", result)
}
