#!/usr/bin/env bash
set -euo pipefail
work="$1"
operator_ns=opampcommander-operator-system
service=runtime-tls.opamp-e2e.svc

# A TLS gateway shares the same installation as the HTTP/WS regression fixture.
openssl req -x509 -newkey rsa:2048 -nodes -days 1 -subj "/CN=$service" \
  -addext "subjectAltName=DNS:$service" -keyout "$work/tls.key" -out "$work/tls.crt"
kubectl -n opamp-e2e create secret tls runtime-tls --cert="$work/tls.crt" --key="$work/tls.key"
kubectl -n "$operator_ns" create configmap runtime-ca --from-file=ca.crt="$work/tls.crt"
kubectl get deployment apiserver -n opamp-e2e -o json | jq '
  {apiVersion,kind,metadata:{name:"runtime-tls",namespace:"opamp-e2e"},spec}
  | .spec.selector.matchLabels.app = "runtime-tls"
  | .spec.template.metadata.labels.app = "runtime-tls"
  | .spec.template.spec.containers[0].env = [{name:"TLS_CERTFILE",value:"/tls/tls.crt"},{name:"TLS_KEYFILE",value:"/tls/tls.key"}]
  | .spec.template.spec.containers[0].volumeMounts += [{name:"tls",mountPath:"/tls",readOnly:true}]
  | .spec.template.spec.volumes += [{name:"tls",secret:{secretName:"runtime-tls"}}]
' | kubectl apply -f -
kubectl expose deployment runtime-tls -n opamp-e2e --port=8080
kubectl rollout status -n opamp-e2e deployment/runtime-tls --timeout=180s
kubectl patch deployment opampcommander-operator-controller-manager -n "$operator_ns" \
  --patch-file operator/config/runtime/connection-patch.yaml
kubectl rollout status -n "$operator_ns" deployment/opampcommander-operator-controller-manager --timeout=120s
