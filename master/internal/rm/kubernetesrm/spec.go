package kubernetesrm

import (
	"context"
	"fmt"
	"math"
	"path"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	batchV1 "k8s.io/api/batch/v1"

	"github.com/docker/docker/api/types/mount"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/tasks"

	k8sV1 "k8s.io/api/core/v1"
	schedulingV1 "k8s.io/api/scheduling/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"

	alphaGatewayTyped "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

const (
	coscheduler = "coscheduler"

	initContainerTarSrcPath = "/run/determined/temp/tar/src"
	initContainerTarDstPath = "/run/determined/temp/tar/dst"
	initContainerWorkDir    = "/run/determined/temp/"

	gcTask            = "gc"
	cmdTask           = "cmd"
	labelPrefix       = "determined.ai/"
	userLabel         = labelPrefix + "user"
	workspaceLabel    = labelPrefix + "workspace"
	resourcePoolLabel = labelPrefix + "resource_pool"
	taskTypeLabel     = labelPrefix + "task_type"
	taskIDLabel       = labelPrefix + "task_id"
	allocationIDLabel = labelPrefix + "allocation_id"
	containerIDLabel  = labelPrefix + "container_id"
)

func (j *job) configureResourcesRequirements() k8sV1.ResourceRequirements {
	switch j.slotType {
	case device.CPU:
		cpuMillisRequested := int64(j.slotResourceRequests.CPU * float32(j.slotsPerPod) * 1000)
		return k8sV1.ResourceRequirements{
			Limits: map[k8sV1.ResourceName]resource.Quantity{
				"cpu": *resource.NewMilliQuantity(cpuMillisRequested, resource.DecimalSI),
			},
			Requests: map[k8sV1.ResourceName]resource.Quantity{
				"cpu": *resource.NewMilliQuantity(cpuMillisRequested, resource.DecimalSI),
			},
		}
	case device.ROCM:
		if j.slotsPerPod > 0 {
			return k8sV1.ResourceRequirements{
				Limits: map[k8sV1.ResourceName]resource.Quantity{
					resourceTypeAMD: *resource.NewQuantity(int64(j.slotsPerPod), resource.DecimalSI),
				},
				Requests: map[k8sV1.ResourceName]resource.Quantity{
					resourceTypeAMD: *resource.NewQuantity(int64(j.slotsPerPod), resource.DecimalSI),
				},
			}
		}

		return k8sV1.ResourceRequirements{
			Limits:   map[k8sV1.ResourceName]resource.Quantity{},
			Requests: map[k8sV1.ResourceName]resource.Quantity{},
		}
	case device.CUDA: // default to CUDA-backed slots.
		fallthrough
	default:
		// Don't request "nvidia.com/gpu=0" in zero slot case because then the job won't run on
		// CPU only nodes.
		if j.slotsPerPod > 0 {
			return k8sV1.ResourceRequirements{
				Limits: map[k8sV1.ResourceName]resource.Quantity{
					resourceTypeNvidia: *resource.NewQuantity(int64(j.slotsPerPod), resource.DecimalSI),
				},
				Requests: map[k8sV1.ResourceName]resource.Quantity{
					resourceTypeNvidia: *resource.NewQuantity(int64(j.slotsPerPod), resource.DecimalSI),
				},
			}
		}
		return k8sV1.ResourceRequirements{
			Limits:   map[k8sV1.ResourceName]resource.Quantity{},
			Requests: map[k8sV1.ResourceName]resource.Quantity{},
		}
	}
}

func (j *job) configureEnvVars(
	envVarsMap map[string]string,
	environment expconf.EnvironmentConfig,
	deviceType device.Type,
) ([]k8sV1.EnvVar, error) {
	for _, envVar := range environment.EnvironmentVariables().For(deviceType) {
		if key, val, found := strings.Cut(envVar, "="); found {
			envVarsMap[key] = val
		} else {
			envVarsMap[envVar] = ""
		}
	}

	var slotIDs []string
	for i := 0; i < j.slotsPerPod; i++ {
		slotIDs = append(slotIDs, strconv.Itoa(i))
	}

	masterScheme := "http"
	if j.masterTLSConfig.Enabled {
		masterScheme = "https"
	}

	// For multi rm support add an override to our defaulting logic because it is possible the
	// external cluster connects to master through a gateway with TLS
	// while the other does not use TLS.
	if j.masterScheme != "" {
		masterScheme = j.masterScheme
	}

	envVarsMap["DET_CLUSTER_ID"] = j.clusterID
	envVarsMap["DET_MASTER"] = fmt.Sprintf("%s://%s:%d", masterScheme, j.masterHost, j.masterPort)
	envVarsMap["DET_MASTER_HOST"] = j.masterHost
	envVarsMap["DET_MASTER_ADDR"] = j.masterHost
	envVarsMap["DET_MASTER_PORT"] = strconv.Itoa(int(j.masterPort))
	envVarsMap["DET_SLOT_IDS"] = fmt.Sprintf("[%s]", strings.Join(slotIDs, ","))
	if j.masterTLSConfig.CertificateName != "" {
		envVarsMap["DET_MASTER_CERT_NAME"] = j.masterTLSConfig.CertificateName
	}

	// Without this zero slot tasks will have access to all GPUs.
	// https://github.com/NVIDIA/k8s-device-plugin/issues/61
	if deviceType == device.CPU || deviceType == device.ZeroSlot {
		envVarsMap["NVIDIA_VISIBLE_DEVICES"] = "void"
	}

	envVarsMap["DET_KUBERNETES_JOB_PARALLELISM"] = strconv.Itoa(j.numPods)

	if j.internalTaskGWConfig != nil {
		envVarsMap["DET_PROXY_THROUGH_GATEWAY"] = "true"
	}

	envVars := make([]k8sV1.EnvVar, 0, len(envVarsMap))
	for envVarKey, envVarValue := range envVarsMap {
		envVars = append(envVars, k8sV1.EnvVar{Name: envVarKey, Value: envVarValue})
	}
	envVars = append(envVars, k8sV1.EnvVar{
		Name:      "DET_AGENT_ID",
		ValueFrom: &k8sV1.EnvVarSource{FieldRef: &k8sV1.ObjectFieldSelector{FieldPath: "spec.nodeName"}},
	})
	envVars = append(envVars, k8sV1.EnvVar{
		Name: "DET_KUBERNETES_POD_IP",
		ValueFrom: &k8sV1.EnvVarSource{
			FieldRef: &k8sV1.ObjectFieldSelector{
				FieldPath: "status.podIP",
			},
		},
	})
	return envVars, nil
}

// proxyResourceGenerator returns a configured list of proxy resources given a set of ports.
// We do this lazily to make port selection easier. If we created the resource before it is
// added to the request queue we risk that port being taken later on.
type proxyResourceGenerator func([]int) []gatewayProxyResource

func (j *job) configureProxyResources(t *tasks.TaskSpec) proxyResourceGenerator {
	if j.internalTaskGWConfig == nil {
		return nil
	}

	generator := proxyResourceGenerator(func(ports []int) []gatewayProxyResource {
		var resources []gatewayProxyResource
		if len(ports) != len(j.req.ProxyPorts) {
			panic("proxy ports and ports must be the same length")
		}

		for i, proxyPort := range j.req.ProxyPorts {
			sharedName := fmt.Sprintf("%s-%d", j.jobName, i)

			gwPort := ports[i]
			allocLabels := map[string]string{
				determinedLabel: t.AllocationID,
			}
			annotations := map[string]string{
				jobNameAnnotation: j.jobName,
			}

			serviceSpec := &k8sV1.Service{
				ObjectMeta: metaV1.ObjectMeta{
					Name:        sharedName,
					Namespace:   j.namespace,
					Labels:      allocLabels,
					Annotations: annotations,
				},
				Spec: k8sV1.ServiceSpec{
					Ports: []k8sV1.ServicePort{
						{
							Protocol: k8sV1.ProtocolTCP,
							Port:     int32(proxyPort.Port),
						},
					},
					Selector: allocLabels,
					Type:     k8sV1.ServiceTypeClusterIP,
				},
			}

			tcpRouteSpec := &alphaGatewayTyped.TCPRoute{
				ObjectMeta: metaV1.ObjectMeta{
					Name:        sharedName,
					Namespace:   j.namespace,
					Labels:      allocLabels,
					Annotations: annotations,
				},
				Spec: alphaGatewayTyped.TCPRouteSpec{
					CommonRouteSpec: alphaGatewayTyped.CommonRouteSpec{
						ParentRefs: []alphaGatewayTyped.ParentReference{
							{
								Namespace: ptrs.Ptr(alphaGatewayTyped.Namespace(j.internalTaskGWConfig.GatewayNamespace)),
								Name:      alphaGatewayTyped.ObjectName(j.internalTaskGWConfig.GatewayName),
								Port:      ptrs.Ptr(alphaGatewayTyped.PortNumber(gwPort)),
								SectionName: ptrs.Ptr(alphaGatewayTyped.SectionName(
									generateListenerName(gwPort),
								)),
							},
						},
					},
					Rules: []alphaGatewayTyped.TCPRouteRule{
						{
							BackendRefs: []alphaGatewayTyped.BackendRef{
								{
									BackendObjectReference: alphaGatewayTyped.BackendObjectReference{
										Name: alphaGatewayTyped.ObjectName(serviceSpec.Name),
										Kind: ptrs.Ptr(alphaGatewayTyped.Kind("Service")),
										Port: ptrs.Ptr(alphaGatewayTyped.PortNumber(proxyPort.Port)),
									},
								},
							},
						},
					},
				},
			}

			gatewayListener := createListenerForPod(gwPort)

			resources = append(resources, gatewayProxyResource{
				serviceSpec:     serviceSpec,
				tcpRouteSpec:    tcpRouteSpec,
				gatewayListener: gatewayListener,
			})
		}
		return resources
	})

	return generator
}

func (j *job) configureConfigMapSpec(
	taskSpec *tasks.TaskSpec,
	runArchives []cproto.RunArchive,
) (*k8sV1.ConfigMap, error) {
	configMapData := make(map[string][]byte, len(runArchives))
	// Add additional files as tar.gz archive.
	for idx, runArchive := range runArchives {
		zippedArchive, err := archive.ToTarGz(runArchive.Archive)
		if err != nil {
			return nil, errors.Wrap(err, "failed to zip archive")
		}
		configMapData[fmt.Sprintf("%d.tar.gz", idx)] = zippedArchive
	}

	// Add initContainer script.
	configMapData[etc.K8InitContainerEntryScriptResource] = etc.MustStaticFile(
		etc.K8InitContainerEntryScriptResource)

	// Create configMap of AdditionalFiles as .tar.gz archive and the entrypoint script
	// for the init container.
	return &k8sV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      j.configMapName,
			Namespace: j.namespace,
			Labels:    map[string]string{determinedLabel: taskSpec.AllocationID},
		},
		BinaryData: configMapData,
	}, nil
}

