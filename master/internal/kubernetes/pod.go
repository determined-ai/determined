package kubernetes

import (
	"fmt"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	docker "github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/agent"
	"github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/tasks"

	k8sV1 "k8s.io/api/core/v1"
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
	resourceRequestQueue     *actor.Ref
	leaveKubernetesResources bool

	pod              *k8sV1.Pod
	podName          string
	configMap        *k8sV1.ConfigMap
	configMapName    string
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
	msg sproto.StartPod,
	cluster *actor.Ref,
	clusterID string,
	clientSet *k8sClient.Clientset,
	namespace string,
	masterIP string,
	masterPort int32,
	podInterface typedV1.PodInterface,
	configMapInterface typedV1.ConfigMapInterface,
	resourceRequestQueue *actor.Ref,
	leaveKubernetesResources bool,
) *pod {
	podContainer := container.Container{
		Parent: msg.TaskHandler.Address(),
		ID:     container.ID(msg.Spec.ContainerID),
		State:  container.Assigned,
	}
	uniqueName := configureUniqueName(msg.Spec, msg.Rank)

	return &pod{
		cluster:                  cluster,
		clusterID:                clusterID,
		taskHandler:              msg.TaskHandler,
		clientSet:                clientSet,
		namespace:                namespace,
		masterIP:                 masterIP,
		masterPort:               masterPort,
		taskSpec:                 msg.Spec,
		gpus:                     msg.Slots,
		rank:                     msg.Rank,
		podInterface:             podInterface,
		configMapInterface:       configMapInterface,
		resourceRequestQueue:     resourceRequestQueue,
		leaveKubernetesResources: leaveKubernetesResources,
		podName:                  uniqueName,
		configMapName:            uniqueName,
		container:                podContainer,
	}
}

func (p *pod) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		ctx.AddLabel("pod", p.podName)
		if err := p.createPodSpecAndSubmit(ctx); err != nil {
			return err
		}

	case resourceCreationFailed:
		p.receiveResourceCreationFailed(ctx, msg)

	case podStatusUpdate:
		if err := p.receivePodStatusUpdate(ctx, msg); err != nil {
			return err
		}

	case podEventUpdate:
		p.receivePodEventUpdate(ctx, msg)

	case sproto.ContainerLog:
		p.receiveContainerLogs(ctx, msg)

	case sproto.KillContainer:
		ctx.Log().Info("received request to stop pod")
		p.deleteKubernetesResources(ctx)

	case resourceCreationCancelled:
		p.receiveResourceCreationCancelled(ctx)

	case resourceDeletionFailed:
		p.receiveResourceDeletionFailed(ctx, msg)

	case getPodNodeInfo:
		p.receiveGetPodNodeInfo(ctx)

	case actor.PostStop:
		defer p.finalizeTaskState(ctx)

		if !p.leaveKubernetesResources {
			p.deleteKubernetesResources(ctx)
		}

	case actor.ChildStopped:
		if !p.resourcesDeleted {
			ctx.Log().Errorf("pod logger exited unexpectedly")
		}

	default:
		ctx.Log().Errorf("unexpected message %T", msg)
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

func (p *pod) createPodSpecAndSubmit(ctx *actor.Context) error {
	var err error
	switch {
	case p.taskSpec.StartCommand != nil:
		err = p.createPodSpecForCommand(ctx)
	case p.taskSpec.StartContainer != nil:
		err = p.createPodSpecForTrial(ctx)
	case p.taskSpec.GCCheckpoints != nil:
		err = p.createPodSpecForGC(ctx)
	default:
		return errors.Errorf("unexpected task spec received")
	}

	if err != nil {
		return err
	}

	ctx.Tell(p.resourceRequestQueue, createKubernetesResources{
		handler:       ctx.Self(),
		podSpec:       p.pod,
		configMapSpec: p.configMap,
	})
	return nil
}

