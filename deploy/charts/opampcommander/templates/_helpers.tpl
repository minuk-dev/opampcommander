{{/*
Chart name, overridable.
*/}}
{{- define "opampcommander.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Fully qualified app name, used as the prefix for every resource name.
*/}}
{{- define "opampcommander.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{- define "opampcommander.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Labels shared by every object in the chart.
*/}}
{{- define "opampcommander.labels" -}}
helm.sh/chart: {{ include "opampcommander.chart" . }}
app.kubernetes.io/name: {{ include "opampcommander.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/part-of: opampcommander
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- with .Values.commonLabels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{- define "opampcommander.annotations" -}}
{{- with .Values.commonAnnotations }}
{{- toYaml . }}
{{- end }}
{{- end }}

{{/*
Image reference. Call with a dict: (dict "image" .Values.<component>.image "context" $).
*/}}
{{- define "opampcommander.image" -}}
{{- $tag := .image.tag | default .context.Chart.AppVersion -}}
{{- printf "%s:%s" .image.repository $tag -}}
{{- end }}

{{/*
=============================================================================
apiserver
=============================================================================
*/}}

{{- define "opampcommander.apiserver.fullname" -}}
{{- printf "%s-apiserver" (include "opampcommander.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "opampcommander.apiserver.selectorLabels" -}}
app.kubernetes.io/name: {{ include "opampcommander.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: apiserver
{{- end }}

{{- define "opampcommander.apiserver.labels" -}}
{{ include "opampcommander.labels" . }}
app.kubernetes.io/component: apiserver
{{- end }}

{{- define "opampcommander.apiserver.serviceAccountName" -}}
{{- if .Values.apiserver.serviceAccount.create }}
{{- default (include "opampcommander.apiserver.fullname" .) .Values.apiserver.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.apiserver.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
The management endpoint (health, metrics, pprof) is a separate, always-ClusterIP
Service. Its metadata carries component "apiserver-management" so a
ServiceMonitor can select it alone, while its spec.selector still targets the
apiserver pods.
*/}}
{{- define "opampcommander.apiserver.managementServiceName" -}}
{{- printf "%s-management" (include "opampcommander.apiserver.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "opampcommander.apiserver.managementSelectorLabels" -}}
app.kubernetes.io/name: {{ include "opampcommander.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: apiserver-management
{{- end }}

{{- define "opampcommander.apiserver.managementLabels" -}}
{{ include "opampcommander.labels" . }}
app.kubernetes.io/component: apiserver-management
{{- end }}

{{- define "opampcommander.apiserver.configMapName" -}}
{{- default (printf "%s-config" (include "opampcommander.apiserver.fullname" .)) .Values.apiserver.existingConfigMap }}
{{- end }}

{{- define "opampcommander.apiserver.secretName" -}}
{{- default (include "opampcommander.apiserver.fullname" .) .Values.apiserver.secrets.existingSecret }}
{{- end }}

{{/*
Whether any Secret-backed credentials are wired into the apiserver pod.
*/}}
{{- define "opampcommander.apiserver.hasSecret" -}}
{{- if or .Values.apiserver.secrets.create .Values.apiserver.secrets.existingSecret -}}
true
{{- end }}
{{- end }}

{{/*
Port helpers. The addresses in `apiserver.config` are the source of truth so the
container ports can never drift from what the process actually binds; the
`containerPorts` values are the fallback when an address is absent or portless.
*/}}
{{- define "opampcommander.portFromAddress" -}}
{{- $port := 0 -}}
{{- if .address -}}
{{- $port = last (splitList ":" (toString .address)) | atoi -}}
{{- end -}}
{{- if not $port -}}
{{- $port = int .default -}}
{{- end -}}
{{- $port -}}
{{- end }}

{{- define "opampcommander.apiserver.httpPort" -}}
{{- include "opampcommander.portFromAddress" (dict
      "address" (dig "address" "" (.Values.apiserver.config | default dict))
      "default" .Values.apiserver.containerPorts.http) -}}
{{- end }}

{{- define "opampcommander.apiserver.managementPort" -}}
{{- include "opampcommander.portFromAddress" (dict
      "address" (dig "management" "address" "" (.Values.apiserver.config | default dict))
      "default" .Values.apiserver.containerPorts.management) -}}
{{- end }}

{{- define "opampcommander.apiserver.directPort" -}}
{{- include "opampcommander.portFromAddress" (dict
      "address" (dig "event" "direct" "listenAddress" "" (.Values.apiserver.config | default dict))
      "default" .Values.apiserver.containerPorts.direct) -}}
{{- end }}

{{- define "opampcommander.apiserver.eventType" -}}
{{- dig "event" "type" "inmemory" (.Values.apiserver.config | default dict) -}}
{{- end }}

{{- define "opampcommander.apiserver.databaseType" -}}
{{- dig "database" "type" "inmemory" (.Values.apiserver.config | default dict) -}}
{{- end }}

{{- define "opampcommander.apiserver.metricsPath" -}}
{{- dig "management" "metric" "prometheus" "path" "/metrics" (.Values.apiserver.config | default dict) -}}
{{- end }}

{{/*
=============================================================================
web
=============================================================================
*/}}

{{- define "opampcommander.web.fullname" -}}
{{- printf "%s-web" (include "opampcommander.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "opampcommander.web.selectorLabels" -}}
app.kubernetes.io/name: {{ include "opampcommander.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: web
{{- end }}

{{- define "opampcommander.web.labels" -}}
{{ include "opampcommander.labels" . }}
app.kubernetes.io/component: web
{{- end }}

{{- define "opampcommander.web.serviceAccountName" -}}
{{- if .Values.web.serviceAccount.create }}
{{- default (include "opampcommander.web.fullname" .) .Values.web.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.web.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Backend URL the dashboard's server-side proxy calls.
*/}}
{{- define "opampcommander.web.apiUrl" -}}
{{- if .Values.web.apiUrl -}}
{{- .Values.web.apiUrl -}}
{{- else -}}
{{- printf "http://%s:%d" (include "opampcommander.apiserver.fullname" .) (int .Values.apiserver.service.port) -}}
{{- end -}}
{{- end }}

{{/*
=============================================================================
Validation. Rendered from the apiserver Deployment so a bad combination fails
`helm install`/`helm template` instead of producing a silently broken release.
=============================================================================
*/}}
{{- define "opampcommander.validateValues" -}}
{{- $messages := list -}}
{{- $eventType := include "opampcommander.apiserver.eventType" . -}}
{{- $databaseType := include "opampcommander.apiserver.databaseType" . -}}
{{- $multiReplica := or (gt (int .Values.apiserver.replicaCount) 1) .Values.apiserver.autoscaling.enabled -}}
{{- if and .Values.apiserver.enabled $multiReplica (eq $databaseType "inmemory") -}}
{{- $messages = append $messages "  - apiserver.config.database.type is \"inmemory\", which is per-process state and cannot be shared. Use \"mongodb\" to run more than one apiserver replica." -}}
{{- end -}}
{{- if and .Values.apiserver.enabled $multiReplica (eq $eventType "inmemory") -}}
{{- $messages = append $messages "  - apiserver.config.event.type is \"inmemory\", so replicas cannot coordinate: a management request handled by one replica cannot reach an agent connected to another. Use \"kafka\" or \"direct\" to run more than one apiserver replica." -}}
{{- end -}}
{{- if and .Values.apiserver.enabled (eq $eventType "kafka") (not (dig "event" "kafka" "brokers" list (.Values.apiserver.config | default dict))) -}}
{{- $messages = append $messages "  - apiserver.config.event.type is \"kafka\" but apiserver.config.event.kafka.brokers is empty." -}}
{{- end -}}
{{- if and .Values.apiserver.enabled (eq $databaseType "mongodb") (not (dig "database" "endpoints" list (.Values.apiserver.config | default dict))) -}}
{{- $messages = append $messages "  - apiserver.config.database.type is \"mongodb\" but apiserver.config.database.endpoints is empty." -}}
{{- end -}}
{{/*
With no Secret and no other credential source, the server falls back to its
built-in flag defaults: admin/admin, and an empty JWT signing key. Anything that
could carry credentials instead — an existing ConfigMap, extraEnv, extraEnvFrom
— counts, so this only fires when there is definitively nothing.
*/}}
{{- if and .Values.apiserver.enabled
      (not .Values.apiserver.secrets.create)
      (not .Values.apiserver.secrets.existingSecret)
      (not .Values.apiserver.existingConfigMap)
      (not .Values.apiserver.extraEnv)
      (not .Values.apiserver.extraEnvFrom) -}}
{{- $messages = append $messages "  - nothing supplies credentials: apiserver.secrets.create is false with no existingSecret, and no existingConfigMap, extraEnv or extraEnvFrom either. The apiserver would start with its built-in admin/admin default and an empty JWT signing key." -}}
{{- end -}}
{{- if and .Values.web.enabled (not .Values.apiserver.enabled) (not .Values.web.apiUrl) -}}
{{- $messages = append $messages "  - apiserver.enabled is false, so web.apiUrl must point at an apiserver deployed elsewhere." -}}
{{- end -}}
{{- if $messages -}}
{{- fail (printf "\nopampcommander: invalid values:\n%s" (join "\n" $messages)) -}}
{{- end -}}
{{- end }}
