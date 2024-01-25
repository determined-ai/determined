{{- define "determined.secretPath" -}}
/mount/determined/secrets/
{{- end -}}

{{- define "determined.masterPort" -}}
8081
{{- end -}}

{{- define "determined.dbHost" -}}
{{- if .Values.db.hostAddress }}{{ .Values.db.hostAddress }}{{- else }}determined-db-service-{{ .Release.Name }}{{- end -}}
{{- end -}}

{{- define "genai.PVCName" -}}
{{- if .Values.genai.sharedPVCName }}{{ .Values.genai.sharedPVCName }}{{ else }}genai-pvc-{{ .Release.Name }}{{ end }}
{{- end -}}
