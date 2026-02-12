{{/*
Expand the name of the chart.
*/}}
{{- define "nats.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}


{{- define "nats.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "nats.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "nats.labels" -}}
helm.sh/chart: {{ include "nats.chart" . }}
{{- range $name, $value := .Values.commonLabels }}
{{ $name }}: {{ $value }}
{{- end }}
{{ include "nats.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "nats.selectorLabels" -}}
{{- if .Values.nats.selectorLabels -}}
{{ .Values.nats.selectorLabels | toYaml }}
{{- else if eq .Release.Name "eventing-nats" -}}
app.kubernetes.io/name: {{ include "nats.name" . }}
app.kubernetes.io/instance: eventing
kyma-project.io/dashboard: eventing
{{- else -}}
app.kubernetes.io/name: {{ include "nats.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
kyma-project.io/dashboard: eventing
{{- end -}}
{{- end }}


{{/*
Return the proper NATS image name
*/}}
{{- define "nats.clusterAdvertise" -}}
{{- printf "$(POD_NAME).%s.$(POD_NAMESPACE).svc.%s" (include "nats.fullname" . ) $.Values.k8sClusterDomain }}
{{- end }}

{{/*
Return the NATS cluster routes.
*/}}
{{- define "nats.clusterRoutes" -}}
{{- $name := (include "nats.fullname" . ) -}}
{{- range $i, $e := until (.Values.cluster.replicas | int) -}}
{{- printf "nats://%s-%d.%s.%s.svc.%s:6222," $name $i $name $.Release.Namespace $.Values.k8sClusterDomain -}}
{{- end -}}
{{- end }}


{{/*
Renders a value that contains template.
Usage:
{{ include "tplvalues.render" ( dict "value" .Values.path.to.the.Value "context" $) }}
*/}}
{{- define "tplvalues.render" -}}
  {{- if typeIs "string" .value }}
    {{- tpl .value .context }}
  {{- else }}
    {{- tpl (.value | toYaml) .context }}
  {{- end }}
{{- end -}}

{{/*
Create a password for admin account.
It will generate a new password if .Values.auth.adminPassword is not provided.
*/}}
{{- define "getAdminPassword" -}}
{{- $result := default (randAlphaNum 60) -}}
{{- if $.pass -}}
{{- printf "%s" $.pass -}}
{{- else -}}
{{- printf "%s" $result -}}
{{- end -}}
{{- end -}}
