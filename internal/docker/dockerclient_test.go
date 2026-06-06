package docker

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTarContext(t *testing.T) {
	// Create a temporary binary file
	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "myapp")
	binaryData := []byte("fake binary content")
	require.NoError(t, os.WriteFile(binaryPath, binaryData, 0644))

	baseImage := "alpine:latest"
	reader, err := tarContext(binaryPath, baseImage)
	require.NoError(t, err)
	defer reader.Close()

	// Read the whole tar stream into a buffer
	var buf bytes.Buffer
	_, err = io.Copy(&buf, reader)
	require.NoError(t, err)

	// Parse the tar archive
	tr := tar.NewReader(&buf)

	// --- First entry: Dockerfile ---
	hdr, err := tr.Next()
	require.NoError(t, err)
	assert.Equal(t, "Dockerfile", hdr.Name)
	assert.Equal(t, int64(0644), hdr.Mode)

	body, err := io.ReadAll(tr)
	require.NoError(t, err)
	// The Dockerfile string is built exactly as in tarContext:
	expectedDockerfile := fmt.Sprintf("FROM %s\nCOPY app /usr/local/bin/app\nENTRYPOINT [\"/usr/local/bin/app\"]\n", baseImage)
	assert.Equal(t, expectedDockerfile, string(body))

	// --- Second entry: app ---
	hdr, err = tr.Next()
	require.NoError(t, err)
	assert.Equal(t, "app", hdr.Name)
	assert.Equal(t, int64(0755), hdr.Mode)
	assert.Equal(t, int64(len(binaryData)), hdr.Size)

	body, err = io.ReadAll(tr)
	require.NoError(t, err)
	assert.Equal(t, binaryData, body)

	// --- No more entries ---
	_, err = tr.Next()
	assert.ErrorIs(t, err, io.EOF)
}

func TestStreamBuildOutput_Success(t *testing.T) {
	// Simulate a normal build stream
	input := `{"stream":"Step 1/2 : FROM alpine\n"}
{"stream":" ---\u003e abc123\n"}
{"stream":"Successfully built abc123\n"}
{"aux":{"ID":"sha256:abc..."}}` + "\n"
	reader := strings.NewReader(input)

	// Capture stdout (streamBuildOutput uses fmt.Print)
	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w

	err := streamBuildOutput(reader)

	w.Close()
	var out bytes.Buffer
	_, _ = io.Copy(&out, r)
	os.Stdout = old

	require.NoError(t, err)
	assert.Contains(t, out.String(), "Step 1/2")
	assert.Contains(t, out.String(), "Successfully built abc123")
}

func TestStreamBuildOutput_Error(t *testing.T) {
	// Simulate a stream that returns an error
	input := `{"stream":"Starting build\n"}
{"error":"build failed: something went wrong"}
`
	reader := strings.NewReader(input)

	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w

	err := streamBuildOutput(reader)

	w.Close()
	var out bytes.Buffer
	_, _ = io.Copy(&out, r)
	os.Stdout = old

	require.Error(t, err)
	assert.EqualError(t, err, "build failed: something went wrong")
	assert.Contains(t, out.String(), "Starting build")
}

// captureStdout replaces os.Stdout, runs f, and returns the captured output.
func captureStdout(f func()) string {
	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w

	f()

	w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	os.Stdout = old
	return buf.String()
}

func TestStreamPushOutput_Success(t *testing.T) {
	input := `{"status":"Preparing","progress":"[=]  ","id":"abc123"}
{"status":"Pushing","progress":"[=>]  ","id":"abc123"}
{"status":"Pushed","id":"abc123"}
{"status":"latest: digest: sha256:abc..."}
`
	reader := strings.NewReader(input)

	var out string
	captureStdout(func() {
		out = captureStdout(func() {
			err := streamPushOutput(reader)
			require.NoError(t, err)
		})
	})

	assert.Contains(t, out, "abc123: Preparing [=]")
	assert.Contains(t, out, "abc123: Pushing [=>]")
	assert.Contains(t, out, "abc123: Pushed ")
	assert.Contains(t, out, "latest: digest: sha256:abc...")
}

func TestStreamPushOutput_Error(t *testing.T) {
	input := `{"status":"Preparing","id":"abc123"}
{"error":"unauthorized: access denied"}
`
	reader := strings.NewReader(input)

	err := streamPushOutput(reader)
	require.Error(t, err)
	assert.EqualError(t, err, "push error: unauthorized: access denied")
}

func TestStreamPushOutput_BadJSON(t *testing.T) {
	input := `{"status":"Preparing"}
this is not json
`
	reader := strings.NewReader(input)

	err := streamPushOutput(reader)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid character") // json syntax error
}

func TestStreamPushOutput_EmptyStream(t *testing.T) {
	reader := strings.NewReader("")
	err := streamPushOutput(reader)
	require.NoError(t, err) // EOF immediately returns nil
}