func (j *job) configureVolumes(
	taskSpec *tasks.TaskSpec,
	dockerMounts []mount.Mount,
	runArchives []cproto.RunArchive,
) ([]k8sV1.VolumeMount, []k8sV1.VolumeMount, []k8sV1.Volume) {
	volumeMounts := make([]k8sV1.VolumeMount, 0)
	volumes := make([]k8sV1.Volume, 0)

	hostVolumeMounts, hostVolumes := dockerMountsToHostVolumes(dockerMounts)
	volumeMounts = append(volumeMounts, hostVolumeMounts...)
	volumes = append(volumes, hostVolumes...)

	shmSize := taskSpec.ShmSize
	if shmSize == 0 {
		shmSize = taskSpec.TaskContainerDefaults.ShmSizeBytes
	}
	shmVolumeMount, shmVolume := configureShmVolume(shmSize)
	volumeMounts = append(volumeMounts, shmVolumeMount)
	volumes = append(volumes, shmVolume)

	// //nolint:lll // There isn't a great way to break this line that makes it more readable.
	initContainerVolumeMounts, mainContainerRunArchiveVolumeMounts, runArchiveVolumes := configureAdditionalFilesVolumes(
		j.configMapName,
		runArchives,
	)

	volumeMounts = append(volumeMounts, mainContainerRunArchiveVolumeMounts...)
	volumes = append(volumes, runArchiveVolumes...)

	return initContainerVolumeMounts, volumeMounts, volumes
}

