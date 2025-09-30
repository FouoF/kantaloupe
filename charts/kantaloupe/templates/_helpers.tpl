{{/* vim: set filetype=mustache: */}}
{{/*

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "kantaloupe.controllerManager.fullname" -}}
{{- printf "%s-%s-%s" (include "common.names.fullname" .) "controller" "manager" | trunc 63 | trimSuffix "-" -}}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "kantaloupe.apiserver.fullname" -}}
{{- printf "%s-%s" (include "common.names.fullname" .) "apiserver" | trunc 63 | trimSuffix "-" -}}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "kantaloupe.ui.fullname" -}}
{{- printf "%s-%s" (include "common.names.fullname" .) "ui" | trunc 63 | trimSuffix "-" -}}
{{- end }}

{{/*
Return the proper image name
*/}}
{{- define "kantaloupe.controllerManager.image" -}}
{{ include "common.images.image" (dict "imageRoot" .Values.controllerManager.image "global" .Values.global) }}
{{- end -}}

{{/*
Return the proper image name
*/}}
{{- define "kantaloupe.apiserver.image" -}}
{{ include "common.images.image" (dict "imageRoot" .Values.apiserver.image "global" .Values.global) }}
{{- end -}}

{{- define "kantaloupe.cloudshell.image" -}}
{{ include "common.images.image" (dict "imageRoot" .Values.cloudshellImage "global" .Values.global) }}
{{- end -}}

{{/*
Return the proper image name
*/}}
{{- define "kantaloupe.ui.image" -}}
{{ include "common.images.image" (dict "imageRoot" .Values.ui.image "global" .Values.global) }}
{{- end -}}

{{/*
Return the proper image Registry Secret Names
*/}}
{{- define "kantaloupe.controllerManager.imagePullSecrets" -}}
{{ include "common.images.pullSecrets" (dict "images" (list .Values.controllerManager.image) "global" .Values.global) }}
{{- end -}}

{{/*
Return the proper image Registry Secret Names
*/}}
{{- define "kantaloupe.apiserver.imagePullSecrets" -}}
{{ include "common.images.pullSecrets" (dict "images" (list .Values.apiserver.image) "global" .Values.global) }}
{{- end -}}

{{/*
Return the proper image Registry Secret Names
*/}}
{{- define "kantaloupe.ui.imagePullSecrets" -}}
{{ include "common.images.pullSecrets" (dict "images" (list .Values.ui.image) "global" .Values.global) }}
{{- end -}}
