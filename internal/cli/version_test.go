package cli

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionCmd(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		commit      string
		date        string
		wantContain []string
	}{
		{
			name:    "default dev values",
			version: "dev",
			commit:  "none",
			date:    "unknown",
			wantContain: []string{
				"hammer",
				"dev",
				"commit",
				"none",
				"built",
				"unknown",
			},
		},
		{
			name:    "release docker values",
			version: "v1.2.3",
			commit:  "abc1234",
			date:    "2024-01-15",
			wantContain: []string{
				"hammer",
				"v1.2.3",
				"commit",
				"abc1234",
				"built",
				"2024-01-15",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version = tt.version
			commit = tt.commit
			date = tt.date

			var buf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&buf, nil))
			slog.SetDefault(logger)
			t.Cleanup(func() { slog.SetDefault(slog.Default()) })

			cmd := newVersionCmd()
			require.NoError(t, cmd.Execute())

			output := buf.String()
			for _, want := range tt.wantContain {
				require.Contains(t, output, want)
			}
		})
	}
}

func TestVersionCmdMetadata(t *testing.T) {
	cmd := newVersionCmd()

	require.Equal(t, "version", cmd.Use)
	require.NotEmpty(t, cmd.Short)
	require.NotNil(t, cmd.Run)
}
