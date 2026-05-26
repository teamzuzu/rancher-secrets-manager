{{/*
Base URL for the extension. Rancher appends /plugin internally when fetching bundles.
Pattern: <baseUrl>/extensions/rancher-secrets-manager/<appVersion>
*/}}
{{- define "rsm-ui.endpoint" -}}
{{- printf "%s/extensions/rancher-secrets-manager/%s" .Values.baseUrl .Chart.AppVersion -}}
{{- end }}
