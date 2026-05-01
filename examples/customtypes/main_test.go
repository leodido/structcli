package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomTypesExample(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		assertFn func(t *testing.T, output string, err error)
	}{
		{
			name: "defaults",
			args: []string{"serve"},
			assertFn: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "listen  = localhost:8080")
				assert.Contains(t, output, "mode    = development")
				assert.Contains(t, output, "timeout = 10s")
				assert.Contains(t, output, "workers = 4")
			},
		},
		{
			name: "all flags",
			args: []string{"serve", "--listen", "0.0.0.0:9090", "--mode", "production", "--timeout", "30s", "--workers", "8"},
			assertFn: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "listen  = 0.0.0.0:9090")
				assert.Contains(t, output, "mode    = production")
				assert.Contains(t, output, "timeout = 30s")
				assert.Contains(t, output, "workers = 8")
			},
		},
		{
			name: "mode alias",
			args: []string{"serve", "--mode", "prod"},
			assertFn: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "mode    = production")
			},
		},
		{
			name: "invalid mode",
			args: []string{"serve", "--mode", "bogus"},
			assertFn: func(t *testing.T, output string, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid mode")
			},
		},
		{
			name: "invalid listen",
			args: []string{"serve", "--listen", "not-a-hostport"},
			assertFn: func(t *testing.T, output string, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid host:port")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts = &ServerOptions{}
			root, serve := buildCLI()
			var buf bytes.Buffer
			serve.SetOut(&buf)
			serve.SetErr(&buf)
			root.SetArgs(tt.args)

			err := root.Execute()
			tt.assertFn(t, buf.String(), err)
		})
	}
}
