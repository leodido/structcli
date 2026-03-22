package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectionsExample(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		envs       map[string]string
		configBody string
		assertFn   func(t *testing.T, output string, err error)
	}{
		{
			name: "flags only",
			args: []string{
				"--retries", "1,2,3",
				"--backoffs", "1s,5s",
				"--feature-on", "true,false",
				"--labels", "env=prod,team=platform",
				"--limits", "cpu=8,memory=16",
				"--counts", "ok=10,fail=3",
			},
			assertFn: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "retries=[1 2 3]")
				assert.Contains(t, output, "backoffs=[1s 5s]")
				assert.Contains(t, output, "feature_on=[true false]")
				assert.Contains(t, output, "labels=map[env:prod team:platform]")
				assert.Contains(t, output, "limits=map[cpu:8 memory:16]")
				assert.Contains(t, output, "counts=map[fail:3 ok:10]")
			},
		},
		{
			name: "env only",
			envs: map[string]string{
				"MYAPP_RETRIES":    "4,5",
				"MYAPP_BACKOFFS":   "4s,5s",
				"MYAPP_FEATURE_ON": "false,true",
				"MYAPP_LABELS":     "env=env,team=ops",
				"MYAPP_LIMITS":     "cpu=6,memory=12",
				"MYAPP_COUNTS":     "ok=7,fail=2",
			},
			assertFn: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "retries=[4 5]")
				assert.Contains(t, output, "backoffs=[4s 5s]")
				assert.Contains(t, output, "feature_on=[false true]")
				assert.Contains(t, output, "labels=map[env:env team:ops]")
				assert.Contains(t, output, "limits=map[cpu:6 memory:12]")
				assert.Contains(t, output, "counts=map[fail:2 ok:7]")
			},
		},
		{
			name: "config only",
			configBody: `retries: "9,10"
backoffs:
  - 9s
  - 10s
feature-on: "true,true"
labels:
  env: cfg
  team: docs
limits:
  cpu: 4
  memory: 8
counts: "ok=6,fail=1"
`,
			assertFn: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "Using config file:")
				assert.Contains(t, output, "retries=[9 10]")
				assert.Contains(t, output, "backoffs=[9s 10s]")
				assert.Contains(t, output, "feature_on=[true true]")
				assert.Contains(t, output, "labels=map[env:cfg team:docs]")
				assert.Contains(t, output, "limits=map[cpu:4 memory:8]")
				assert.Contains(t, output, "counts=map[fail:1 ok:6]")
			},
		},
		{
			name: "flags override env and config",
			args: []string{
				"--retries", "1,2,3",
				"--backoffs", "1s,5s",
				"--feature-on", "true,false",
				"--labels", "env=prod,team=platform",
				"--limits", "cpu=8,memory=16",
				"--counts", "ok=10,fail=3",
			},
			envs: map[string]string{
				"MYAPP_RETRIES":    "7,8",
				"MYAPP_BACKOFFS":   "7s,8s",
				"MYAPP_FEATURE_ON": "false,true",
				"MYAPP_LABELS":     "env=env,team=ops",
				"MYAPP_LIMITS":     "cpu=6,memory=12",
				"MYAPP_COUNTS":     "ok=7,fail=2",
			},
			configBody: `retries: "9,10"
backoffs:
  - 9s
  - 10s
feature-on: "true,true"
labels:
  env: cfg
  team: docs
limits:
  cpu: 4
  memory: 8
counts: "ok=6,fail=1"
`,
			assertFn: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "retries=[1 2 3]")
				assert.Contains(t, output, "backoffs=[1s 5s]")
				assert.Contains(t, output, "feature_on=[true false]")
				assert.Contains(t, output, "labels=map[env:prod team:platform]")
				assert.Contains(t, output, "limits=map[cpu:8 memory:16]")
				assert.Contains(t, output, "counts=map[fail:3 ok:10]")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for key, value := range tc.envs {
				t.Setenv(key, value)
			}

			cmd, err := NewRootCmd()
			require.NoError(t, err)

			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&out)

			args := append([]string(nil), tc.args...)
			if tc.configBody != "" {
				configPath := filepath.Join(t.TempDir(), "config.yaml")
				err := os.WriteFile(configPath, []byte(tc.configBody), 0o600)
				require.NoError(t, err)
				args = append([]string{"--config", configPath}, args...)
			}

			cmd.SetArgs(args)
			err = cmd.Execute()

			tc.assertFn(t, out.String(), err)
		})
	}
}
