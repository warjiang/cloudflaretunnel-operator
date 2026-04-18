{{- define "cloudflaretunnel-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "cloudflaretunnel-operator.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := include "cloudflaretunnel-operator.name" . -}}
{{- if or (eq .Release.Name $name) (hasPrefix (printf "%s-" .Release.Name) $name) -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "cloudflaretunnel-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" -}}
{{- end -}}

{{- define "cloudflaretunnel-operator.labels" -}}
helm.sh/chart: {{ include "cloudflaretunnel-operator.chart" . }}
app.kubernetes.io/name: {{ include "cloudflaretunnel-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "cloudflaretunnel-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "cloudflaretunnel-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
control-plane: controller-manager
{{- end -}}

{{- define "cloudflaretunnel-operator.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (printf "%s-controller-manager" (include "cloudflaretunnel-operator.fullname" .) | trunc 63 | trimSuffix "-") .Values.serviceAccount.name -}}
{{- else -}}
{{- required "serviceAccount.name must be set when serviceAccount.create=false" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{- define "cloudflaretunnel-operator.image" -}}
{{- $tag := default .Chart.AppVersion .Values.image.tag -}}
{{- printf "%s:%s" .Values.image.repository $tag -}}
{{- end -}}

{{- define "cloudflaretunnel-operator.controllerManagerName" -}}
{{- printf "%s-controller-manager" (include "cloudflaretunnel-operator.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "cloudflaretunnel-operator.leaderElectionRoleName" -}}
{{- printf "%s-leader-election-role" (include "cloudflaretunnel-operator.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "cloudflaretunnel-operator.leaderElectionRoleBindingName" -}}
{{- printf "%s-leader-election-rolebinding" (include "cloudflaretunnel-operator.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "cloudflaretunnel-operator.managerRoleName" -}}
{{- printf "%s-manager-role" (include "cloudflaretunnel-operator.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "cloudflaretunnel-operator.managerRoleBindingName" -}}
{{- printf "%s-manager-rolebinding" (include "cloudflaretunnel-operator.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "cloudflaretunnel-operator.metricsAuthRoleName" -}}
{{- printf "%s-metrics-auth-role" (include "cloudflaretunnel-operator.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "cloudflaretunnel-operator.metricsReaderRoleName" -}}
{{- printf "%s-metrics-reader" (include "cloudflaretunnel-operator.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "cloudflaretunnel-operator.metricsAuthRoleBindingName" -}}
{{- printf "%s-metrics-auth-rolebinding" (include "cloudflaretunnel-operator.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "cloudflaretunnel-operator.adminRoleName" -}}
{{- printf "%s-cloudflaretunnel-admin-role" (include "cloudflaretunnel-operator.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "cloudflaretunnel-operator.editorRoleName" -}}
{{- printf "%s-cloudflaretunnel-editor-role" (include "cloudflaretunnel-operator.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "cloudflaretunnel-operator.viewerRoleName" -}}
{{- printf "%s-cloudflaretunnel-viewer-role" (include "cloudflaretunnel-operator.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "cloudflaretunnel-operator.metricsServiceName" -}}
{{- printf "%s-controller-manager-metrics-service" (include "cloudflaretunnel-operator.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}
