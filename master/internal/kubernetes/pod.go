package kubernetes

import (
	"fmt"
	"path"
	"reflect"
	"strconv"
	"strings"

	"github.com/determined-ai/determined/master/pkg/container"

	"github.com/determined-ai/determined/master/pkg/etc"

	"github.com/determined-ai/determined/master/pkg/archive"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	initContainerTarSrcPath = "/run/temp/determined/tar/src"
	initContainerTarDstPath = "/run/temp/determined/tar/dst"
	initContainerWorkDir    = "/run/temp/determined/"
	hostNetwork             = "host"
)

type pod struct {
	cluster            *actor.Ref
	clientSet          *k8sclient.Clientset
	namespace          string
	masterIP           string
	masterPort         int32
	taskSpec           tasks.TaskSpec
	gpus               int
	rank               int
	podInterface       typedV1.PodInterface
	configMapInterface typedV1.ConfigMapInterface
	pod                *v1.Pod
	configMaps         []*v1.ConfigMap
}

func newPod(
	cluster *actor.Ref,
	clientSet *k8sclient.Clientset,
	namespace string,
	masterIP string,
	masterPort int32,
	taskSpec tasks.TaskSpec,
	gpus int,
	rank int,
	podInterface typedV1.PodInterface,
	configMapInterface typedV1.ConfigMapInterface,
) *pod {
	return &pod{
		cluster:            cluster,
		clientSet:          clientSet,
		namespace:          namespace,
		masterIP:           masterIP,
		masterPort:         masterPort,
		taskSpec:           taskSpec,
		gpus:               gpus,
		rank:               rank,
		podInterface:       podInterface,
		configMapInterface: configMapInterface,
		configMaps:         make([]*v1.ConfigMap, 0),
	}
}

func (p *pod) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		if err := p.startPod(ctx); err != nil {
			return err
		}

	default:
		ctx.Log().Error("unexpected message: ", reflect.TypeOf(msg))
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

func (p *pod) startPod(ctx *actor.Context) error {
	switch {
	case p.taskSpec.StartCommand != nil:
		return p.startPodForCommand(ctx)
	case p.taskSpec.StartContainer != nil:
		return p.startPodForTrial(ctx)
	case p.taskSpec.GCCheckpoints != nil:
		return p.startPodForGC(ctx)
	default:
		return errors.Errorf("unexpected task spec received")
	}
}

func (p *pod) configureResourcesRequirements() v1.ResourceRequirements {
	return v1.ResourceRequirements{
		Limits: map[v1.ResourceName]resource.Quantity{
			"nvidia.com/gpu": *resource.NewQuantity(int64(p.gpus), resource.DecimalSI),
		},
	}
}

func (p *pod) configureEnvVars(
	envVarsMap map[string]string,
	environment model.Environment,
	deviceType device.Type,
) ([]v1.EnvVar, error) {
	// TODO (DET-3457): Include env variables set in experiment config.
	if len(environment.EnvironmentVariables.For(deviceType)) > 0 {
		return nil, errors.Errorf(
			"kubernetes resource provider does not currently support environment " +
				"variables set in the experiment config, use startup-hook.sh instead.")
	}

	var slotIds []string
	for i := 0; i < p.gpus; i++ {
		slotIds = append(slotIds, strconv.Itoa(i))
	}

	envVarsMap["DET_CLUSTER_ID"] = "k8cluster"
	envVarsMap["DET_MASTER"] = fmt.Sprintf("%s:%d", p.masterIP, p.masterPort)
	envVarsMap["DET_MASTER_HOST"] = p.masterIP
	envVarsMap["DET_MASTER_ADDR"] = p.masterIP
	envVarsMap["DET_MASTER_PORT"] = fmt.Sprintf("%d", p.masterPort)
	envVarsMap["DET_AGENT_ID"] = "k8agent"
	envVarsMap["DET_CONTAINER_ID"] = p.taskSpec.ContainerID
	envVarsMap["DET_SLOT_IDS"] = strings.Join(slotIds, ",")
	envVarsMap["DET_USE_GPU"] = fmt.Sprintf("%t", p.gpus > 0)

	envVars := make([]v1.EnvVar, 0, len(envVarsMap))
	for envVarKey, envVarValue := range envVarsMap {
		envVars = append(envVars, v1.EnvVar{Name: envVarKey, Value: envVarValue})
	}

	return envVars, nil
}