func (j *job) modifyPodSpec(
	taskSpec *tasks.TaskSpec,
	newPod *k8sV1.Pod,
	scheduler string,
) {
	if taskSpec.Description == cmdTask {
		return
	}

	if taskSpec.Description == gcTask {
		if newPod.Spec.PriorityClassName != "" {
			log.Warnf(
				"GC Priority is currently using priority class: %s. "+
					"It will be reset to determined-system-priority",
				newPod.Spec.PriorityClassName,
			)
		}
		newPod.Spec.PriorityClassName = "determined-system-priority"
	} else if scheduler == coscheduler {
		if newPod.Spec.SchedulerName == "" {
			newPod.Spec.SchedulerName = scheduler
		}
		j.configureCoscheduler(taskSpec, newPod, scheduler)
	}

	if newPod.Spec.PriorityClassName == "" &&
		taskSpec.ResourcesConfig.Priority() != nil {
		priority := int32(*taskSpec.ResourcesConfig.Priority())
		name := fmt.Sprintf("%s-priorityclass", taskSpec.ContainerID)

		err := j.createPriorityClass(name, priority)

		if err == nil {
			newPod.Spec.PriorityClassName = name
		}
	} else if newPod.Spec.PriorityClassName == "" {
		newPod.Spec.PriorityClassName = "determined-medium-priority"
	}
}

