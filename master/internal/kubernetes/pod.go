package kubernetes

import (
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/docker/docker/api/types/mount"
	petname "github.com/dustinkirkland/golang-petname"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/agent"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"

	k8sV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sClient "k8s.io/client-go/kubernetes"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	initContainerTarSrcPath = "/run/determined/temp/tar/src"
	initContainerTarDstPath = "/run/determined/temp/tar/dst"
	initContainerWorkDir    = "/run/determined/temp/"
	determinedLabel         = "determined"
)

type pod struct {
	cluster                  *actor.Ref
	clusterID                string
	taskHandler              *actor.Ref
	clientSet                *k8sClient.Clientset
	namespace                string
	masterIP                 string
	masterPort               int32
	taskSpec                 tasks.TaskSpec
	gpus                     int
	rank                     int
	podInterface             typedV1.PodInterface
	configMapInterface       typedV1.ConfigMapInterface
	leaveKubernetesResources bool

	pod              *k8sV1.Pod
	configMaps       []*k8sV1.ConfigMap
	podName          string
	container        container.Container
	ports            []int
	resourcesDeleted bool
}

type getPodNodeInfo struct{}

type podNodeInfo struct {
	nodeName  string
	numGPUs   int
	container *container.Container
}

func newPod(
	cluster *actor.Ref,
	clusterID string,
	taskHandler *actor.Ref,
	clientSet *k8sClient.Clientset,
	namespace string,
	masterIP string,
	masterPort int32,
	taskSpec tasks.TaskSpec,
	gpus int,
	rank int,
	podInterface typedV1.PodInterface,
	configMapInterface typedV1.ConfigMapInterface,
	leaveKubernetesResources bool,
) *pod {
	podContainer := container.Container{
		Parent: taskHandler.Address(),
		ID:     container.ID(taskSpec.ContainerID),
		State:  container.Assigned,
	}

	return &pod{
		cluster:                  cluster,
		clusterID:                clusterID,
		taskHandler:              taskHandler,
		clientSet:                clientSet,
		namespace:                namespace,
		masterIP:                 masterIP,
		masterPort:               masterPort,
		taskSpec:                 taskSpec,
		gpus:                     gpus,
		rank:                     rank,
		podInterface:             podInterface,
		configMapInterface:       configMapInterface,
		leaveKubernetesResources: leaveKubernetesResources,
		podName:                  configurePodName(taskSpec, rank),
		container:                podContainer,
	}
}

func (p *pod) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		ctx.AddLabel("pod", p.podName)
		if err := p.startPod(ctx); err != nil {
			return err
		}

	case podStatusUpdate:
		if err := p.receivePodStatusUpdate(ctx, msg); err != nil {
			return err
		}

	case podEventUpdate:
		p.receivePodEventUpdate(ctx, msg)

	case sproto.ContainerLog:
		p.receiveContainerLogs(ctx, msg)

	case sproto.StopPod:
		ctx.Log().Info("received request to stop pod")
		if err := p.deleteKubernetesResources(ctx); err != nil {
			return err
		}

	case getPodNodeInfo:
		ctx.Respond(podNodeInfo{
			nodeName:  p.pod.Spec.NodeName,
			numGPUs:   p.gpus,
			container: &p.container,
		})

	case actor.PostStop:
		defer p.finalizeTaskState(ctx)

		if !p.leaveKubernetesResources {
			if err := p.deleteKubernetesResources(ctx); err != nil {
				return err
			}
		}

	case actor.ChildStopped:

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
	case k8sV1.PodPending:
		// When pods are deleted, Kubernetes sometimes transitions pod statuses to pending prior
		// to deleting them. In these cases we have observed that we do not always receive a PodFailed
		// or a PodSucceeded message. We check if pods have a set pod deletion timestamp to see if this
		// is the case.
		if p.pod.ObjectMeta.DeletionTimestamp != nil {
			p.processMissingPodDeletion(ctx)
			return nil
		}

		containerState := getContainerState(msg.updatedPod.Status.Conditions)
		if containerState == container.Running {
			ctx.Log().Errorf("unexpected containers status while pod is pending")
		}

		if containerState == p.container.State {
			return nil
		}

		if containerState == container.Starting {
			// Kubernetes does not have an explicit state for pulling container
			// images. We insert it here because our  current implementation of
			// the trial actor requires it.
			ctx.Log().Infof("transitioning pod state from %s to %s",
				p.container.State, container.Pulling)
			p.container = p.container.Transition(container.Pulling)

			rsc := sproto.ContainerStateChanged{Container: p.container}
			ctx.Tell(p.taskHandler, rsc)
		}

		ctx.Log().Infof("transitioning pod state from %s to %s", p.container.State, containerState)
		p.container = p.container.Transition(containerState)

		rsc := sproto.ContainerStateChanged{Container: p.container}
		ctx.Tell(p.taskHandler, rsc)

	case k8sV1.PodRunning:
		if p.container.State == container.Running {
			return nil
		}
		p.container = p.container.Transition(container.Running)

		logStreamer, err := newPodLogStreamer(p.podInterface, p.podName, ctx.Self())
		if err != nil {
			return err
		}
		if _, ok := ctx.ActorOf(fmt.Sprintf("%s-logs", p.podName), logStreamer); !ok {
			return errors.Errorf("log streamer already exists")
		}

		ctx.Tell(p.taskHandler, sproto.ContainerStateChanged{Container: p.container})
		ctx.Tell(p.cluster, sproto.PodStarted{
			ContainerID: p.container.ID,
			IP:          p.pod.Status.PodIP,
			Ports:       p.ports,
		})

	case k8sV1.PodFailed:
		if p.container.State == container.Terminated {
			return nil
		}
		p.container = p.container.Transition(container.Terminated)

		exitCode, exitMessage, err := getExitCodeAndMessage(p.pod)
		if err != nil {
			return err
		}
		ctx.Log().Infof("pod failed: %d %s", exitCode, exitMessage)

		exitCodeConverted := agent.ExitCode(exitCode)
		containerStopped := agent.ContainerStopped{
			Failure: &agent.ContainerFailure{
				FailureType: agent.ContainerFailed,
				ErrMsg:      exitMessage,
				ExitCode:    &exitCodeConverted,
			},
		}

		p.informThatContainerStopped(ctx, containerStopped)
		ctx.Self().Stop()

	case k8sV1.PodSucceeded:
		if p.container.State == container.Terminated {
			return nil
		}
		p.container = p.container.Transition(container.Terminated)

		ctx.Log().Infof("pod exited successfully")
		containerStopped := agent.ContainerStopped{}

		p.informThatContainerStopped(ctx, containerStopped)
		ctx.Self().Stop()

	default:
		return errors.Errorf(
			"unexpected pod status %s for pod %s", msg.updatedPod.Status.Phase, p.podName)
	}

	return nil
}