func (p *pod) configureRunArchives(
	ctx *actor.Context,
	podName string,
	runArchives []container.RunArchive,
) ([]v1.VolumeMount, []v1.VolumeMount, []v1.Volume, error) {
	zippedArchives := make(map[string][]byte)
	for idx, runArchive := range runArchives {
		zippedArchive, errZip := archive.ToTarGz(runArchive.Archive)
		if errZip != nil {
			return nil, nil, nil, errors.Wrap(errZip, "failed to zip archive")
		}
		zippedArchives[fmt.Sprintf("%d.tar.gz", idx)] = zippedArchive
	}

	// Create configMap of AdditionalFiles as .tar.gz archive.
	archiveConfigMap, err := startConfigMap(
		ctx,
		createConfigMapSpec(podName, zippedArchives, p.namespace),
		p.configMapInterface,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	p.configMaps = append(p.configMaps, archiveConfigMap)

	// Create a configMap for the executable for un-taring.
	initContainerEntrypointArchive := map[string][]byte{
		etc.InitContainerEntryScriptResource: etc.MustStaticFile(etc.InitContainerEntryScriptResource),
	}
	initContainerEntrypointConfigMap, err := startConfigMap(
		ctx,
		createConfigMapSpec(podName, initContainerEntrypointArchive, p.namespace),
		p.configMapInterface,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	p.configMaps = append(p.configMaps, initContainerEntrypointConfigMap)

	initContainerVolumeMounts, mainContainerVolumeMounts, initContainerVolumes :=
		configureAdditionalFilesVolumes(
			archiveConfigMap, initContainerEntrypointConfigMap, runArchives)

	return initContainerVolumeMounts, mainContainerVolumeMounts, initContainerVolumes, nil
}

func (p *pod) launchPod(ctx *actor.Context, podSpec *v1.Pod) error {
	var err error
	p.pod, err = p.podInterface.Create(podSpec)
	if err != nil {
		return errors.Wrap(err, "error creating pod")
	}
	ctx.Log().Infof("Created pod %s.", p.pod.Name)

	return nil
}

func (p *pod) startPodForTrial(ctx *actor.Context) error {
	exp := *p.taskSpec.StartContainer
	podName := fmt.Sprintf(
		"exp-%d-trial-%d-%d",
		exp.InitialWorkload.ExperimentID,
		exp.InitialWorkload.TrialID, p.rank,
	)

	deviceType := device.CPU
	if p.gpus > 0 {
		deviceType = device.GPU
	}

	volumeMounts := make([]v1.VolumeMount, 0)
	volumes := make([]v1.Volume, 0)

	hostVolumeMounts, hostVolumes := dockerMountsToHostVolumes(
		tasks.ConfigureTrialDockerMounts(exp))
	volumeMounts = append(volumeMounts, hostVolumeMounts...)
	volumes = append(volumes, hostVolumes...)

	shmVolumeMount, shmVolume := configureShmVolume(p.taskSpec.TaskContainerDefaults.ShmSizeBytes)
	volumeMounts = append(volumeMounts, shmVolumeMount)
	volumes = append(volumes, shmVolume)

	rendezvousPorts := []string{
		fmt.Sprintf("%d", tasks.LocalRendezvousPort),
		fmt.Sprintf("%d", tasks.LocalRendezvousPort+tasks.LocalRendezvousPortOffset),
	}
	envVars, err := p.configureEnvVars(
		tasks.ConfigureTrialEnvVars(p.taskSpec, rendezvousPorts),
		p.taskSpec.StartContainer.ExperimentConfig.Environment,
		deviceType,
	)
	if err != nil {
		return err
	}

	runArchives := tasks.ConfigureTrialArchives(p.taskSpec)
	initContainerVolumeMounts, mainContainerVolumeMounts, initContainerVolumes, err :=
		p.configureRunArchives(ctx, podName, runArchives)
	if err != nil {
		return err
	}
	volumeMounts = append(volumeMounts, mainContainerVolumeMounts...)
	volumes = append(volumes, initContainerVolumes...)

	initContainers := []v1.Container{
		configureInitContainer(
			len(runArchives),
			initContainerVolumeMounts,
			exp.ExperimentConfig.Environment.Image.For(deviceType),
			configureImagePullPolicy(exp.ExperimentConfig.Environment),
		),
	}

	containers := []v1.Container{
		{
			Name:            "determined-trial",
			Command:         []string{"/run/determined/workdir/entrypoint.sh"},
			Image:           exp.ExperimentConfig.Environment.Image.For(deviceType),
			ImagePullPolicy: configureImagePullPolicy(exp.ExperimentConfig.Environment),
			SecurityContext: configureSecurityContext(exp.AgentUserGroup),
			Resources:       p.configureResourcesRequirements(),
			VolumeMounts:    volumeMounts,
			Env:             envVars,
			WorkingDir:      tasks.ContainerWorkDir,
		},
	}

	podSpec := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: p.namespace,
			Labels:    map[string]string{"determined": p.taskSpec.TaskID},
		},
		Spec: v1.PodSpec{
			Volumes:        volumes,
			HostNetwork:    p.taskSpec.TaskContainerDefaults.NetworkMode == hostNetwork,
			InitContainers: initContainers,
			Containers:     containers,
			RestartPolicy:  v1.RestartPolicyNever,
		},
	}

	return p.launchPod(ctx, podSpec)
}

