package kubernetes

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/docker/docker/api/types/mount"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	initContainerTarSrcPath = "/run/determined/temp/tar/src"
	initContainerTarDstPath = "/run/determined/temp/tar/dst"
	initContainerWorkDir    = "/run/determined/temp/"
	determinedLabel         = "determined"
)

type pod struct {
	cluster            *actor.Ref
	taskHandler        *actor.Ref
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
	podName            string
	logStreamer        *actor.Ref
	container          container.Container
	ports              []int
}

func newPod(
	cluster *actor.Ref,
	taskHandler *actor.Ref,
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
		taskHandler:        taskHandler,
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
		podName:            configurePodName(taskSpec, rank),
		ports:              make([]int, 0),
	}
}

func (p *pod) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		if err := p.startPod(ctx); err != nil {
			return err
		}

	case podStatusUpdate:
		if err := p.receivePodStatusUpdate(ctx, msg); err != nil {
			return err
		}

	case sproto.ContainerLog:
		p.receiveContainerLogs(ctx, msg)

	case actor.PostStop:
		if p.logStreamer != nil {
			p.logStreamer.Stop()
		}

	default:
		ctx.Log().Errorf("unexpected message %T", msg)
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

func (p *pod) receivePodStatusUpdate(ctx *actor.Context, msg podStatusUpdate) error {
	p.pod = msg.updatedPod

	switch msg.updatedPod.Status.Phase {
	case v1.PodPending:
		containerState := getContainerState(msg.updatedPod.Status.Conditions)
		if containerState != p.container.State {
			if containerState == container.Starting {
				// Kubernetes does not have an explicit state for pulling container
				// images. We insert it here because our  current implementation of
				// the trial actor requires it.
				ctx.Log().Infof(
					"transitioning pod state from %s to %s for pod %s",
					p.container.State, container.Pulling, p.podName)
				p.container = p.container.Transition(container.Pulling)

				rsc := sproto.ContainerStateChanged{Container: p.container}
				ctx.Tell(p.taskHandler, rsc)
			}

			ctx.Log().Infof(
				"transitioning pod state from %s to %s for pod %s",
				p.container.State, containerState, p.podName)
			p.container = p.container.Transition(containerState)

			// TODO: Refactor the containerStarted part of this message
			// to be less specific to agents.
			rsc := sproto.ContainerStateChanged{Container: p.container}
			ctx.Tell(p.taskHandler, rsc)
		}

	case v1.PodRunning:
		if p.container.State != container.Running {
			p.container = p.container.Transition(container.Running)

			var ok bool
			p.logStreamer, ok = ctx.ActorOf(
				fmt.Sprintf("%s-logs", p.podName),
				newPodLogStreamer(p.podInterface, p.podName, ctx.Self()))

			if !ok {
				return errors.Errorf("log streamer already exists")
			}

			ctx.Tell(p.logStreamer, streamLogs{})

			ctx.Tell(p.taskHandler, sproto.ContainerStateChanged{Container: p.container})
			ctx.Tell(p.cluster, sproto.PodStarted{
				ContainerID: p.container.ID,
				IP:          p.pod.Status.PodIP,
				Ports:       p.ports,
			})
		}

	case v1.PodFailed:
	case v1.PodSucceeded:
	default:
		return errors.Errorf(
			"unexpected pod status %s for pod %s", msg.updatedPod.Status.Phase, p.podName)
	}

	return nil
}

func (p *pod) receiveContainerLogs(ctx *actor.Context, msg sproto.ContainerLog) {
	ctx.Tell(p.taskHandler, sproto.ContainerLog{
		Container:   p.container,
		Timestamp:   msg.Timestamp,
		PullMessage: msg.PullMessage,
		RunMessage:  msg.RunMessage,
		AuxMessage:  msg.AuxMessage,
	})
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
				"variables set in the experiment config; use startup-hook.sh instead")
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
	envVarsMap["DET_SLOT_IDS"] = fmt.Sprintf("[%s]", strings.Join(slotIds, ","))
	envVarsMap["DET_USE_GPU"] = fmt.Sprintf("%t", p.gpus > 0)

	envVars := make([]v1.EnvVar, 0, len(envVarsMap))
	for envVarKey, envVarValue := range envVarsMap {
		envVars = append(envVars, v1.EnvVar{Name: envVarKey, Value: envVarValue})
	}

	return envVars, nil
}