func addNodeDisabledAffinityToPodSpec(pod *k8sV1.Pod, clusterID string) {
	addNodeSelectorRequirement(pod, k8sV1.NodeSelectorRequirement{
		Key:      clusterID,
		Operator: k8sV1.NodeSelectorOpDoesNotExist,
	}, addOnLabel)

	// TODO once k8s supports
	// RequiredDuringSchedulingRequiredDuringExecution
	// we can add two node affininties for noExecuteNodeLabel and noScheduleNodeLabel
	// so we can skip the step in k8s disable where we kill everything in non drain.
}

func addDisallowedNodesToPodSpec(req *sproto.AllocateRequest, pod *k8sV1.Pod) {
	// Can't just replace []string{nodeName} with
	// logpattern.DisallowedNodes(taskID).ToSlice() and not loop
	// because of the k8s error given "Required value:
	// must be only one value when `operator` is 'In' or 'NotIn' for node field selector".
	for _, nodeName := range req.BlockedNodes {
		addNodeSelectorRequirement(pod, k8sV1.NodeSelectorRequirement{
			Key:      "metadata.name",
			Operator: k8sV1.NodeSelectorOpNotIn,
			Values:   []string{nodeName},
		}, addOnField)
	}
}

const (
	addOnLabel = true
	addOnField = false
)

func addNodeSelectorRequirement(
	pod *k8sV1.Pod, req k8sV1.NodeSelectorRequirement, onLabel bool,
) {
	if pod.Spec.Affinity == nil {
		pod.Spec.Affinity = &k8sV1.Affinity{}
	}
	if pod.Spec.Affinity.NodeAffinity == nil {
		pod.Spec.Affinity.NodeAffinity = &k8sV1.NodeAffinity{}
	}
	nodeAffinity := pod.Spec.Affinity.NodeAffinity

	if nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
		nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &k8sV1.NodeSelector{}
	}
	nodeSelector := nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution

	if len(nodeSelector.NodeSelectorTerms) == 0 {
		nodeSelector.NodeSelectorTerms = append(nodeSelector.NodeSelectorTerms,
			k8sV1.NodeSelectorTerm{})
	}

	reqs := nodeSelector.NodeSelectorTerms[0].MatchFields
	if onLabel {
		reqs = nodeSelector.NodeSelectorTerms[0].MatchExpressions
	}

	// Make function idempotent.
	for _, r := range reqs {
		if reflect.DeepEqual(r, req) {
			return
		}
	}

	if onLabel {
		nodeSelector.NodeSelectorTerms[0].MatchExpressions = append(
			nodeSelector.NodeSelectorTerms[0].MatchExpressions, req)
	} else {
		nodeSelector.NodeSelectorTerms[0].MatchFields = append(
			nodeSelector.NodeSelectorTerms[0].MatchFields, req)
	}
}

