package runner

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOSRunner_Run(t *testing.T) {
	t.Setenv("TEST", "TRUE")
	type args struct {
		ctx  context.Context
		name string
		args []string
		opts Options
	}
	tests := []struct {
		name    string
		args    args
		want    *Result
		wantErr bool
	}{
		{
			name: "RunCommandWithEnvVariables",
			args: args{
				ctx:  context.Background(),
				name: "sh",
				args: []string{"-c", "echo $FOO"},
				opts: Options{
					Dir: ".",
					Env: []string{"FOO=BAR"},
				},
			},
			want: &Result{
				Stdout:   []byte("BAR\n"),
				Stderr:   []byte{},
				ExitCode: 0,
			},
			wantErr: false,
		},
		{
			name: "RunCommandWithDir",
			args: args{
				ctx:  context.Background(),
				name: "sh",
				args: []string{"-c", "pwd"},
				opts: Options{
					Dir: "/",
					Env: nil,
				},
			},
			want: &Result{
				Stdout:   []byte("/\n"),
				Stderr:   []byte{},
				ExitCode: 0,
			},
			wantErr: false,
		},
		{
			name: "RunCommandWithExit1",
			args: args{
				ctx:  context.Background(),
				name: "sh",
				args: []string{"-c", "exit 1"},
				opts: Options{
					Dir: ".",
					Env: nil,
				},
			},
			want: &Result{
				Stdout:   []byte{},
				Stderr:   []byte{},
				ExitCode: 1,
			},
			wantErr: true,
		},
		{
			name: "RunCommandWithError",
			args: args{
				ctx:  context.Background(),
				name: "asdfasdf",
				args: []string{"adsfasdf"},
				opts: Options{
					Dir: ".",
					Env: nil,
				},
			},
			want: &Result{
				Stdout:   nil,
				Stderr:   nil,
				ExitCode: -1,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os := New()
			got, err := os.Run(tt.args.ctx, tt.args.name, tt.args.args, tt.args.opts)
			if (err != nil) != tt.wantErr {
				require.Equal(t, tt.wantErr, err)
			} else {
				require.Equal(t, tt.want, got)
			}
		})
	}
}
