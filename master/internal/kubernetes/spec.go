package kubernetes

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/mount"
	petName "github.com/dustinkirkland/golang-petname"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"

	k8sV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (p *pod) configureResourcesRequirements() k8sV1.ResourceRequirements {
	return k8sV1.ResourceRequirements{
		Limits: map[k8sV1.ResourceName]resource.Quantity{
			"nvidia.com/gpu": *resource.NewQuantity(int64(p.gpus), resource.DecimalSI),
		},
		Requests: map[k8sV1.ResourceName]resource.Quantity{
			"nvidia.com/gpu": *resource.NewQuantity(int64(p.gpus), resource.DecimalSI),
		},
	}
}

func (p *pod) configureEnvVars(
	envVarsMap map[string]string,
	environment model.Environment,
	deviceType device.Type,
) ([]k8sV1.EnvVar, error) {
	for _, envVar := range environment.EnvironmentVariables.For(deviceType) {
		envVarSplit := strings.Split(envVar, "=")
		if len(envVarSplit) != 2 {
			return nil, errors.Errorf("unable to split envVar %s", envVar)
		}
		envVarsMap[envVarSplit[0]] = envVarSplit[1]
	}

	var slotIds []string
	for i := 0; i < p.gpus; i++ {
		slotIds = append(slotIds, strconv.Itoa(i))
	}

	envVarsMap["DET_CLUSTER_ID"] = p.clusterID
	envVarsMap["DET_MASTER"] = fmt.Sprintf("%s:%d", p.masterIP, p.masterPort)
	envVarsMap["DET_MASTER_HOST"] = p.masterIP
	envVarsMap["DET_MASTER_ADDR"] = p.masterIP
	envVarsMap["DET_MASTER_PORT"] = fmt.Sprintf("%d", p.masterPort)
	envVarsMap["DET_AGENT_ID"] = "k8agent"
	envVarsMap["DET_CONTAINER_ID"] = p.taskSpec.ContainerID
	envVarsMap["DET_SLOT_IDS"] = fmt.Sprintf("[%s]", strings.Join(slotIds, ","))
	envVarsMap["DET_USE_GPU"] = fmt.Sprintf("%t", p.gpus > 0)

	envVars := make([]k8sV1.EnvVar, 0, len(envVarsMap))
	for envVarKey, envVarValue := range envVarsMap {
		envVars = append(envVars, k8sV1.EnvVar{Name: envVarKey, Value: envVarValue})
	}

	return envVars, nil
}

func (p *pod) configureConfigMapSpec(runArchives []container.RunArchive) (*k8sV1.ConfigMap, error) {
	configMapData := make(map[string][]byte)
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
			Labels:    map[string]string{determinedLabel: p.taskSpec.TaskID},
		},
		BinaryData: configMapData,
	}, nil
}

func (p *pod) configureVolumes(
	ctx *actor.Context,
	dockerMounts []mount.Mount,
	runArchives []container.RunArchive,
) ([]k8sV1.VolumeMount, []k8sV1.VolumeMount, []k8sV1.Volume) {
	volumeMounts := make([]k8sV1.VolumeMount, 0)
	volumes := make([]k8sV1.Volume, 0)

	hostVolumeMounts, hostVolumes := dockerMountsToHostVolumes(dockerMounts)
	volumeMounts = append(volumeMounts, hostVolumeMounts...)
	volumes = append(volumes, hostVolumes...)

	shmVolumeMount, shmVolume := configureShmVolume(p.taskSpec.TaskContainerDefaults.ShmSizeBytes)
	volumeMounts = append(volumeMounts, shmVolumeMount)
	volumes = append(volumes, shmVolume)

	initContainerVolumeMounts, mainContainerRunArchiveVolumeMounts, runArchiveVolumes :=
		configureAdditionalFilesVolumes(p.configMapName, runArchives)

	volumeMounts = append(volumeMounts, mainContainerRunArchiveVolumeMounts...)
	volumes = append(volumes, runArchiveVolumes...)

	return initContainerVolumeMounts, volumeMounts, volumes
}