func (j *job) configureCoscheduler(
	taskSpec *tasks.TaskSpec,
	newPod *k8sV1.Pod,
	scheduler string,
) {
	if newPod.Spec.SchedulerName != scheduler {
		return
	}

	resources := taskSpec.ResourcesConfig
	minAvailable := 0

	if j.slotType == device.CUDA && j.slotsPerPod > 0 {
		minAvailable = int(math.Ceil(float64(resources.SlotsPerTrial()) / float64(j.slotsPerPod)))
	}

	if newPod.APIVersion == "" {
		newPod.APIVersion = "v1"
	}
	if newPod.Kind == "" {
		newPod.Kind = "Pod" //nolint:goconst
	}

	_, ok := newPod.ObjectMeta.Labels["pod-group.scheduling.sigs.k8s.io/name"]
	if !ok {
		newPod.ObjectMeta.Labels["pod-group.scheduling.sigs.k8s.io/name"] = j.jobName
	}
	_, ok = newPod.ObjectMeta.Labels["pod-group.scheduling.sigs.k8s.io/min-available"]
	if !ok {
		newPod.ObjectMeta.Labels["pod-group.scheduling.sigs.k8s.io/min-available"] = strconv.Itoa(
			minAvailable)
	}
}

var defaultTTLSecondsAfterFinished int32 = 15 * 60 // 15 minutes

func (j *job) createPriorityClass(name string, priority int32) error {
	preemptionPolicy := k8sV1.PreemptNever

	_, err := j.clientSet.SchedulingV1().PriorityClasses().Create(context.TODO(),
		&schedulingV1.PriorityClass{
			TypeMeta: metaV1.TypeMeta{},
			ObjectMeta: metaV1.ObjectMeta{
				Name:      name,
				Namespace: "",
			},
			Value:            priority,
			GlobalDefault:    false,
			Description:      "temporary priorityClass for determined",
			PreemptionPolicy: &preemptionPolicy,
		}, metaV1.CreateOptions{})

	return err
}

const (
	maxChars             int    = 63
	fmtAlphaNumeric      string = "A-Za-z0-9"
	fmtAllowedChars      string = fmtAlphaNumeric + `\.\-_`
	defaultPodLabelValue string = "invalid_value"
)

var (
	regDisallowedSpecialChars  = regexp.MustCompile("[^" + fmtAllowedChars + "]")
	regLeadingNonAlphaNumeric  = regexp.MustCompile("^[^" + fmtAlphaNumeric + "]+")
	regTrailingNonAlphaNumeric = regexp.MustCompile("[^" + fmtAlphaNumeric + "]+$")
)

func validatePodLabelValue(value string) (string, error) {
	errs := validation.IsValidLabelValue(value)
	if len(errs) == 0 {
		return value, nil
	}

	// Label value is not valid; attempt to fix it.
	// 0. Convert dis-allowed special characters to underscore.
	fixedValue := regDisallowedSpecialChars.ReplaceAllString(value, "_")

	// 1. Strip leading non-alphanumeric characters.
	fixedValue = regLeadingNonAlphaNumeric.ReplaceAllString(fixedValue, "")

	// 2. Truncate to 63 characters.
	if len(fixedValue) > maxChars {
		fixedValue = fixedValue[:maxChars]
	}

	// 3. Strip ending non-alphanumeric characters.
	fixedValue = regTrailingNonAlphaNumeric.ReplaceAllString(fixedValue, "")

	log.Debugf(
		"conform to Kubernetes pod label value standards: reformatting %s to %s",
		value, fixedValue,
	)

	// Final validation check, return error if still not valid for safety.
	errs = validation.IsValidLabelValue(fixedValue)
	if len(errs) != 0 {
		return "", errors.New("pod label value is not valid")
	}

	return fixedValue, nil
}

