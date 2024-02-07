{{- define "determined.secretPath" -}}
/mount/determined/secrets/
{{- end -}}

{{- define "determined.masterPort" -}}
8081
{{- end -}}

{{- define "determined.dbHost" -}}
    {{- if .Values.db.hostAddress }}
        {{- .Values.db.hostAddress }}
    {{- else }}
        {{- "determined-db-service-" }}{{ .Release.Name }}
    {{- end -}}
{{- end -}}

{{- define "genai.PVCName" -}}
    {{- if .Values.genai.sharedPVCName }}
        {{- .Values.genai.sharedPVCName }}
    {{- else }}
        {{- "genai-pvc-" }}{{ .Release.Name }}
    {{- end -}}
{{- end -}}

{{- define "genai.allResourcePoolNames" -}}
    {{- $orig_resource_pool_data := (required "A valid .Values.resourcePools entry required!" .Values.resourcePools) }}
    {{- $resource_pools := list -}}
    {{- range $v := $orig_resource_pool_data }}
        {{- $resource_pools = mustAppend $resource_pools $v.pool_name }}
    {{- end }}
    {{ toJson $resource_pools }}
{{- end }}
