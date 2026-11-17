package build_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCRDCommandsPropagateFailures(t *testing.T) {
	const renderFailure = "render failure"
	for _, target := range []string{"install", "uninstall"} {
		t.Run(target, func(t *testing.T) {
			for _, scenario := range []string{renderFailure, "kubectl failure", "success"} {
				t.Run(scenario, func(t *testing.T) {
					dir := t.TempDir()
					marker := filepath.Join(dir, "kubectl-called")
					t.Setenv("TEST_KUBECTL_MARKER", marker)
					kustomizeScript := "printf 'kind: CustomResourceDefinition\\n'\n"
					if scenario == renderFailure {
						kustomizeScript = "echo render-failed >&2\nexit 7\n"
					}
					kubectlScript := "cat >/dev/null\ntouch \"$TEST_KUBECTL_MARKER\"\n"
					if scenario == "kubectl failure" {
						kubectlScript += "exit 9\n"
					}
					kustomize := writeCommand(t, dir, "kustomize", kustomizeScript)
					kubectl := writeCommand(t, dir, "kubectl", kubectlScript)
					cmd := exec.Command("make", "-o", "manifests", "-o", "kustomize", target,
						"KUSTOMIZE="+kustomize, "KUBECTL="+kubectl)
					cmd.Dir = filepath.Join("..", "..")
					output, err := cmd.CombinedOutput()
					if (err == nil) != (scenario == "success") {
						t.Fatalf("make: %v\n%s", err, output)
					}
					if scenario == renderFailure {
						if !strings.Contains(string(output), "render-failed") {
							t.Fatalf("error hidden: %s", output)
						}
						if _, err := os.Stat(marker); !os.IsNotExist(err) {
							t.Fatal("kubectl ran after render failure")
						}
					} else if _, err := os.Stat(marker); err != nil {
						t.Fatal("kubectl was not called")
					}
				})
			}
		})
	}
}

func TestKindSetupPreservesUserKubeconfig(t *testing.T) {
	for _, existing := range []bool{true, false} {
		name := "create"
		if existing {
			name = "reuse"
		}
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			original := filepath.Join(dir, "user.kubeconfig")
			if err := os.WriteFile(original, []byte("untouched"), 0600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("KUBECONFIG", original)
			clusters := "review-other"
			if existing {
				clusters = "review"
			}
			t.Setenv("TEST_KIND_CLUSTERS", clusters)
			marker := filepath.Join(dir, "created")
			t.Setenv("TEST_KIND_MARKER", marker)
			kind := writeCommand(t, dir, "kind", `if [ "$*" = 'get clusters' ]; then
  echo "$TEST_KIND_CLUSTERS"
elif [ "$1 $2 $3 $4 $5" = 'create cluster --name review --kubeconfig' ]; then
  [ "$6" != "$KUBECONFIG" ] && [ -f "$6" ] || exit 3
  echo isolated > "$6"
  touch "$TEST_KIND_MARKER"
else
  exit 4
fi
`)
			cmd := exec.Command("make", "setup-test-e2e", "KIND="+kind, "KIND_CLUSTER=review")
			cmd.Dir = filepath.Join("..", "..")
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("make: %v\n%s", err, output)
			}
			data, err := os.ReadFile(original)
			if err != nil || string(data) != "untouched" {
				t.Fatal("user kubeconfig changed")
			}
			_, err = os.Stat(marker)
			if existing && !os.IsNotExist(err) {
				t.Fatal("existing cluster was recreated")
			}
			if !existing && err != nil {
				t.Fatal("cluster was not created")
			}
		})
	}
}

func writeCommand(t *testing.T, dir, name, script string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\nset -eu\n"+script), 0700); err != nil {
		t.Fatal(err)
	}
	return path
}