func (p *pod) configurePodSpec(
	ctx *actor.Context,
	volumes []k8sV1.Volume,
	initContainers []k8sV1.Container,
	containers []k8sV1.Container,
	podSpec *k8sV1.Pod,
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
	podSpec.ObjectMeta.Labels[determinedLabel] = p.taskSpec.TaskID

	if len(podSpec.Spec.Containers) > 0 {
		for k, v := range podSpec.Spec.Containers[0].Resources.Limits {
			if _, present := containers[0].Resources.Limits[k]; !present {
				containers[0].Resources.Limits[k] = v
			}
		}

		for k, v := range podSpec.Spec.Containers[0].Resources.Requests {
			if _, present := containers[0].Resources.Requests[k]; !present {
				containers[0].Resources.Requests[k] = v
			}
		}

		containers[0].VolumeMounts = append(
			containers[0].VolumeMounts, podSpec.Spec.Containers[0].VolumeMounts...)

		containers[0].VolumeDevices = append(
			containers[0].VolumeDevices, podSpec.Spec.Containers[0].VolumeDevices...)
	}

	podSpec.Spec.Volumes = append(podSpec.Spec.Volumes, volumes...)
	podSpec.Spec.HostNetwork = p.taskSpec.TaskContainerDefaults.NetworkMode.IsHost()
	podSpec.Spec.InitContainers = initContainers
	podSpec.Spec.Containers = containers
	podSpec.Spec.RestartPolicy = k8sV1.RestartPolicyNever

	return podSpec
}

func (p *pod) createPodSpecForTrial(ctx *actor.Context) error {
	exp := *p.taskSpec.StartContainer

	deviceType := device.CPU
	if p.gpus > 0 {
		deviceType = device.GPU
	}

	runArchives := tasks.TrialArchives(p.taskSpec)
	initContainerVolumeMounts, volumeMounts, volumes := p.configureVolumes(
		ctx, tasks.TrialDockerMounts(exp), runArchives)

	p.ports = []int{
		tasks.LocalRendezvousPort, tasks.LocalRendezvousPort + tasks.LocalRendezvousPortOffset}
	rendezvousPorts := []string{
		fmt.Sprintf("%d", p.ports[0]), fmt.Sprintf("%d", p.ports[1]),
	}

	envVars, err := p.configureEnvVars(
		tasks.TrialEnvVars(p.taskSpec, rendezvousPorts, 0),
		p.taskSpec.StartContainer.ExperimentConfig.Environment,
		deviceType,
	)
	if err != nil {
		return err
	}

	initContainers := []k8sV1.Container{
		configureInitContainer(
			len(runArchives),
			initContainerVolumeMounts,
			exp.ExperimentConfig.Environment.Image.For(deviceType),
			configureImagePullPolicy(exp.ExperimentConfig.Environment),
		),
	}

	containers := []k8sV1.Container{
		{
			Name:            "determined-trial",
			Command:         []string{"/run/determined/train/entrypoint.sh"},
			Image:           exp.ExperimentConfig.Environment.Image.For(deviceType),
			ImagePullPolicy: configureImagePullPolicy(exp.ExperimentConfig.Environment),
			SecurityContext: configureSecurityContext(exp.AgentUserGroup),
			Resources:       p.configureResourcesRequirements(),
			VolumeMounts:    volumeMounts,
			Env:             envVars,
			WorkingDir:      tasks.ContainerWorkDir,
		},
	}

	p.pod = p.configurePodSpec(
		ctx, volumes, initContainers, containers, exp.ExperimentConfig.Environment.PodSpec)

	p.configMap, err = p.configureConfigMapSpec(runArchives)
	if err != nil {
		return err
	}

	return nil
}

func (p *pod) createPodSpecForCommand(ctx *actor.Context) error {
	cmd := *p.taskSpec.StartCommand

	deviceType := device.CPU
	if p.gpus > 0 {
		deviceType = device.GPU
	}

	runArchives := tasks.CommandArchives(p.taskSpec)
	initContainerVolumeMounts, volumeMounts, volumes := p.configureVolumes(
		ctx, tasks.ToDockerMounts(cmd.Config.BindMounts), runArchives)

	for _, port := range cmd.Config.Environment.Ports {
		p.ports = append(p.ports, port)
	}

	envVars, err := p.configureEnvVars(
		tasks.CommandEnvVars(p.taskSpec),
		p.taskSpec.StartCommand.Config.Environment,
		deviceType,
	)
	if err != nil {
		return err
	}

	initContainers := []k8sV1.Container{
		configureInitContainer(
			len(runArchives),
			initContainerVolumeMounts,
			cmd.Config.Environment.Image.For(deviceType),
			configureImagePullPolicy(cmd.Config.Environment),
		),
	}

	containers := []k8sV1.Container{
		{
			Name:            "determined-task",
			Command:         cmd.Config.Entrypoint,
			Env:             envVars,
			Image:           cmd.Config.Environment.Image.For(deviceType),
			ImagePullPolicy: configureImagePullPolicy(cmd.Config.Environment),
			SecurityContext: configureSecurityContext(cmd.AgentUserGroup),
			Resources:       p.configureResourcesRequirements(),
			VolumeMounts:    volumeMounts,
			WorkingDir:      tasks.ContainerWorkDir,
		},
	}

	p.pod = p.configurePodSpec(
		ctx, volumes, initContainers, containers, cmd.Config.Environment.PodSpec)

	p.configMap, err = p.configureConfigMapSpec(runArchives)
	if err != nil {
		return err
	}

	return nil
}

