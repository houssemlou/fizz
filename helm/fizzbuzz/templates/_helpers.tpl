{{- define "fizzbuzz.name" -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "fizzbuzz.apiImage" -}}
{{- if .Values.image.registry -}}
{{ .Values.image.registry }}/{{ .Values.image.repository }}-server:{{ .Values.image.tag }}
{{- else -}}
{{ .Values.image.repository }}-server:{{ .Values.image.tag }}
{{- end }}
{{- end }}

{{- define "fizzbuzz.workerImage" -}}
{{- if .Values.image.registry -}}
{{ .Values.image.registry }}/{{ .Values.image.repository }}-worker:{{ .Values.image.tag }}
{{- else -}}
{{ .Values.image.repository }}-worker:{{ .Values.image.tag }}
{{- end }}
{{- end }}

{{- define "fizzbuzz.labels" -}}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "fizzbuzz.selectorLabels" -}}
app.kubernetes.io/name: {{ include "fizzbuzz.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "fizzbuzz.apiSelectorLabels" -}}
{{ include "fizzbuzz.selectorLabels" . }}
app.kubernetes.io/component: api
{{- end }}

{{- define "fizzbuzz.workerSelectorLabels" -}}
{{ include "fizzbuzz.selectorLabels" . }}
app.kubernetes.io/component: worker
{{- end }}
