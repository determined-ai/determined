package kubernetesrm

import (
	"context"
	"fmt"
	"math"
	"path"
	"strconv"
	"strings"

	"github.com/determined-ai/determined/master/internal/config"

	"github.com/docker/docker/api/types/mount"
	petName "github.com/dustinkirkland/golang-petname"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

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
)

const (
	coscheduler = "coscheduler"

	gcTask  = "gc"
	cmdTask = "cmd"
)

func (p *pod) configureResourcesRequirements() k8sV1.ResourceRequirements {
	switch p.slotType {
	case device.CPU:
		cpuMillisRequested := int64(p.slotResourceRequests.CPU * float32(p.slots) * 1000)
		return k8sV1.ResourceRequirements{
			Limits: map[k8sV1.ResourceName]resource.Quantity{
				"cpu": *resource.NewMilliQuantity(cpuMillisRequested, resource.DecimalSI),
			},
			Requests: map[k8sV1.ResourceName]resource.Quantity{
				"cpu": *resource.NewMilliQuantity(cpuMillisRequested, resource.DecimalSI),
			},
		}
	case device.ROCM:
		panic("ROCm is not supported on k8s yet")
	case device.CUDA: // default to CUDA-backed slots.
		fallthrough
	default:
		if p.slots > 0 {
			return k8sV1.ResourceRequirements{
				Limits: map[k8sV1.ResourceName]resource.Quantity{
					ResourceTypeNvidia: *resource.NewQuantity(int64(p.slots), resource.DecimalSI),
				},
				Requests: map[k8sV1.ResourceName]resource.Quantity{
					ResourceTypeNvidia: *resource.NewQuantity(int64(p.slots), resource.DecimalSI),
				},
			}
		}
		return k8sV1.ResourceRequirements{
			Limits:   map[k8sV1.ResourceName]resource.Quantity{},
			Requests: map[k8sV1.ResourceName]resource.Quantity{},
		}
	}
}

func (p *pod) configureEnvVars(
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

	var slotIds []string
	for i := 0; i < p.slots; i++ {
		slotIds = append(slotIds, strconv.Itoa(i))
	}

	masterScheme := "http"
	if p.masterTLSConfig.Enabled {
		masterScheme = "https"
	}
	envVarsMap["DET_CLUSTER_ID"] = p.clusterID
	envVarsMap["DET_MASTER"] = fmt.Sprintf("%s://%s:%d", masterScheme, p.masterIP, p.masterPort)
	envVarsMap["DET_MASTER_HOST"] = p.masterIP
	envVarsMap["DET_MASTER_ADDR"] = p.masterIP
	envVarsMap["DET_MASTER_PORT"] = fmt.Sprintf("%d", p.masterPort)
	envVarsMap["DET_AGENT_ID"] = "k8agent"
	envVarsMap["DET_SLOT_IDS"] = fmt.Sprintf("[%s]", strings.Join(slotIds, ","))
	if p.masterTLSConfig.CertificateName != "" {
		envVarsMap["DET_MASTER_CERT_NAME"] = p.masterTLSConfig.CertificateName
	}

	// Without this zero slot tasks will have access to all GPUs.
	// https://github.com/NVIDIA/k8s-device-plugin/issues/61
	if deviceType == device.CPU || deviceType == device.ZeroSlot {
		envVarsMap["NVIDIA_VISIBLE_DEVICES"] = "void"
	}

	envVars := make([]k8sV1.EnvVar, 0, len(envVarsMap))
	for envVarKey, envVarValue := range envVarsMap {
		envVars = append(envVars, k8sV1.EnvVar{Name: envVarKey, Value: envVarValue})
	}

	return envVars, nil
}

func (p *pod) configureConfigMapSpec(
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
			Name:      p.configMapName,
			Namespace: p.namespace,
			Labels:    map[string]string{determinedLabel: p.submissionInfo.taskSpec.AllocationID},
		},
		BinaryData: configMapData,
	}, nil
}

func (p *pod) configureLoggingVolumes() ([]k8sV1.VolumeMount, []k8sV1.Volume) {
	logsVolumeName := "det-logs"
	mounts := []k8sV1.VolumeMount{
		{
			Name:      logsVolumeName,
			MountPath: "/run/determined/train/logs",
		},
	}
	volumes := []k8sV1.Volume{
		{
			Name: logsVolumeName,
			VolumeSource: k8sV1.VolumeSource{EmptyDir: &k8sV1.EmptyDirVolumeSource{
				Medium: k8sV1.StorageMediumMemory,
			}},
		},
	}
	return mounts, volumes
}