func (j *job) configureJobSpec(
	taskSpec *tasks.TaskSpec,
	volumes []k8sV1.Volume,
	determinedInitContainers k8sV1.Container,
	determinedContainer k8sV1.Container,
	sidecarContainers []k8sV1.Container,
	podSpec *k8sV1.Pod,
	scheduler string,
) *batchV1.Job {
	if podSpec == nil {
		podSpec = &k8sV1.Pod{}
	} else {
		podSpec = podSpec.DeepCopy()
	}

	podSpec.ObjectMeta.Name = j.jobName
	podSpec.ObjectMeta.Namespace = j.namespace
	if podSpec.ObjectMeta.Labels == nil {
		podSpec.ObjectMeta.Labels = make(map[string]string)
	}
	if taskSpec.Owner != nil {
		// Owner label will disappear if Owner is somehow nil.
		labelValue, err := validatePodLabelValue(taskSpec.Owner.Username)
		if err != nil {
			labelValue = defaultPodLabelValue
			log.Warnf("unable to reformat username=%s to Kubernetes standards; using %s",
				taskSpec.Owner.Username, labelValue)
		}
		podSpec.ObjectMeta.Labels[userLabel] = labelValue
	}

	labelValue, err := validatePodLabelValue(taskSpec.Workspace)
	if err != nil {
		labelValue = defaultPodLabelValue
		log.Warnf("unable to reformat workspace=%s to Kubernetes standards; using %s",
			taskSpec.Workspace, labelValue)
	}
	podSpec.ObjectMeta.Labels[workspaceLabel] = labelValue

	labelValue, err = validatePodLabelValue(j.req.ResourcePool)
	if err != nil {
		labelValue = defaultPodLabelValue
		log.Warnf("unable to reformat resource_pool=%s to Kubernetes standards; using %s",
			j.req.ResourcePool, labelValue)
	}
	podSpec.ObjectMeta.Labels[resourcePoolLabel] = labelValue

	podSpec.ObjectMeta.Labels[taskTypeLabel] = string(taskSpec.TaskType)
	podSpec.ObjectMeta.Labels[taskIDLabel] = taskSpec.TaskID
	podSpec.ObjectMeta.Labels[containerIDLabel] = taskSpec.ContainerID
	podSpec.ObjectMeta.Labels[determinedLabel] = taskSpec.AllocationID
	podSpec.ObjectMeta.Labels[allocationIDLabel] = taskSpec.AllocationID

	// If map is not populated, labels will be missing and observability will be impacted.
	for k, v := range taskSpec.ExtraPodLabels {
		labelValue, err := validatePodLabelValue(v)
		if err != nil {
			labelValue = defaultPodLabelValue
		}
		podSpec.ObjectMeta.Labels[labelPrefix+k] = labelValue
	}

	j.modifyPodSpec(taskSpec, podSpec, scheduler)

	addNodeDisabledAffinityToPodSpec(podSpec, clusterIDNodeLabel())
	addDisallowedNodesToPodSpec(j.req, podSpec)

	nonDeterminedContainers := make([]k8sV1.Container, 0)
	for idx, container := range podSpec.Spec.Containers {
		if container.Name != model.DeterminedK8ContainerName {
			nonDeterminedContainers = append(nonDeterminedContainers, container)
			continue
		}

		determinedContainer.Env = append(determinedContainer.Env, container.Env...)
		determinedContainer.EnvFrom = append(determinedContainer.EnvFrom, container.EnvFrom...)

		for k, v := range podSpec.Spec.Containers[idx].Resources.Limits {
			if _, present := determinedContainer.Resources.Limits[k]; !present {
				determinedContainer.Resources.Limits[k] = v
			}
		}

		for k, v := range podSpec.Spec.Containers[idx].Resources.Requests {
			if _, present := determinedContainer.Resources.Requests[k]; !present {
				determinedContainer.Resources.Requests[k] = v
			}
		}

		determinedContainer.VolumeMounts = append(
			determinedContainer.VolumeMounts, podSpec.Spec.Containers[idx].VolumeMounts...)

		determinedContainer.VolumeDevices = append(
			determinedContainer.VolumeDevices, podSpec.Spec.Containers[idx].VolumeDevices...)
	}

	podSpec.Spec.Containers = nonDeterminedContainers
	podSpec.Spec.Containers = append(podSpec.Spec.Containers, sidecarContainers...)
	podSpec.Spec.Containers = append(podSpec.Spec.Containers, determinedContainer)
	podSpec.Spec.Volumes = append(podSpec.Spec.Volumes, volumes...)
	podSpec.Spec.HostNetwork = taskSpec.TaskContainerDefaults.NetworkMode.IsHost()
	podSpec.Spec.InitContainers = append(podSpec.Spec.InitContainers, determinedInitContainers)
	podSpec.Spec.RestartPolicy = k8sV1.RestartPolicyNever
	podSpec.ObjectMeta.Namespace = j.namespace

	return &batchV1.Job{
		ObjectMeta: podSpec.ObjectMeta,
		Spec: batchV1.JobSpec{
			Parallelism:  ptrs.Ptr(int32(j.numPods)),
			Completions:  ptrs.Ptr(int32(j.numPods)),
			BackoffLimit: ptrs.Ptr(int32(0)),
			Template: k8sV1.PodTemplateSpec{
				ObjectMeta: podSpec.ObjectMeta,
				Spec:       podSpec.Spec,
			},
			// TTLSeconds is useful for debugging but also must be set reasonably high so we
			// can recover job exit codes in the case where the job exits while the master
			// is down.
			TTLSecondsAfterFinished: &defaultTTLSecondsAfterFinished,
		},
	}
}