func (p *pod) createPodSpecForGC(ctx *actor.Context) error {
	gcc := *p.taskSpec.GCCheckpoints

	deviceType := device.CPU
	if p.gpus > 0 {
		deviceType = device.GPU
	}

	runArchives := tasks.GCArchives(p.taskSpec)
	initContainerVolumeMounts, volumeMounts, volumes := p.configureVolumes(
		ctx, tasks.GCDockerMounts(gcc), runArchives)

	envVars, err := p.configureEnvVars(
		tasks.GCEnvVars(),
		p.taskSpec.GCCheckpoints.ExperimentConfig.Environment,
		deviceType,
	)
	if err != nil {
		return err
	}

	initContainers := []k8sV1.Container{
		configureInitContainer(
			len(runArchives),
			initContainerVolumeMounts,
			gcc.ExperimentConfig.Environment.Image.For(deviceType),
			configureImagePullPolicy(gcc.ExperimentConfig.Environment),
		),
	}

	containers := []k8sV1.Container{
		{
			Name:            "determined-gc",
			Command:         tasks.GCCmd(),
			Env:             envVars,
			Image:           gcc.ExperimentConfig.Environment.Image.For(deviceType),
			ImagePullPolicy: configureImagePullPolicy(gcc.ExperimentConfig.Environment),
			SecurityContext: configureSecurityContext(gcc.AgentUserGroup),
			Resources:       p.configureResourcesRequirements(),
			VolumeMounts:    volumeMounts,
			WorkingDir:      tasks.ContainerWorkDir,
		},
	}

	p.pod = p.configurePodSpec(
		ctx, volumes, initContainers, containers, gcc.ExperimentConfig.Environment.PodSpec)

	p.configMap, err = p.configureConfigMapSpec(runArchives)
	if err != nil {
		return err
	}

	return nil
}

func configureUniqueName(t tasks.TaskSpec) string {
	uniqueName := petName.Generate(2, "-")
	switch {
	case t.StartCommand != nil:
		return fmt.Sprintf("cmd-%s-%s", t.TaskID, uniqueName)
	case t.StartContainer != nil:
		return fmt.Sprintf(
			"exp-%d-trial-%d-%s",
			t.StartContainer.InitialWorkload.ExperimentID,
			t.StartContainer.InitialWorkload.TrialID,
			uniqueName,
		)
	case t.GCCheckpoints != nil:
		return fmt.Sprintf("gc-%s-%s", t.TaskID, uniqueName)
	default:
		return ""
	}
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

func configureImagePullPolicy(environment model.Environment) k8sV1.PullPolicy {
	pullPolicy := k8sV1.PullAlways
	if !environment.ForcePullImage {
		pullPolicy = k8sV1.PullIfNotPresent
	}
	return pullPolicy
}

func configureInitContainer(
	numArchives int,
	volumeMounts []k8sV1.VolumeMount,
	image string,
	imagePullPolicy k8sV1.PullPolicy,
) k8sV1.Container {
	return k8sV1.Container{
		Name:    "determined-init-container",
		Command: []string{path.Join(initContainerWorkDir, etc.K8InitContainerEntryScriptResource)},
		Args: []string{
			fmt.Sprintf("%d", numArchives), initContainerTarSrcPath, initContainerTarDstPath},
		Image:           image,
		ImagePullPolicy: imagePullPolicy,
		VolumeMounts:    volumeMounts,
		WorkingDir:      initContainerWorkDir,
	}
}
