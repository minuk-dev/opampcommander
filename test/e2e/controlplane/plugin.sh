#!/usr/bin/env bash
set -euo pipefail

# run.sh supplies a disposable Kind kubeconfig and the directory for test binaries.
plugin_bin="$1"
export PATH="$plugin_bin:$PATH"
go build -o "$plugin_bin/kubectl-opamp" ./cmd/kubectl-opamp
kubectl_bin="$(command -v kubectl)"
PATH="$plugin_bin" "$kubectl_bin" plugin list | grep -F "$plugin_bin/kubectl-opamp"
sample=operator/config/samples/opampcommander_v1alpha1_agentgroup.yaml
kubectl create namespace plugin-validation
kubectl config set-context plugin-validation --cluster="kind-$cluster" --user="kind-$cluster" --namespace=plugin-validation
kubectl opamp validate --kubeconfig "$KUBECONFIG" --context plugin-validation -f "$sample"
test -z "$(kubectl get agentgroup collectors -n plugin-validation --ignore-not-found -o name)"
kubectl apply --server-side -n plugin-validation -f "$sample"
rv="$(kubectl get agentgroup collectors -n plugin-validation -o jsonpath='{.metadata.resourceVersion}')"
sed 's/priority: 0/priority: 8/' "$sample" | kubectl opamp validate -n plugin-validation -f -
test "$rv" = "$(kubectl get agentgroup collectors -n plugin-validation -o jsonpath='{.metadata.resourceVersion}')"
if sed 's/priority: 0/unknownField: 8/' "$sample" | kubectl opamp validate -n plugin-validation -f -; then
  echo 'unexpected success for an unknown field' >&2
  exit 1
fi
# Admission can include rejected values in its diagnostics.
if sed 's/priority: 0/priority: private-config-marker/' "$sample" | kubectl opamp validate -n plugin-validation -f - > "$plugin_bin/rejected.log" 2>&1; then
  echo 'unexpected success for a non-integer priority' >&2
  exit 1
fi
# Kubernetes can report malformed server-side apply conversions as InternalServerError.
grep -Eq 'server dry-run request failed \(HTTP (4[0-9]{2}|500) ' "$plugin_bin/rejected.log"
if grep -q private-config-marker "$plugin_bin/rejected.log"; then
  echo 'validation disclosed a rejected value' >&2
  exit 1
fi
# Keep the existing credentials and impersonate a subject with no RoleBinding.
kubectl config view --raw --minify --flatten -o json | jq '
  .users[0].user.as = "system:serviceaccount:plugin-validation:unbound"
' > "$plugin_bin/denied.json"
if kubectl opamp validate --kubeconfig "$plugin_bin/denied.json" -n plugin-validation -f "$sample" > "$plugin_bin/denied.log" 2>&1; then
  echo 'unexpected success for an unauthorized subject' >&2
  exit 1
fi
grep -qi forbidden "$plugin_bin/denied.log"