func (j *job) createSpec(scheduler string, taskSpec *tasks.TaskSpec) (*batchV1.Job, *k8sV1.ConfigMap, error) {
	deviceType := j.slotType
	// Device type is currently configured globally on KubernetesResourceManagerConfig.
	// So we special case certain functionality to use device.CPU.
	if deviceType == device.ZeroSlot || j.slotsPerPod == 0 {
		deviceType = device.CPU
	}

	runArchives, rootArchives := taskSpec.Archives()

	initContainerVolumeMounts, volumeMounts, volumes := j.configureVolumes(taskSpec, taskSpec.Mounts, runArchives)

	env := taskSpec.Environment

	// This array containerPorts is set on the container spec.
	// This field on the container spec is for "primarily informational"
	// reasons and to allow us to read these ports in reattaching pods.
	var containerPorts []k8sV1.ContainerPort
	for _, port := range env.Ports() {
		containerPorts = append(containerPorts, k8sV1.ContainerPort{
			ContainerPort: int32(port),
		})
	}

	envVars, err := j.configureEnvVars(taskSpec.EnvVars(), env, deviceType)
	if err != nil {
		return nil, nil, err
	}

	initContainer := configureInitContainer(
		len(runArchives),
		initContainerVolumeMounts,
		env.Image().For(deviceType),
		configureImagePullPolicy(env),
		taskSpec.AgentUserGroup,
	)

	var sidecars []k8sV1.Container

	container := k8sV1.Container{
		Name:            model.DeterminedK8ContainerName,
		Command:         taskSpec.LogShipperWrappedEntrypoint(),
		Env:             envVars,
		Image:           env.Image().For(deviceType),
		ImagePullPolicy: configureImagePullPolicy(env),
		SecurityContext: getDetContainerSecurityContext(
			taskSpec.AgentUserGroup,
			env.PodSpec(),
		),
		Resources:    j.configureResourcesRequirements(),
		VolumeMounts: volumeMounts,
		WorkingDir:   taskSpec.WorkDir,
		Ports:        containerPorts,
	}

	configMapSpec, err := j.configureConfigMapSpec(taskSpec, runArchives)
	if err != nil {
		return nil, nil, err
	}

	rootVolumes, rootVolumeMounts, err := handleRootArchiveFiles(rootArchives, configMapSpec)
	if err != nil {
		return nil, nil, err
	}
	volumes = append(volumes, rootVolumes...)
	container.VolumeMounts = append(container.VolumeMounts, rootVolumeMounts...)

	return j.configureJobSpec(
		taskSpec,
		volumes,
		initContainer,
		container,
		sidecars,
		(*k8sV1.Pod)(env.PodSpec()),
		scheduler,
	), configMapSpec, nil
}

func configureUniqueName(t tasks.TaskSpec) string {
	name := t.Description

	// Prefix with a cluster ID so multiple Determined installations can coexist within cluster. But
	// limit to the first 8 chars of the cluster ID to avoid the 63 character limit (this is ~53).
	// Handle short cluster IDs for tests.
	var clusterIDPrefix string
	if len(t.ClusterID) >= 8 {
		clusterIDPrefix = t.ClusterID[:8]
	} else {
		clusterIDPrefix = t.ClusterID
	}
	if clusterIDPrefix != "" {
		// Starting with clusterID is not a valid DNS name since it could be a number sometimes.
		name = fmt.Sprintf("det-%s-%s", clusterIDPrefix, name)
	}

	return name
}