func (p *pod) processMissingPodDeletion(ctx *actor.Context) {
	ctx.Log().Warn("processing missing pod deletion")
	if p.container.State == container.Terminated {
		ctx.Log().Info(
			"skipping processing missing pod deletion as container is in a terminated state")
		return
	}

	if !p.resourcesDeleted {
		ctx.Log().Errorf("processing missing pod deletion for a pod that was never deleted")
	}

	p.container = p.container.Transition(container.Terminated)
	// Missed pod deletions occur only when a pod is deleted so we assume
	// that the container was killed.
	exitCodeConverted := agent.ExitCode(137)
	containerStopped := agent.ContainerStopped{
		Failure: &agent.ContainerFailure{
			FailureType: agent.ContainerFailed,
			ExitCode:    &exitCodeConverted,
		},
	}
	p.informThatContainerStopped(ctx, containerStopped)
	ctx.Self().Stop()
}

func (p *pod) deleteKubernetesResources(ctx *actor.Context) error {
	if p.resourcesDeleted {
		return nil
	}

	ctx.Log().Infof("deleting pod")
	var gracePeriod int64 = 15
	err := p.podInterface.Delete(p.podName, &metaV1.DeleteOptions{GracePeriodSeconds: &gracePeriod})
	if err != nil {
		ctx.Log().WithError(err).Errorf("pod deletion failed %s", p.podName)
	}

	for _, cf := range p.configMaps {
		errDeletingConfigMap := p.configMapInterface.Delete(cf.Name, &metaV1.DeleteOptions{
			GracePeriodSeconds: &gracePeriod})

		if errDeletingConfigMap != nil {
			ctx.Log().WithError(errDeletingConfigMap).Errorf("config map deletion failed %s", cf.Name)
			err = errDeletingConfigMap
		}
	}

	p.resourcesDeleted = true
	return err
}

func (p *pod) finalizeTaskState(ctx *actor.Context) {
	// If an error occurred during the lifecycle of the pods, we need to update the scheduler
	// and the task handler with new state.
	if p.container.State != container.Terminated {
		ctx.Log().Warnf("updating container state after pod actor exited unexpectedly")
		p.container = p.container.Transition(container.Terminated)

		containerStopped := agent.ContainerError(
			agent.TaskError, errors.New("agent failed while container was running"))

		p.informThatContainerStopped(ctx, containerStopped)
	}
}

func (p *pod) informThatContainerStopped(
	ctx *actor.Context,
	containerStopped agent.ContainerStopped,
) {
	ctx.Tell(p.taskHandler, sproto.ContainerStateChanged{
		Container:        p.container,
		ContainerStopped: &containerStopped,
	})

	ctx.Tell(p.cluster, sproto.PodTerminated{
		ContainerID:      p.container.ID,
		ContainerStopped: &containerStopped,
	})
}

func (p *pod) receiveContainerLogs(ctx *actor.Context, msg sproto.ContainerLog) {
	msg.Container = p.container
	ctx.Tell(p.taskHandler, msg)
}

