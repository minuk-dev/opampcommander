package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteKindKubeconfig(t *testing.T) {
	const clusterName = "review"
	const valid = `apiVersion: v1
kind: Config
current-context: kind-review
contexts:
- name: kind-review
  context:
    cluster: kind-review
clusters:
- name: kind-review
  cluster:
    server: https://127.0.0.1:6443
`
	for _, tc := range []struct {
		name, cluster, config   string
		commandFails, wantError bool
	}{
		{name: "isolated config", cluster: clusterName, config: valid},
		{name: "missing cluster selection", config: valid, wantError: true},
		{name: "wrong current context", cluster: clusterName,
			config:    strings.Replace(valid, "current-context: kind-review", "current-context: production", 1),
			wantError: true},
		{name: "wrong cluster", cluster: clusterName,
			config: strings.Replace(valid, "cluster: kind-review", "cluster: production", 1), wantError: true},
		{name: "missing context", cluster: clusterName, config: "current-context: kind-review\n", wantError: true},
		{name: "invalid config", cluster: clusterName, config: "[invalid", wantError: true},
		{name: "kind failure", cluster: clusterName, commandFails: true, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			original := filepath.Join(dir, "user.kubeconfig")
			for name, content := range map[string]string{"user.kubeconfig": "untouched", "export.yaml": tc.config} {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
					t.Fatal(err)
				}
			}
			script := "#!/bin/sh\n[ \"$*\" = 'get kubeconfig --name review' ] || exit 2\ncat \"$TEST_KIND_CONFIG\"\n"
			if tc.commandFails {
				script = "#!/bin/sh\nexit 1\n"
			}
			kind := filepath.Join(dir, "kind")
			if err := os.WriteFile(kind, []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("KIND", kind)
			t.Setenv("KIND_CLUSTER", tc.cluster)
			t.Setenv("TEST_KIND_CONFIG", filepath.Join(dir, "export.yaml"))
			t.Setenv("KUBECONFIG", original)
			output := filepath.Join(dir, "test.kubeconfig")
			err := WriteKindKubeconfig(output)
			if (err != nil) != tc.wantError {
				t.Fatalf("error = %v, want error = %v", err, tc.wantError)
			}
			if tc.wantError {
				if _, err := os.Stat(output); !os.IsNotExist(err) {
					t.Fatalf("unsafe config was written: %v", err)
				}
			} else {
				data, err := os.ReadFile(output)
				if err != nil || string(data) != valid {
					t.Fatalf("export = %q, error = %v", data, err)
				}
				info, err := os.Stat(output)
				if err != nil {
					t.Fatal(err)
				}
				if info.Mode().Perm() != 0600 {
					t.Fatalf("permissions = %v", info.Mode().Perm())
				}
			}
			data, err := os.ReadFile(original)
			if err != nil || string(data) != "untouched" || os.Getenv("KUBECONFIG") != original {
				t.Fatal("user kubeconfig changed")
			}
		})
	}
}
