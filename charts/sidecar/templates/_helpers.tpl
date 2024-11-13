{{/*
Expand the name of the chart.
*/}}
{{- define "sidecar.name" -}}
{{- default .Chart.Name .Values.sidecar.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}


{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "sidecar.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}


{{/*
Common labels
*/}}
{{- define "sidecar.labels" -}}
helm.sh/chart: {{ include "sidecar.chart" . }}
{{ include "sidecar.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}
{{- with .Values.sidecar.additionalLabels }}
{{- toYaml . | nindent 0 }}
{{- end }}


{{- define "sidecar.selectorLabels" -}}
app.kubernetes.io/name: {{ include "sidecar.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}


{{- define "sidecar.metadataLabels" -}}
{{ include "sidecar.selectorLabels" . }}
{{- with .Values.sidecar.metadataLabels }}
{{- toYaml . | nindent 0 }}
{{- end }}
{{- end }}