func (p *pod) configureVolumes(
	dockerMounts []mount.Mount,
	runArchives []cproto.RunArchive,
) ([]k8sV1.VolumeMount, []k8sV1.VolumeMount, []k8sV1.Volume) {
	volumeMounts := make([]k8sV1.VolumeMount, 0)
	volumes := make([]k8sV1.Volume, 0)

	hostVolumeMounts, hostVolumes := dockerMountsToHostVolumes(dockerMounts)
	volumeMounts = append(volumeMounts, hostVolumeMounts...)
	volumes = append(volumes, hostVolumes...)

	shmSize := p.submissionInfo.taskSpec.ShmSize
	if shmSize == 0 {
		shmSize = p.submissionInfo.taskSpec.TaskContainerDefaults.ShmSizeBytes
	}
	shmVolumeMount, shmVolume := configureShmVolume(shmSize)
	volumeMounts = append(volumeMounts, shmVolumeMount)
	volumes = append(volumes, shmVolume)

	// //nolint:lll // There isn't a great way to break this line that makes it more readable.
	initContainerVolumeMounts, mainContainerRunArchiveVolumeMounts, runArchiveVolumes := configureAdditionalFilesVolumes(
		p.configMapName,
		runArchives,
	)

	volumeMounts = append(volumeMounts, mainContainerRunArchiveVolumeMounts...)
	volumes = append(volumes, runArchiveVolumes...)

	return initContainerVolumeMounts, volumeMounts, volumes
}

func (p *pod) modifyPodSpec(newPod *k8sV1.Pod, scheduler string) {
	if p.submissionInfo.taskSpec.Description == cmdTask {
		return
	}

	if p.submissionInfo.taskSpec.Description == gcTask {
		if newPod.Spec.PriorityClassName != "" {
			log.Warnf(
				"GC Priority is currently using priority class: %s. "+
					"It will be reset to determined-system-priority",
				newPod.Spec.PriorityClassName,
			)
		}
		newPod.Spec.PriorityClassName = "determined-system-priority"
	} else if scheduler == coscheduler || scheduler == config.PreemptionScheduler {
		if newPod.Spec.SchedulerName == "" {
			newPod.Spec.SchedulerName = scheduler
		}
		p.configureCoscheduler(newPod, scheduler)
	}

	if newPod.Spec.PriorityClassName == "" &&
		p.submissionInfo.taskSpec.ResourcesConfig.Priority() != nil {
		priority := int32(*p.submissionInfo.taskSpec.ResourcesConfig.Priority())
		name := fmt.Sprintf("%s-priorityclass", p.submissionInfo.taskSpec.ContainerID)

		err := p.createPriorityClass(name, priority)

		if err == nil {
			newPod.Spec.PriorityClassName = name
		}
	} else if newPod.Spec.PriorityClassName == "" {
		newPod.Spec.PriorityClassName = "determined-medium-priority"
	}
}

func (p *pod) configureCoscheduler(newPod *k8sV1.Pod, scheduler string) {
	if newPod.Spec.SchedulerName != scheduler {
		return
	}

	resources := p.submissionInfo.taskSpec.ResourcesConfig
	minAvailable := 0

	if p.slotType == device.CUDA && p.slots > 0 {
		minAvailable = int(math.Ceil(float64(resources.SlotsPerTrial()) / float64(p.slots)))
	}

	if newPod.APIVersion == "" {
		newPod.APIVersion = "v1"
	}
	if newPod.Kind == "" {
		newPod.Kind = "Pod"
	}

	_, ok := newPod.ObjectMeta.Labels["pod-group.scheduling.sigs.k8s.io/name"]
	if !ok {
		newPod.ObjectMeta.Labels["pod-group.scheduling.sigs.k8s.io/name"] = trialNameFromPod(
			p.podName,
		)
	}
	_, ok = newPod.ObjectMeta.Labels["pod-group.scheduling.sigs.k8s.io/min-available"]
	if !ok {
		newPod.ObjectMeta.Labels["pod-group.scheduling.sigs.k8s.io/min-available"] = strconv.Itoa(
			minAvailable)
	}
}

