{{- define "determined.secretPath" -}}
/mount/determined/secrets/
{{- end -}}

{{- define "determined.masterPort" -}}
8081
{{- end -}}

{{- define "determined.cpuPodSpec" -}}
spec:
  priorityClassName: determined-system-priority
  enableServiceLinks: false
  containers:
  - name: determined-container
    resources:
      requests:
        memory: 64Gi
        cpu: 16
      limits:
        memory: 64Gi
        cpu: 16
    volumeMounts:
      - mountPath: /dev/shm
        name: dshm
      {{- range .Values.mounts }}
      - name: {{ regexReplaceAll "[_]" .pvc "-" | lower }}
        mountPath: {{ .name }}
      {{- end }}
  volumes:
    - name: dshm
      emptyDir:
        medium: Memory
    {{- range .Values.mounts }}
    - name: {{ regexReplaceAll "[_]" .pvc "-" | lower }}
      persistentVolumeClaim:
        claimName: {{ .pvc }}
    {{- end }}
{{- end -}}

{{- define "determined.gpuPodSpecRTX_A4000" -}}
spec:
  priorityClassName: determined-system-priority
  enableServiceLinks: false
  affinity:
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
          - matchExpressions:
            - key: topology.kubernetes.io/region
              operator: In
              values:
                - {{ .Values.region | upper }}
            - key: gpu.nvidia.com/class
              operator: In
              values:
                - RTX_A4000
  containers:
  - name: determined-container
    resources:
      requests:
        memory: 200Gi
        cpu: 32
      limits:
        memory: 200Gi
        cpu: 32
    volumeMounts:
      - mountPath: /dev/shm
        name: dshm
      {{- range .Values.mounts }}
      - name: {{ regexReplaceAll "[_]" .pvc "-" | lower }}
        mountPath: {{ .name }}
      {{- end }}
  volumes:
    - name: dshm
      emptyDir:
        medium: Memory
    {{- range .Values.mounts }}
    - name: {{ regexReplaceAll "[_]" .pvc "-" | lower }}
      persistentVolumeClaim:
        claimName: {{ .pvc }}
    {{- end }}
{{- end -}}

{{- define "determined.gpuPodSpecRTX_A5000" -}}
spec:
  priorityClassName: determined-system-priority
  enableServiceLinks: false
  affinity:
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
          - matchExpressions:
            - key: topology.kubernetes.io/region
              operator: In
              values:
                - {{ .Values.region | upper }}
            - key: gpu.nvidia.com/class
              operator: In
              values:
                - RTX_A5000
  containers:
  - name: determined-container
    resources:
      requests:
        memory: 200Gi
        cpu: 32
      limits:
        memory: 200Gi
        cpu: 32
    volumeMounts:
      - mountPath: /dev/shm
        name: dshm
      {{- range .Values.mounts }}
      - name: {{ regexReplaceAll "[_]" .pvc "-" | lower }}
        mountPath: {{ .name }}
      {{- end }}
  volumes:
    - name: dshm
      emptyDir:
        medium: Memory
    {{- range .Values.mounts }}
    - name: {{ regexReplaceAll "[_]" .pvc "-" | lower }}
      persistentVolumeClaim:
        claimName: {{ .pvc }}
    {{- end }}
{{- end -}}

{{- define "determined.gpuPodSpecA40" -}}
spec:
  priorityClassName: determined-system-priority
  enableServiceLinks: false
  affinity:
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
          - matchExpressions:
            - key: topology.kubernetes.io/region
              operator: In
              values:
                - {{ .Values.region | upper }}
            - key: gpu.nvidia.com/class
              operator: In
              values:
                - A40
  containers:
  - name: determined-container
    resources:
      requests:
        memory: 32Gi
        cpu: 8
      limits:
        memory: 32Gi
        cpu: 8
    volumeMounts:
      - mountPath: /dev/shm
        name: dshm
      {{- range .Values.mounts }}
      - name: {{ regexReplaceAll "[_]" .pvc "-" | lower }}
        mountPath: {{ .name }}
      {{- end }}
  volumes:
    - name: dshm
      emptyDir:
        medium: Memory
    {{- range .Values.mounts }}
    - name: {{ regexReplaceAll "[_]" .pvc "-" | lower }}
      persistentVolumeClaim:
        claimName: {{ .pvc }}
    {{- end }}
{{- end -}}