func (p *pod) configureRunArchives(
	ctx *actor.Context,
	runArchives []container.RunArchive,
) ([]v1.VolumeMount, []v1.VolumeMount, []v1.Volume, error) {
	tarredArchives := make(map[string][]byte)
	for idx, runArchive := range runArchives {
		zippedArchive, errZip := archive.ToTarGz(runArchive.Archive)
		if errZip != nil {
			return nil, nil, nil, errors.Wrap(errZip, "failed to zip archive")
		}
		tarredArchives[fmt.Sprintf("%d.tar.gz", idx)] = zippedArchive
	}

	// Create configMap of AdditionalFiles as .tar.gz archive.
	archiveConfigMap, err := startConfigMap(
		ctx,
		createConfigMapSpec(p.podName, tarredArchives, p.namespace),
		p.configMapInterface,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	p.configMaps = append(p.configMaps, archiveConfigMap)

	// Create a configMap for the executable for un-taring.
	initContainerEntrypointArchive := map[string][]byte{
		etc.K8InitContainerEntryScriptResource: etc.MustStaticFile(
			etc.K8InitContainerEntryScriptResource),
	}
	initContainerEntrypointConfigMap, err := startConfigMap(
		ctx,
		createConfigMapSpec(p.podName, initContainerEntrypointArchive, p.namespace),
		p.configMapInterface,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	p.configMaps = append(p.configMaps, initContainerEntrypointConfigMap)

	initContainerVolumeMounts, mainContainerVolumeMounts, volumes :=
		configureAdditionalFilesVolumes(
			archiveConfigMap, initContainerEntrypointConfigMap, runArchives)

	return initContainerVolumeMounts, mainContainerVolumeMounts, volumes, nil
}

func (p *pod) configureVolumes(
	ctx *actor.Context,
	dockerMounts []mount.Mount,
	runArchives []container.RunArchive,
) ([]v1.VolumeMount, []v1.VolumeMount, []v1.Volume, error) {
	volumeMounts := make([]v1.VolumeMount, 0)
	volumes := make([]v1.Volume, 0)

	hostVolumeMounts, hostVolumes := dockerMountsToHostVolumes(dockerMounts)
	volumeMounts = append(volumeMounts, hostVolumeMounts...)
	volumes = append(volumes, hostVolumes...)

	shmVolumeMount, shmVolume := configureShmVolume(p.taskSpec.TaskContainerDefaults.ShmSizeBytes)
	volumeMounts = append(volumeMounts, shmVolumeMount)
	volumes = append(volumes, shmVolume)

	initContainerVolumeMounts, mainContainerRunArchiveVolumeMounts, runArchiveVolumes, err :=
		p.configureRunArchives(ctx, runArchives)
	if err != nil {
		return nil, nil, nil, err
	}
	volumeMounts = append(volumeMounts, mainContainerRunArchiveVolumeMounts...)
	volumes = append(volumes, runArchiveVolumes...)

	return initContainerVolumeMounts, volumeMounts, volumes, nil
}

func (p *pod) configurePodSpec(
	volumes []v1.Volume,
	initContainers []v1.Container,
	containers []v1.Container,
) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.podName,
			Namespace: p.namespace,
			Labels:    map[string]string{determinedLabel: p.taskSpec.TaskID},
		},
		Spec: v1.PodSpec{
			Volumes:        volumes,
			HostNetwork:    p.taskSpec.TaskContainerDefaults.NetworkMode.IsHost(),
			InitContainers: initContainers,
			Containers:     containers,
			RestartPolicy:  v1.RestartPolicyNever,
		},
	}
}

func (p *pod) launchPod(ctx *actor.Context, podSpec *v1.Pod) error {
	var err error
	p.pod, err = p.podInterface.Create(podSpec)
	if err != nil {
		return errors.Wrap(err, "error creating pod")
	}
	ctx.Log().Infof("Created pod %s", p.pod.Name)

	p.container = container.Container{
		Parent:  p.taskHandler.Address(),
		ID:      container.ID(p.taskSpec.ContainerID),
		State:   container.Assigned,
		Devices: make([]device.Device, 0),
	}
	return nil
}