func (p *pod) createPriorityClass(name string, priority int32) error {
	preemptionPolicy := k8sV1.PreemptNever

	_, err := p.clientSet.SchedulingV1().PriorityClasses().Create(context.TODO(),
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

func (p *pod) configurePodSpec(
	volumes []k8sV1.Volume,
	determinedInitContainers k8sV1.Container,
	determinedContainer k8sV1.Container,
	sidecarContainers []k8sV1.Container,
	podSpec *k8sV1.Pod,
	scheduler string,
) *k8sV1.Pod {
	if podSpec == nil {
		podSpec = &k8sV1.Pod{}
	} else {
		podSpec = podSpec.DeepCopy()
	}

	podSpec.ObjectMeta.Name = p.podName
	podSpec.ObjectMeta.Namespace = p.namespace
	if podSpec.ObjectMeta.Labels == nil {
		podSpec.ObjectMeta.Labels = make(map[string]string)
	}
	podSpec.ObjectMeta.Labels[determinedLabel] = p.submissionInfo.taskSpec.AllocationID

	p.modifyPodSpec(podSpec, scheduler)

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
	podSpec.Spec.HostNetwork = p.submissionInfo.taskSpec.TaskContainerDefaults.NetworkMode.IsHost()
	podSpec.Spec.InitContainers = append(podSpec.Spec.InitContainers, determinedInitContainers)
	podSpec.Spec.RestartPolicy = k8sV1.RestartPolicyNever

	return podSpec
}

func (p *pod) createPodSpec(scheduler string) error {
	deviceType := p.slotType
	// Device type is currently configured globally on KubernetesResourceManagerConfig.
	// So we special case certain functionality to use device.CPU.
	if deviceType == device.ZeroSlot || p.slots == 0 {
		deviceType = device.CPU
	}

	spec := p.submissionInfo.taskSpec

	runArchives, rootArchives := spec.Archives()

	initContainerVolumeMounts, volumeMounts, volumes := p.configureVolumes(spec.Mounts, runArchives)

	env := spec.Environment

	// This array containerPorts is set on the container spec.
	// This field on the container spec is for "primarily informational"
	// reasons and to allow us to read these ports in reattaching pods.
	var containerPorts []k8sV1.ContainerPort
	for _, port := range env.Ports() {
		p.ports = append(p.ports, port)
		containerPorts = append(containerPorts, k8sV1.ContainerPort{
			ContainerPort: int32(port),
		})
	}

	envVars, err := p.configureEnvVars(spec.EnvVars(), env, deviceType)
	if err != nil {
		return err
	}

	initContainer := configureInitContainer(
		len(runArchives),
		initContainerVolumeMounts,
		env.Image().For(deviceType),
		configureImagePullPolicy(env),
		spec.AgentUserGroup,
	)

	var sidecars []k8sV1.Container

	envVars = append(envVars, k8sV1.EnvVar{Name: "DET_K8S_LOG_TO_FILE", Value: "true"})

	loggingMounts, loggingVolumes := p.configureLoggingVolumes()

	volumes = append(volumes, loggingVolumes...)
	volumeMounts = append(volumeMounts, loggingMounts...)

	container := k8sV1.Container{
		Name:            model.DeterminedK8ContainerName,
		Command:         spec.Entrypoint,
		Env:             envVars,
		Image:           env.Image().For(deviceType),
		ImagePullPolicy: configureImagePullPolicy(env),
		SecurityContext: getDetContainerSecurityContext(
			spec.AgentUserGroup,
			env.PodSpec(),
		),
		Resources:    p.configureResourcesRequirements(),
		VolumeMounts: volumeMounts,
		WorkingDir:   spec.WorkDir,
		Ports:        containerPorts,
	}

	p.configMap, err = p.configureConfigMapSpec(runArchives)
	if err != nil {
		return err
	}

	rootVolumes, rootVolumeMounts, err := handleRootArchiveFiles(rootArchives, p.configMap)
	if err != nil {
		return err
	}
	volumes = append(volumes, rootVolumes...)
	container.VolumeMounts = append(container.VolumeMounts, rootVolumeMounts...)

	p.pod = p.configurePodSpec(
		volumes, initContainer, container, sidecars, (*k8sV1.Pod)(env.PodSpec()), scheduler)
	return nil
}

func configureUniqueName(t tasks.TaskSpec, rank int) string {
	return fmt.Sprintf("%s-%d-%s-%s",
		t.Description, rank, t.AllocationID, petName.Generate(2, "-"))
}

func trialNameFromPod(podName string) string {
	// Given a pod name of the form exp-#-trial-#-rank-#..., returns a string exp#trial#
	// e.g. input: exp-1-trial-1-rank-0-71af9..., returns: exp1trial1

	newName := ""
	for i, v := range strings.Split(podName, "-") {
		if i > 3 {
			break
		}
		newName += v
	}
	return newName
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
			fmt.Sprintf("%d", numArchives), initContainerTarSrcPath, initContainerTarDstPath,
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