func (p *pod) startPodForCommand(ctx *actor.Context) error {
	cmd := *p.taskSpec.StartCommand
	podName := fmt.Sprintf("cmd-%s", p.taskSpec.TaskID)

	deviceType := device.CPU
	if p.gpus > 0 {
		deviceType = device.GPU
	}

	volumeMounts := make([]v1.VolumeMount, 0)
	volumes := make([]v1.Volume, 0)

	hostVolumeMounts, hostVolumes := dockerMountsToHostVolumes(
		tasks.ToDockerMounts(cmd.Config.BindMounts))
	volumeMounts = append(volumeMounts, hostVolumeMounts...)
	volumes = append(volumes, hostVolumes...)

	shmVolumeMount, shmVolume := configureShmVolume(p.taskSpec.TaskContainerDefaults.ShmSizeBytes)
	volumeMounts = append(volumeMounts, shmVolumeMount)
	volumes = append(volumes, shmVolume)

	envVars, err := p.configureEnvVars(
		tasks.ConfigureCommandEnvVars(p.taskSpec),
		p.taskSpec.StartCommand.Config.Environment,
		deviceType,
	)
	if err != nil {
		return err
	}

	runArchives := tasks.ConfigureCommandArchives(p.taskSpec)
	initContainerVolumeMounts, mainContainerVolumeMounts, initContainerVolumes, err :=
		p.configureRunArchives(ctx, podName, runArchives)
	if err != nil {
		return err
	}
	volumeMounts = append(volumeMounts, mainContainerVolumeMounts...)
	volumes = append(volumes, initContainerVolumes...)

	initContainers := []v1.Container{
		configureInitContainer(
			len(runArchives),
			initContainerVolumeMounts,
			cmd.Config.Environment.Image.For(deviceType),
			configureImagePullPolicy(cmd.Config.Environment),
		),
	}

	containers := []v1.Container{
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

	podSpec := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: p.namespace,
			Labels:    map[string]string{"determined": p.taskSpec.TaskID},
		},
		Spec: v1.PodSpec{
			Volumes:        volumes,
			HostNetwork:    p.taskSpec.TaskContainerDefaults.NetworkMode == hostNetwork,
			InitContainers: initContainers,
			Containers:     containers,
			RestartPolicy:  v1.RestartPolicyNever,
		},
	}

	return p.launchPod(ctx, podSpec)
}

