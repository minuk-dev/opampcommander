#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/../../.."
repo="$PWD"
artifacts="${E2E_ARTIFACTS_DIR:-$repo/build/e2e-controlplane}"
mkdir -p "$artifacts"
artifacts="$(cd "$artifacts" && pwd)"
work="$(mktemp -d)"
cluster="opamp-e2e-$(date +%s)-$$"
export KUBECONFIG="$work/kubeconfig" KUBECTL_KUBERC=false
created=false
forward_pid=""

cleanup() {
  result=$?
  trap - EXIT
  if [ "$created" = true ]; then
    kind export logs --name "$cluster" "$artifacts/kind" || true
    kubectl get pods -A -o wide > "$artifacts/pods.txt" 2>&1 || true
    kubectl get events -A --sort-by=.lastTimestamp > "$artifacts/events.txt" 2>&1 || true
    for deployment in apiserver mongodb redis; do
      kubectl logs -n opamp-e2e "deployment/$deployment" --all-containers --tail=2000 \
        > "$artifacts/$deployment.log" 2>&1 || true
    done
    kubectl logs -n opampcommander-operator-system deployment/opampcommander-operator-controller-manager \
      --all-containers --tail=2000 > "$artifacts/operator.log" 2>&1 || true
    kind delete cluster --name "$cluster" || true
  fi
  if [ -n "$forward_pid" ]; then
    kill "$forward_pid" 2>/dev/null || true
    wait "$forward_pid" 2>/dev/null || true
  fi
  rm -rf "$work"
  exit "$result"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

for tool in docker kind kubectl go; do command -v "$tool" >/dev/null; done
docker info >/dev/null
if kind get clusters | grep -Fxq "$cluster"; then
  echo "Refusing to reuse cluster $cluster" >&2
  exit 1
fi
created=true
kind create cluster --name "$cluster" --image kindest/node:v1.37.0 --kubeconfig "$KUBECONFIG" --wait 120s
test "$(kubectl config current-context)" = "kind-$cluster"
test "$(kubectl config view --minify -o jsonpath='{.contexts[0].context.cluster}')" = "kind-$cluster"

arch="$(docker info --format '{{.Architecture}}')"
case "$arch" in
  aarch64|arm64) arch=arm64 ;;
  x86_64|amd64) arch=amd64 ;;
  *) echo "Unsupported Docker architecture: $arch" >&2; exit 1 ;;
esac
CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -o "$work/apiserver" ./cmd/apiserver
mkdir -p "$work/configs/apiserver"
cp -R configs/apiserver/initial "$work/configs/apiserver/initial"
docker build -f build/docker/apiserver/Dockerfile -t "opamp-e2e-apiserver:$cluster" "$work"
make -C operator docker-build "IMG=opamp-e2e-operator:$cluster"
kind load docker-image --name "$cluster" "opamp-e2e-apiserver:$cluster" "opamp-e2e-operator:$cluster"

# Reuse the existing manager health/metrics suite on the same isolated Kind cluster.
(
  cd operator
  KIND_CLUSTER="$cluster" OPERATOR_E2E_IMAGE="opamp-e2e-operator:$cluster" CERT_MANAGER_INSTALL_SKIP=true \
    go test -tags=e2e ./test/e2e -count=1 -v -timeout=10m
) 2>&1 | tee "$artifacts/operator-tests.log"

# The manager suite tears down its fixtures; install a fresh combined control plane.
operator/bin/kustomize build operator/config/default > "$work/operator.yaml"
sed "s|ghcr.io/minuk-dev/opampcommander-operator:dev|opamp-e2e-operator:$cluster|g" \
  "$work/operator.yaml" > "$work/operator-test.yaml"
kubectl apply -f "$work/operator-test.yaml"
installation_uid="$(kubectl get configmap -n opampcommander-operator-system \
  opampcommander-operator-opamp-installation -o jsonpath='{.metadata.uid}')"
test -n "$installation_uid"
sed -e "s|APISERVER_IMAGE|opamp-e2e-apiserver:$cluster|g" -e "s|INSTALLATION_UID|$installation_uid|g" \
  test/e2e/controlplane/manifests.yaml > "$work/runtime.yaml"
kubectl apply -f "$work/runtime.yaml"
kubectl rollout status -n opamp-e2e deployment/mongodb --timeout=180s
kubectl rollout status -n opamp-e2e deployment/redis --timeout=180s
kubectl rollout status -n opamp-e2e deployment/apiserver --timeout=180s
kubectl rollout status -n opampcommander-operator-system deployment/opampcommander-operator-controller-manager --timeout=180s

kubectl port-forward -n opamp-e2e service/apiserver :8080 --address=127.0.0.1 > "$artifacts/port-forward.log" 2>&1 &
forward_pid=$!
port=""
for _ in $(seq 1 80); do
  kill -0 "$forward_pid"
  port="$(sed -nE 's/Forwarding from 127\.0\.0\.1:([0-9]+).*/\1/p' "$artifacts/port-forward.log" | head -n1)"
  if [ -n "$port" ]; then break; fi
  sleep 0.25
done
test -n "$port"
OPAMP_E2E_PORT="$port" OPAMP_E2E_CLUSTER="$cluster" \
  go test -tags=kind_e2e ./test/e2e/controlplane -count=1 -v -timeout=10m 2>&1 | tee "$artifacts/controlplane-tests.log"