func (p *pod) receiveResourceCreationFailed(ctx *actor.Context, msg resourceCreationFailed) {
	ctx.Log().WithError(msg.err).Error("pod actor notified that resource creation failed")
	errMsg := msg.err.Error()
	ctx.Tell(p.taskHandler, sproto.ContainerLog{
		Container:   p.container,
		Timestamp:   time.Now(),
		PullMessage: nil,
		RunMessage:  nil,
		AuxMessage:  &errMsg,
	})

	// If a subset of resources were created (e.g., configMap but podCreation failed) they will
	// be deleted during actor.PostStop.
	ctx.Self().Stop()
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

		// This is a hack to construct a Docker container info struct for pod since
		// the Docker container info struct is used in the protocol between the resource
		// provider and the task handler.
		transPort := func(ports []int) nat.PortSet {
			res := make(nat.PortSet)
			for _, port := range ports {
				res[nat.Port(strconv.Itoa(port))] = struct{}{}
			}
			return res
		}
		p.informTaskContainerStarted(ctx, agent.ContainerStarted{
			ProxyAddress: p.pod.Status.PodIP,
			ContainerInfo: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &docker.HostConfig{
						NetworkMode: "host",
					},
				},
				Config: &docker.Config{
					ExposedPorts: transPort(p.ports),
				},
			},
		})

	case k8sV1.PodFailed:
		if p.container.State == container.Terminated {
			return nil
		}

		exitCode, exitMessage, err := getExitCodeAndMessage(p.pod)
		if err != nil {
			return err
		}
		ctx.Log().Infof("pod failed with exit code: %d %s", exitCode, exitMessage)

		p.container = p.container.Transition(container.Terminated)
		exitCodeConverted := agent.ExitCode(exitCode)
		containerStopped := agent.ContainerStopped{
			Failure: &agent.ContainerFailure{
				FailureType: agent.ContainerFailed,
				ErrMsg:      exitMessage,
				ExitCode:    &exitCodeConverted,
			},
		}

		p.informTaskContainerStopped(ctx, containerStopped)
		ctx.Self().Stop()

	case k8sV1.PodSucceeded:
		if p.container.State == container.Terminated {
			return nil
		}
		p.container = p.container.Transition(container.Terminated)

		ctx.Log().Infof("pod exited successfully")
		containerStopped := agent.ContainerStopped{}

		p.informTaskContainerStopped(ctx, containerStopped)
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
	p.informTaskContainerStopped(ctx, containerStopped)
	ctx.Self().Stop()
}

func (p *pod) deleteKubernetesResources(ctx *actor.Context) {
	if p.resourcesDeleted {
		return
	}

	ctx.Log().Infof("requesting to delete kubernetes resources")
	ctx.Tell(p.resourceRequestQueue, deleteKubernetesResources{
		handler:       ctx.Self(),
		podName:       p.podName,
		configMapName: p.configMapName,
	})

	p.resourcesDeleted = true
}

func (p *pod) receiveResourceCreationCancelled(ctx *actor.Context) {
	ctx.Log().Infof("pod actor notified that resource creation was canceled")
	p.resourcesDeleted = true
	ctx.Self().Stop()
}

func (p *pod) receiveResourceDeletionFailed(
	ctx *actor.Context,
	msg resourceDeletionFailed,
) {
	ctx.Log().WithError(msg.err).Error("pod actor notified that resource deletion failed")
	ctx.Self().Stop()
}

func (p *pod) receiveGetPodNodeInfo(ctx *actor.Context) {
	ctx.Respond(podNodeInfo{
		nodeName:  p.pod.Spec.NodeName,
		numGPUs:   p.gpus,
		container: &p.container,
	})
}

func (p *pod) finalizeTaskState(ctx *actor.Context) {
	// If an error occurred during the lifecycle of the pods, we need to update the scheduler
	// and the task handler with new state.
	if p.container.State != container.Terminated {
		ctx.Log().Warnf("updating container state after pod actor exited unexpectedly")
		p.container = p.container.Transition(container.Terminated)

		containerStopped := agent.ContainerError(
			agent.TaskError, errors.New("agent failed while container was running"))

		p.informTaskContainerStopped(ctx, containerStopped)
	}
}

func (p *pod) informTaskContainerStarted(
	ctx *actor.Context,
	containerStarted agent.ContainerStarted,
) {
	ctx.Tell(p.taskHandler, sproto.ContainerStateChanged{
		Container:        p.container,
		ContainerStarted: &containerStarted,
	})
}

func (p *pod) informTaskContainerStopped(
	ctx *actor.Context,
	containerStopped agent.ContainerStopped,
) {
	ctx.Tell(p.taskHandler, sproto.ContainerStateChanged{
		Container:        p.container,
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

	if msg.event.Message[0:23] == gpuTextReplacement {
		msg.event.Message += strconv.Itoa(p.gpus) + " GPUs required."
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
