{{/*
Expand the name of the chart.
*/}}
{{- define "mcp-capi.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "mcp-capi.fullname" -}}
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

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "mcp-capi.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimAll "-._" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "mcp-capi.labels" -}}
helm.sh/chart: {{ include "mcp-capi.chart" . }}
{{ include "mcp-capi.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
application.giantswarm.io/team: {{ index .Chart.Annotations "io.giantswarm.application.team" | quote }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "mcp-capi.selectorLabels" -}}
app.kubernetes.io/name: {{ include "mcp-capi.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "mcp-capi.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "mcp-capi.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Platform identity contract (global.identity) fallbacks for the OAuth settings.
Each helper returns the explicit local value (oauth.*) when set and falls back
to global.identity / global.domain, so an umbrella chart describes the
platform's identity provider once and a standalone install behaves as before.
*/}}
{{- define "mcp-capi.oauth.globalIdentity" -}}
{{- $g := .Values.global | default dict -}}
{{- $g.identity | default dict | toJson -}}
{{- end }}

{{- define "mcp-capi.oauth.provider" -}}
{{- $p := .Values.oauth.provider | default "dex" -}}
{{- if not (has $p (list "dex" "google")) -}}
{{- fail (printf "oauth.provider must be one of: dex, google (got %q)" $p) -}}
{{- end -}}
{{- $p -}}
{{- end }}

{{- define "mcp-capi.oauth.baseURL" -}}
{{- $domain := dig "domain" "" (.Values.global | default dict) -}}
{{- if .Values.oauth.baseURL -}}
{{- .Values.oauth.baseURL | trimSuffix "/" -}}
{{- else if $domain -}}
{{- printf "https://%s.%s" (include "mcp-capi.fullname" .) $domain -}}
{{- end -}}
{{- end }}

{{- define "mcp-capi.oauth.dexIssuerURL" -}}
{{- $identity := include "mcp-capi.oauth.globalIdentity" . | fromJson -}}
{{- .Values.oauth.dex.issuerURL | default (dig "issuerUrl" "" $identity) -}}
{{- end }}

{{- define "mcp-capi.oauth.dexClientID" -}}
{{- $identity := include "mcp-capi.oauth.globalIdentity" . | fromJson -}}
{{- .Values.oauth.dex.clientID | default (dig "clientId" "" $identity) -}}
{{- end }}

{{- define "mcp-capi.oauth.existingSecret" -}}
{{- $identity := include "mcp-capi.oauth.globalIdentity" . | fromJson -}}
{{- .Values.oauth.existingSecret | default (dig "existingSecret" "" $identity) -}}
{{- end }}

{{/* The Secret the Deployment reads credentials from. */}}
{{- define "mcp-capi.oauth.secretName" -}}
{{- include "mcp-capi.oauth.existingSecret" . | default (printf "%s-oauth" (include "mcp-capi.fullname" .)) -}}
{{- end }}

{{- define "mcp-capi.oauth.caSecretName" -}}
{{- $identity := include "mcp-capi.oauth.globalIdentity" . | fromJson -}}
{{- .Values.oauth.dex.caSecret.name | default (dig "ca" "secretName" "" $identity) -}}
{{- end }}

{{- define "mcp-capi.oauth.caSecretKey" -}}
{{- $identity := include "mcp-capi.oauth.globalIdentity" . | fromJson -}}
{{- if .Values.oauth.dex.caSecret.name -}}
{{- .Values.oauth.dex.caSecret.key | default "ca.crt" -}}
{{- else -}}
{{- dig "ca" "key" (.Values.oauth.dex.caSecret.key | default "ca.crt") $identity -}}
{{- end -}}
{{- end }}

{{/* Comma-separated audiences for OAUTH_TRUSTED_AUDIENCES. */}}
{{- define "mcp-capi.oauth.trustedAudiences" -}}
{{- $identity := include "mcp-capi.oauth.globalIdentity" . | fromJson -}}
{{- if .Values.oauth.trustedAudiences -}}
{{- .Values.oauth.trustedAudiences | join "," -}}
{{- else -}}
{{- dig "clientId" "" $identity -}}
{{- end -}}
{{- end }}

{{/* True when the server acts as the caller (no ServiceAccount credentials). */}}
{{- define "mcp-capi.oauth.downstream" -}}
{{- if and .Values.oauth.enabled .Values.oauth.downstream.enabled -}}true{{- end -}}
{{- end }}
