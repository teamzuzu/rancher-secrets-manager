{{/*
Full endpoint URL for the extension JS bundle directory.
Pattern: <baseUrl>/extensions/rancher-secrets-manager/<appVersion>/plugin
*/}}
{{- define "rsm-ui.endpoint" -}}
{{- printf "%s/extensions/rancher-secrets-manager/%s/plugin" .Values.baseUrl .Chart.AppVersion -}}
{{- end }}
