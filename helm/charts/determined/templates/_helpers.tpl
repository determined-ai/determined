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

{{- define "determined.dbCertVolumeMount" -}}
{{- if .Values.db.certResourceName -}}
- name: database-cert
  mountPath: {{ include "determined.secretPath" . }}
  readOnly: true
{{- end }}
{{- end -}}

{{- define "determined.dbCertVolume" }}
{{- if .Values.db.sslMode -}}
- name: database-cert
  {{- $resourceType := (required "A valid .Values.db.resourceType entry required!" .Values.db.resourceType | trim)}}
  {{- if eq $resourceType "configMap"}}
  configMap:
    name: {{ required  "A valid Values.db.certResourceName entry is required!" .Values.db.certResourceName }}
  {{- else }}
  secret:
    secretName: {{ required  "A valid Values.db.certResourceName entry is required!" .Values.db.certResourceName }}
  {{- end }}
{{- end }}
{{- end }}

{{- define "genai.PVCName" -}}
    {{- if .Values.genai.sharedPVCName }}
        {{- .Values.genai.sharedPVCName }}
    {{- else }}
        {{- "genai-pvc-" }}{{ .Release.Name }}
    {{- end -}}
{{- end -}}

{{- define "genai.sharedFSMountPath" -}}
    {{- if .Values.genai.sharedFSMountPath -}}
        {{- .Values.genai.sharedFSMountPath }}
    {{- else }}
        {{- "/run/determined/workdir/shared_fs" }}
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

{{- /* Necessary because of the way that useNodePortForMaster makes a LoadBalancer that only allows */ -}}
{{- /* https connections through */ -}}
{{- define "genai.detMasterScheme" -}}
    {{- if (and (not .Values.useNodePortForMaster) .Values.tlsSecret) }}
        {{- "https" }}
    {{- else }}
        {{- "http" }}
    {{- end }}
{{- end }}
