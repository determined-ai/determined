{{- define "determined.secretPath" -}}
/mount/determined/secrets/
{{- end -}}

{{- define "determined.masterPort" -}}
8081
{{- end -}}

{{- define "genai.PVCName" -}}
{{- if .Values.genai.sharedPVCName }}{{ .Values.genai.sharedPVCName }}{{ else }}genai-pvc-{{ .Release.Name }}{{ end }}
{{- end -}}