func (p *pod) startPodForTrial(ctx *actor.Context) error {
	exp := *p.taskSpec.StartContainer

	deviceType := device.CPU
	if p.gpus > 0 {
		deviceType = device.GPU
	}

	runArchives := tasks.TrialArchives(p.taskSpec)
	initContainerVolumeMounts, volumeMounts, volumes, err := p.configureVolumes(
		ctx, tasks.TrialDockerMounts(exp), runArchives)
	if err != nil {
		return err
	}

	p.ports = []int{
		tasks.LocalRendezvousPort, tasks.LocalRendezvousPort + tasks.LocalRendezvousPortOffset}
	rendezvousPorts := []string{
		fmt.Sprintf("%d", p.ports[0]), fmt.Sprintf("%d", p.ports[1]),
	}

	envVars, err := p.configureEnvVars(
		tasks.TrialEnvVars(p.taskSpec, rendezvousPorts),
		p.taskSpec.StartContainer.ExperimentConfig.Environment,
		deviceType,
	)
	if err != nil {
		return err
	}

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

	podSpec := p.configurePodSpec(volumes, initContainers, containers)
	return p.launchPod(ctx, podSpec)
}

func (p *pod) startPodForCommand(ctx *actor.Context) error {
	cmd := *p.taskSpec.StartCommand

	deviceType := device.CPU
	if p.gpus > 0 {
		deviceType = device.GPU
	}

	runArchives := tasks.CommandArchives(p.taskSpec)
	initContainerVolumeMounts, volumeMounts, volumes, err := p.configureVolumes(
		ctx, tasks.ToDockerMounts(cmd.Config.BindMounts), runArchives)
	if err != nil {
		return err
	}

	envVars, err := p.configureEnvVars(
		tasks.CommandEnvVars(p.taskSpec),
		p.taskSpec.StartCommand.Config.Environment,
		deviceType,
	)
	if err != nil {
		return err
	}

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

	podSpec := p.configurePodSpec(volumes, initContainers, containers)
	return p.launchPod(ctx, podSpec)
}

func (p *pod) startPodForGC(ctx *actor.Context) error {
	gcc := *p.taskSpec.GCCheckpoints

	deviceType := device.CPU
	if p.gpus > 0 {
		deviceType = device.GPU
	}

	runArchives := tasks.GCArchives(p.taskSpec)
	initContainerVolumeMounts, volumeMounts, volumes, err := p.configureVolumes(
		ctx, tasks.GCDockerMounts(gcc), runArchives)
	if err != nil {
		return err
	}

	envVars, err := p.configureEnvVars(
		tasks.GCEnvVars(),
		p.taskSpec.GCCheckpoints.ExperimentConfig.Environment,
		deviceType,
	)
	if err != nil {
		return err
	}

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

	podSpec := p.configurePodSpec(volumes, initContainers, containers)
	return p.launchPod(ctx, podSpec)
}

func configurePodName(t tasks.TaskSpec, rank int) string {
	switch {
	case t.StartCommand != nil:
		return fmt.Sprintf("cmd-%s", t.TaskID)
	case t.StartContainer != nil:
		return fmt.Sprintf(
			"exp-%d-trial-%d-%d",
			t.StartContainer.InitialWorkload.ExperimentID,
			t.StartContainer.InitialWorkload.TrialID, rank,
		)
	case t.GCCheckpoints != nil:
		return fmt.Sprintf("gc-%s", t.TaskID)
	default:
		return ""
	}
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
		Command: []string{path.Join(initContainerWorkDir, etc.K8InitContainerEntryScriptResource)},
		Args: []string{
			fmt.Sprintf("%d", numArchives), initContainerTarSrcPath, initContainerTarDstPath},
		Image:           image,
		ImagePullPolicy: imagePullPolicy,
		VolumeMounts:    volumeMounts,
		WorkingDir:      initContainerWorkDir,
	}
}

func getContainerState(conditions []v1.PodCondition) container.State {
	conditionsMap := make(map[v1.PodConditionType]bool)
	for _, condition := range conditions {
		conditionsMap[condition.Type] = condition.Status == v1.ConditionTrue
	}

	if conditionsMap[v1.PodReady] {
		return container.Running
	}

	if conditionsMap[v1.PodScheduled] {
		return container.Starting
	}

	return container.Assigned
}