func (p *pod) receivePodEventUpdate(ctx *actor.Context, msg podEventUpdate) {
	// We only forward messages while pods are starting up.
	switch p.container.State {
	case container.Running, container.Terminated:
		return
	}

	message := fmt.Sprintf("Pod %s: %s", msg.event.InvolvedObject.Name, msg.event.Message)
	ctx.Tell(p.taskHandler, sproto.ContainerLog{
		Container:   p.container,
		Timestamp:   msg.event.CreationTimestamp.Time,
		PullMessage: nil,
		RunMessage:  nil,
		AuxMessage:  &message,
	})
}

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

func (p *pod) configureRunArchives(
	ctx *actor.Context,
	runArchives []container.RunArchive,
) ([]k8sV1.VolumeMount, []k8sV1.VolumeMount, []k8sV1.Volume, error) {
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
		createConfigMapSpec(p.podName, tarredArchives, p.namespace, p.taskSpec.TaskID),
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
		createConfigMapSpec(
			p.podName, initContainerEntrypointArchive, p.namespace, p.taskSpec.TaskID),
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
) ([]k8sV1.VolumeMount, []k8sV1.VolumeMount, []k8sV1.Volume, error) {
	volumeMounts := make([]k8sV1.VolumeMount, 0)
	volumes := make([]k8sV1.Volume, 0)

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
	ctx *actor.Context,
	volumes []k8sV1.Volume,
	initContainers []k8sV1.Container,
	containers []k8sV1.Container,
	podSpec *k8sV1.Pod,
) *k8sV1.Pod {
	if podSpec == nil {
		podSpec = &k8sV1.Pod{}
	} else {
		ctx.Log().Info("using user provided pod_spec as a template")
		podSpec = podSpec.DeepCopy()
	}
	ctx.Log().Debugf("using base pods spec: %v", podSpec.Spec)

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

	ctx.Log().Debugf("launching pod spec %v", podSpec.Spec)
	return podSpec
}

func (p *pod) launchPod(ctx *actor.Context, podSpec *k8sV1.Pod) error {
	var err error
	p.pod, err = p.podInterface.Create(podSpec)
	if err != nil {
		errMsg := err.Error()
		ctx.Tell(p.taskHandler, sproto.ContainerLog{
			Container:   p.container,
			Timestamp:   time.Now(),
			PullMessage: nil,
			RunMessage:  nil,
			AuxMessage:  &errMsg,
		})
		return errors.Wrap(err, "error creating pod")
	}
	ctx.Log().Infof("Created pod %s", p.pod.Name)
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

	podSpec := p.configurePodSpec(
		ctx, volumes, initContainers, containers, exp.ExperimentConfig.Kuberenetes.PodSpec)
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

	podSpec := p.configurePodSpec(
		ctx, volumes, initContainers, containers, cmd.Config.Kubernetes.PodSpec)
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

	podSpec := p.configurePodSpec(
		ctx, volumes, initContainers, containers, gcc.ExperimentConfig.Kuberenetes.PodSpec)
	return p.launchPod(ctx, podSpec)
}

func configurePodName(t tasks.TaskSpec, rank int) string {
	uniqueName := petname.Generate(2, "-")
	switch {
	case t.StartCommand != nil:
		return fmt.Sprintf("cmd-%s-%s", t.TaskID, uniqueName)
	case t.StartContainer != nil:
		return fmt.Sprintf(
			"exp-%d-trial-%d-%d-%s",
			t.StartContainer.InitialWorkload.ExperimentID,
			t.StartContainer.InitialWorkload.TrialID, rank,
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
		userID := int64(agentUserGroup.ID)
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

func getContainerState(conditions []k8sV1.PodCondition) container.State {
	conditionsMap := make(map[k8sV1.PodConditionType]bool)
	for _, condition := range conditions {
		conditionsMap[condition.Type] = condition.Status == k8sV1.ConditionTrue
	}

	switch {
	case conditionsMap[k8sV1.PodReady]:
		return container.Running
	case conditionsMap[k8sV1.PodScheduled]:
		return container.Starting
	}

	return container.Assigned
}

func getExitCodeAndMessage(pod *k8sV1.Pod) (int, string, error) {
	if len(pod.Status.InitContainerStatuses) != 1 {
		return 0, "", errors.Errorf(
			"unexpected number of init containers when processing failure for pod %s", pod.Name)
	}

	initContainerStatus := pod.Status.InitContainerStatuses[0].State.Terminated
	if initContainerStatus.ExitCode != agent.SuccessExitCode {
		return int(initContainerStatus.ExitCode), initContainerStatus.Message, nil
	}

	if len(pod.Status.ContainerStatuses) != 1 {
		return 0, "", errors.Errorf(
			"unexpected number of containers when processing failure for pod %s", pod.Name)
	}

	containerStatus := pod.Status.ContainerStatuses[0].State.Terminated
	return int(containerStatus.ExitCode), containerStatus.Message, nil
}