func configureSecurityContext(agentUserGroup *model.AgentUserGroup) *k8sV1.SecurityContext {
	if agentUserGroup != nil {
		userID := int64(agentUserGroup.UID)
		groupID := int64(agentUserGroup.GID)
		return &k8sV1.SecurityContext{
			RunAsUser:  &userID,
			RunAsGroup: &groupID,
		}
	}

	return nil
}

func getDetContainerSecurityContext(
	agentUserGroup *model.AgentUserGroup,
	podSpec *expconf.PodSpec,
) *k8sV1.SecurityContext {
	securityContext := configureSecurityContext(agentUserGroup)

	if podSpec != nil {
		for _, container := range podSpec.Spec.Containers {
			if container.Name == model.DeterminedK8ContainerName {
				userInput := container.SecurityContext
				if userInput == nil {
					userInput = &k8sV1.SecurityContext{}
				}

				// Use det user link-with-agent-user to configure RunAsUser
				// and/or RunAsGroup. We disallow this in security context.
				if securityContext != nil {
					userInput.RunAsUser = securityContext.RunAsUser
					userInput.RunAsGroup = securityContext.RunAsGroup
				} else {
					userInput.RunAsUser = nil
					userInput.RunAsGroup = nil
				}
				return userInput
			}
		}
	}

	return securityContext
}

func configureImagePullPolicy(environment expconf.EnvironmentConfig) k8sV1.PullPolicy {
	pullPolicy := k8sV1.PullAlways
	if !environment.ForcePullImage() {
		pullPolicy = k8sV1.PullIfNotPresent
	}
	return pullPolicy
}

func configureInitContainer(
	numArchives int,
	volumeMounts []k8sV1.VolumeMount,
	image string,
	imagePullPolicy k8sV1.PullPolicy,
	agentUserGroup *model.AgentUserGroup,
) k8sV1.Container {
	return k8sV1.Container{
		Name:    "determined-init-container",
		Command: []string{path.Join(initContainerWorkDir, etc.K8InitContainerEntryScriptResource)},
		Args: []string{
			strconv.Itoa(numArchives), initContainerTarSrcPath, initContainerTarDstPath,
		},
		Image:           image,
		ImagePullPolicy: imagePullPolicy,
		VolumeMounts:    volumeMounts,
		WorkingDir:      initContainerWorkDir,
		SecurityContext: configureSecurityContext(agentUserGroup),
	}
}

func handleRootArchiveFiles(
	rootArchives []cproto.RunArchive,
	cm *k8sV1.ConfigMap,
) ([]k8sV1.Volume, []k8sV1.VolumeMount, error) {
	rootPathsToKeys := make(map[string][]k8sV1.KeyToPath)
	for _, a := range rootArchives {
		for _, item := range a.Archive {
			base := item.BaseName()
			if _, ok := cm.BinaryData[base]; ok {
				return nil, nil, fmt.Errorf(
					"multiple rooted files have same file name %s",
					item.Path,
				)
			}
			cm.BinaryData[base] = item.Content

			dir := item.DirName()
			rootPathsToKeys[dir] = append(rootPathsToKeys[dir], k8sV1.KeyToPath{
				Key:  base,
				Path: base,
				Mode: ptrs.Ptr(int32(item.FileMode)),
			})
		}
	}

	var volumes []k8sV1.Volume
	var volumeMounts []k8sV1.VolumeMount
	i := 0
	for dir, keys := range rootPathsToKeys {
		volumeName := fmt.Sprintf("root-volume-%d", i)
		i++
		volumes = append(volumes, k8sV1.Volume{
			Name: volumeName,
			VolumeSource: k8sV1.VolumeSource{
				ConfigMap: &k8sV1.ConfigMapVolumeSource{
					LocalObjectReference: k8sV1.LocalObjectReference{
						Name: cm.Name,
					},
					Items: keys,
				},
			},
		})

		volumeMounts = append(volumeMounts, k8sV1.VolumeMount{
			Name:      volumeName,
			MountPath: dir,
			ReadOnly:  true, // Assume root files will be read only.
		})
	}
	return volumes, volumeMounts, nil
}