func (p *pod) startPodForGC(ctx *actor.Context) error {
	gcc := *p.taskSpec.GCCheckpoints
	podName := fmt.Sprintf("gc-%s", p.taskSpec.TaskID)

	deviceType := device.CPU
	if p.gpus > 0 {
		deviceType = device.GPU
	}

	volumeMounts := make([]v1.VolumeMount, 0)
	volumes := make([]v1.Volume, 0)

	hostVolumeMounts, hostVolumes := dockerMountsToHostVolumes(
		tasks.ConfigureGCDockerMounts(gcc))
	volumeMounts = append(volumeMounts, hostVolumeMounts...)
	volumes = append(volumes, hostVolumes...)

	envVars, err := p.configureEnvVars(
		tasks.ConfigureGCEnvVars(),
		p.taskSpec.StartContainer.ExperimentConfig.Environment,
		deviceType,
	)
	if err != nil {
		return err
	}

	runArchives := tasks.ConfigureGCArchives(p.taskSpec)
	initContainerVolumeMounts, mainContainerVolumeMounts, initContainerVolumes, err :=
		p.configureRunArchives(ctx, podName, runArchives)
	if err != nil {
		return err
	}
	volumeMounts = append(volumeMounts, mainContainerVolumeMounts...)
	volumes = append(volumes, initContainerVolumes...)

	initContainers := []v1.Container{
		configureInitContainer(
			len(runArchives),
			initContainerVolumeMounts,
			gcc.ExperimentConfig.Environment.Image.For(deviceType),
			configureImagePullPolicy(gcc.ExperimentConfig.Environment),
		),
	}

	containers := []v1.Container{
		{
			Name:            "determined-gc",
			Command:         tasks.ConfigureGCCmd(),
			Env:             envVars,
			Image:           gcc.ExperimentConfig.Environment.Image.For(deviceType),
			ImagePullPolicy: configureImagePullPolicy(gcc.ExperimentConfig.Environment),
			SecurityContext: configureSecurityContext(gcc.AgentUserGroup),
			Resources:       p.configureResourcesRequirements(),
			VolumeMounts:    volumeMounts,
			WorkingDir:      tasks.ContainerWorkDir,
		},
	}

	podSpec := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: p.namespace,
			Labels:    map[string]string{"determined": p.taskSpec.TaskID},
		},
		Spec: v1.PodSpec{
			Volumes:        volumes,
			HostNetwork:    p.taskSpec.TaskContainerDefaults.NetworkMode == hostNetwork,
			InitContainers: initContainers,
			Containers:     containers,
			RestartPolicy:  v1.RestartPolicyNever,
		},
	}

	return p.launchPod(ctx, podSpec)
}

func configureSecurityContext(agentUserGroup *model.AgentUserGroup) *v1.SecurityContext {
	if agentUserGroup != nil {
		userID := int64(agentUserGroup.ID)
		groupID := int64(agentUserGroup.GID)
		return &v1.SecurityContext{
			RunAsUser:  &userID,
			RunAsGroup: &groupID,
		}
	}

	return nil
}

func configureImagePullPolicy(environment model.Environment) v1.PullPolicy {
	pullPolicy := v1.PullAlways
	if !environment.ForcePullImage {
		pullPolicy = v1.PullIfNotPresent
	}
	return pullPolicy
}

func configureInitContainer(
	numArchives int,
	volumeMounts []v1.VolumeMount,
	image string,
	imagePullPolicy v1.PullPolicy,
) v1.Container {
	return v1.Container{
		Name:    "determined-init-container",
		Command: []string{path.Join(initContainerWorkDir, etc.InitContainerEntryScriptResource)},
		Args: []string{
			fmt.Sprintf("%d", numArchives), initContainerTarSrcPath, initContainerTarDstPath},
		Image:           image,
		ImagePullPolicy: imagePullPolicy,
		VolumeMounts:    volumeMounts,
		WorkingDir:      initContainerWorkDir,
	}
}
