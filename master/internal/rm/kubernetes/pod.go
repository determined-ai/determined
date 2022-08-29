package kubernetes

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/tasks"

	k8sV1 "k8s.io/api/core/v1"
	k8sClient "k8s.io/client-go/kubernetes"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	initContainerTarSrcPath   = "/run/determined/temp/tar/src"
	initContainerTarDstPath   = "/run/determined/temp/tar/dst"
	initContainerWorkDir      = "/run/determined/temp/"
	determinedLabel           = "determined"
	determinedPreemptionLabel = "determined-preemption"
	determinedSystemLabel     = "determined-system"
)

// pod manages the lifecycle of a Kubernetes pod that executes a
// Determined task. The lifecycle of the pod is managed based on
// the status of the specified set of containers.
type pod struct {
	cluster                  *actor.Ref
	clusterID                string
	taskActor                *actor.Ref
	clientSet                *k8sClient.Clientset
	namespace                string
	masterIP                 string
	masterPort               int32
	taskSpec                 tasks.TaskSpec
	masterTLSConfig          model.TLSClientConfig
	loggingTLSConfig         model.TLSClientConfig
	loggingConfig            model.LoggingConfig
	slots                    int
	podInterface             typedV1.PodInterface
	configMapInterface       typedV1.ConfigMapInterface
	resourceRequestQueue     *actor.Ref
	leaveKubernetesResources bool
	scheduler                string
	slotType                 device.Type
	slotResourceRequests     PodSlotResourceRequests
	fluentConfig             FluentConfig

	pod           *k8sV1.Pod
	podName       string
	configMap     *k8sV1.ConfigMap
	configMapName string
	// TODO: Drop this manufactured container obj all together.
	container        cproto.Container
	ports            []int
	resourcesDeleted bool
	testLogStreamer  bool
	containerNames   map[string]bool

	logCtx logger.Context
}

// PodSlotResourceRequests contains the per-slot container requests.
type PodSlotResourceRequests struct {
	CPU float32 `json:"cpu"`
}

// FluentConfig stores k8s-configurable Fluent Bit-related options.
type FluentConfig struct {
	Image string `json:"image"`
}

// DefaultFluentConfig stores defaults for k8s-configurable Fluent Bit-related options.
var DefaultFluentConfig = FluentConfig{
	Image: aproto.FluentImage,
}

type getPodNodeInfo struct{}

type podNodeInfo struct {
	nodeName  string
	numSlots  int
	slotType  device.Type
	container *cproto.Container
}

func newPod(
	msg StartTaskPod,
	cluster *actor.Ref,
	clusterID string,
	clientSet *k8sClient.Clientset,
	namespace string,
	masterIP string,
	masterPort int32,
	masterTLSConfig model.TLSClientConfig,
	loggingTLSConfig model.TLSClientConfig,
	loggingConfig model.LoggingConfig,
	podInterface typedV1.PodInterface,
	configMapInterface typedV1.ConfigMapInterface,
	resourceRequestQueue *actor.Ref,
	leaveKubernetesResources bool,
	slotType device.Type,
	slotResourceRequests PodSlotResourceRequests,
	scheduler string,
	fluentConfig FluentConfig,
) *pod {
	podContainer := cproto.Container{
		Parent: msg.TaskActor.Address(),
		ID:     cproto.ID(msg.Spec.ContainerID),
		State:  cproto.Assigned,
	}
	uniqueName := configureUniqueName(msg.Spec, msg.Rank)

	// The lifecycle of the containers specified in this map will be monitored.
	// As soon as one or more of them exits outs, the pod will be terminated.
	containerNames := map[string]bool{model.DeterminedK8ContainerName: true}

	return &pod{
		cluster:                  cluster,
		clusterID:                clusterID,
		taskActor:                msg.TaskActor,
		clientSet:                clientSet,
		namespace:                namespace,
		masterIP:                 masterIP,
		masterPort:               masterPort,
		taskSpec:                 msg.Spec,
		masterTLSConfig:          masterTLSConfig,
		loggingTLSConfig:         loggingTLSConfig,
		loggingConfig:            loggingConfig,
		slots:                    msg.Slots,
		podInterface:             podInterface,
		configMapInterface:       configMapInterface,
		resourceRequestQueue:     resourceRequestQueue,
		leaveKubernetesResources: leaveKubernetesResources,
		podName:                  uniqueName,
		configMapName:            uniqueName,
		container:                podContainer,
		containerNames:           containerNames,
		scheduler:                scheduler,
		slotType:                 slotType,
		slotResourceRequests:     slotResourceRequests,
		fluentConfig:             fluentConfig,
		logCtx: logger.MergeContexts(msg.LogContext, logger.Context{
			"pod": uniqueName,
		}),
	}
}

func (p *pod) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		ctx.AddLabels(p.logCtx)
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

	case PreemptTaskPod:
		ctx.Log().Info("received preemption command")
		p.taskActor.System().Tell(p.taskActor, sproto.ReleaseResources{})

	case ChangePriority:
		ctx.Log().Info("interrupting pod to change priorities")
		p.taskActor.System().Tell(p.taskActor, sproto.ReleaseResources{})

	case ChangePosition:
		ctx.Log().Info("interrupting pod to change positions")
		p.taskActor.System().Tell(p.taskActor, sproto.ReleaseResources{})

	case sproto.ContainerLog:
		p.receiveContainerLog(ctx, msg)

	case KillTaskPod:
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
	if err := p.createPodSpec(ctx, p.scheduler); err != nil {
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
	p.insertLog(ctx, time.Now().UTC(), msg.err.Error())

	// If a subset of resources were created (e.g., configMap but podCreation failed) they will
	// be deleted during actor.PostStop.
	ctx.Self().Stop()
}

func (p *pod) receivePodStatusUpdate(ctx *actor.Context, msg podStatusUpdate) error {
	p.pod = msg.updatedPod

	containerState, err := getPodState(ctx, p.pod, p.containerNames)
	if err != nil {
		return err
	}

	if containerState == p.container.State {
		return nil
	}

	switch containerState {
	case cproto.Assigned:
		// Don't need to do anything.

	case cproto.Starting:
		// Kubernetes does not have an explicit state for pulling container images.
		// We insert it here because our  current implementation of the trial actor requires it.
		ctx.Log().Infof(
			"transitioning pod state from %s to %s", p.container.State, cproto.Pulling)
		p.container = p.container.Transition(cproto.Pulling)
		p.informTaskResourcesState(ctx)

		ctx.Log().Infof("transitioning pod state from %s to %s", p.container.State, containerState)
		p.container = p.container.Transition(cproto.Starting)
		p.informTaskResourcesState(ctx)

	case cproto.Running:
		ctx.Log().Infof("transitioning pod state from %s to %s", p.container.State, containerState)
		p.container = p.container.Transition(cproto.Running)
		// testLogStreamer is a testing flag only set in the pod_tests.
		// This allows us to bypass the need for a log streamer or REST server.
		if !p.testLogStreamer {
			logStreamer, err := newPodLogStreamer(p.podInterface, p.podName, ctx.Self())
			if err != nil {
				return err
			}
			if _, ok := ctx.ActorOf(fmt.Sprintf("%s-logs", p.podName), logStreamer); !ok {
				return errors.Errorf("log streamer already exists")
			}
		}

		addresses := []cproto.Address{}
		for _, port := range p.ports {
			addresses = append(addresses, cproto.Address{
				ContainerIP:   p.pod.Status.PodIP,
				ContainerPort: port,
				HostIP:        p.pod.Status.PodIP,
				HostPort:      port,
			})
		}
		var taskContainerID string
		for _, containerStatus := range p.pod.Status.ContainerStatuses {
			if containerStatus.Name == model.DeterminedK8ContainerName {
				taskContainerID = containerStatus.ContainerID
				break
			}
		}

		p.informTaskResourcesStarted(ctx, sproto.ResourcesStarted{
			Addresses:         addresses,
			NativeResourcesID: taskContainerID,
		})

	case cproto.Terminated:
		exitCode, exitMessage, err := getExitCodeAndMessage(p.pod, p.containerNames)
		if err != nil {
			// When a pod is deleted, it is possible that it will exit before the
			// determined containers generates an exit code. To check if this is
			// the case we check if a deletion timestamp has been set.
			if p.pod.ObjectMeta.DeletionTimestamp != nil {
				ctx.Log().Info("unable to get exit code for pod setting exit code to 137")
				exitCode = 137
				exitMessage = ""
			} else {
				return err
			}
		}

		ctx.Log().Infof("transitioning pod state from %s to %s", p.container.State, containerState)
		p.container = p.container.Transition(cproto.Terminated)

		var resourcesStopped sproto.ResourcesStopped
		switch exitCode {
		case aproto.SuccessExitCode:
			ctx.Log().Infof("pod exited successfully")
		default:
			ctx.Log().Infof("pod failed with exit code: %d %s", exitCode, exitMessage)
			resourcesStopped.Failure = sproto.NewResourcesFailure(
				sproto.ResourcesFailed,
				exitMessage,
				ptrs.Ptr(sproto.ExitCode(exitCode)))
		}
		p.informTaskResourcesStopped(ctx, resourcesStopped)
		ctx.Self().Stop()

	default:
		panic(fmt.Sprintf("unexpected container state %s", containerState))
	}

	return nil
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
		numSlots:  p.slots,
		slotType:  p.slotType,
		container: &p.container,
	})
}

func (p *pod) finalizeTaskState(ctx *actor.Context) {
	// If an error occurred during the lifecycle of the pods, we need to update the scheduler
	// and the task handler with new state.
	if p.container.State != cproto.Terminated {
		ctx.Log().Warnf("updating container state after pod actor exited unexpectedly")
		p.container = p.container.Transition(cproto.Terminated)

		p.informTaskResourcesStopped(ctx, sproto.ResourcesError(
			sproto.TaskError, errors.New("pod actor exited while pod was running")))
	}
}

func (p *pod) informTaskResourcesState(ctx *actor.Context) {
	ctx.Tell(p.taskActor, sproto.ResourcesStateChanged{
		ResourcesID:    sproto.FromContainerID(p.container.ID),
		ResourcesState: sproto.FromContainerState(p.container.State),
		Container:      p.container.DeepCopy(),
	})
}

func (p *pod) informTaskResourcesStarted(
	ctx *actor.Context,
	rs sproto.ResourcesStarted,
) {
	ctx.Tell(p.taskActor, sproto.ResourcesStateChanged{
		ResourcesID:      sproto.FromContainerID(p.container.ID),
		ResourcesState:   sproto.FromContainerState(p.container.State),
		ResourcesStarted: &rs,
		Container:        p.container.DeepCopy(),
	})
}

func (p *pod) informTaskResourcesStopped(
	ctx *actor.Context,
	rs sproto.ResourcesStopped,
) {
	ctx.Tell(p.taskActor, sproto.ResourcesStateChanged{
		ResourcesID:      sproto.FromContainerID(p.container.ID),
		ResourcesState:   sproto.FromContainerState(p.container.State),
		ResourcesStopped: &rs,
		Container:        p.container.DeepCopy(),
	})
}

func (p *pod) receiveContainerLog(ctx *actor.Context, msg sproto.ContainerLog) {
	msg.Container = p.container
	ctx.Tell(p.taskActor, msg)
}

func (p *pod) insertLog(ctx *actor.Context, timestamp time.Time, msg string) {
	p.receiveContainerLog(ctx, sproto.ContainerLog{
		Timestamp:  timestamp,
		AuxMessage: &msg,
	})
}

// Converts k8s message to be more understandable.
func (p *pod) preparePodUpdateMessage(msgText string) string {
	// Handle simple message replacements.
	replacements := map[string]string{
		"pod triggered scale-up":     "Job requires additional resources, scaling up cluster.",
		"Successfully assigned":      "Pod resources allocated.",
		"skip schedule deleting pod": "Deleting unscheduled pod.",
	}

	simpleReplacement := false

	for k, v := range replacements {
		matched, err := regexp.MatchString(k, msgText)
		if err != nil {
			break
		} else if matched {
			msgText = v
			simpleReplacement = true
		}
	}

	// Otherwise, try special treatment for slots availability message.
	if !simpleReplacement {
		matched, err := regexp.MatchString("nodes are available", msgText)
		if err == nil && matched {
			available := string(msgText[0])
			required := strconv.Itoa(p.slots)
			var resourceName string
			switch p.slotType {
			case device.CPU:
				resourceName = "CPU slots"
			default:
				resourceName = "GPUs"
			}

			msgText = fmt.Sprintf("Waiting for resources. %s %s are available, %s %s required",
				available, resourceName, required, resourceName)
		}
	}

	return msgText
}

func (p *pod) receivePodEventUpdate(ctx *actor.Context, msg podEventUpdate) {
	// We only forward messages while pods are starting up.
	switch p.container.State {
	case cproto.Running, cproto.Terminated:
		return
	}

	msgText := p.preparePodUpdateMessage(msg.event.Message)
	msg.event.Message = msgText

	message := fmt.Sprintf("Pod %s: %s", msg.event.InvolvedObject.Name, msgText)
	p.insertLog(ctx, msg.event.CreationTimestamp.Time, message)
}

func getPodState(
	ctx *actor.Context,
	pod *k8sV1.Pod,
	containerNames map[string]bool,
) (cproto.State, error) {
	switch pod.Status.Phase {
	case k8sV1.PodPending:
		// When pods are deleted, Kubernetes sometimes transitions pod statuses to pending
		// prior to deleting them. In these cases we have observed that we do not always
		// receive a PodFailed or a PodSucceeded message. We check if pods have a set pod
		// deletion timestamp to see if this is the case.
		if pod.ObjectMeta.DeletionTimestamp != nil {
			ctx.Log().Warn("marking pod as terminated due to deletion timestamp")
			return cproto.Terminated, nil
		}

		for _, condition := range pod.Status.Conditions {
			if condition.Type == k8sV1.PodScheduled && condition.Status == k8sV1.ConditionTrue {
				return cproto.Starting, nil
			}
		}
		return cproto.Assigned, nil

	case k8sV1.PodRunning:
		// Pods are in a running state as long as at least one container has not terminated.
		// We check the status of the Determined containers directly to determine if they
		// are still running.
		containerStatuses, err := getDeterminedContainersStatus(
			pod.Status.ContainerStatuses, containerNames)
		if err != nil {
			return "", err
		}

		for _, containerStatus := range containerStatuses {
			if containerStatus.State.Terminated != nil {
				return cproto.Terminated, nil
			}
		}

		for _, containerStatus := range containerStatuses {
			// Check that all Determined containers are running.
			if containerStatus.State.Running == nil {
				return cproto.Starting, nil
			}
		}

		return cproto.Running, nil

	case k8sV1.PodFailed, k8sV1.PodSucceeded:
		return cproto.Terminated, nil

	default:
		return "", errors.Errorf(
			"unexpected pod status %s for pod %s", pod.Status.Phase, pod.Name)
	}
}

func getExitCodeAndMessage(pod *k8sV1.Pod, containerNames map[string]bool) (int, string, error) {
	if len(pod.Status.InitContainerStatuses) == 0 {
		return 0, "", errors.Errorf(
			"unexpected number of init containers when processing exit code for pod %s", pod.Name)
	}

	for _, initContainerStatus := range pod.Status.InitContainerStatuses {
		if initContainerStatus.State.Terminated == nil {
			continue
		}
		exitCode := initContainerStatus.State.Terminated.ExitCode
		if exitCode != aproto.SuccessExitCode {
			errMessage := fmt.Sprintf(
				"container %s: %s", initContainerStatus.Name,
				initContainerStatus.State.Terminated.Message,
			)
			return int(exitCode), errMessage, nil
		}
	}

	if len(pod.Status.ContainerStatuses) < len(containerNames) {
		return 0, "", errors.Errorf(
			"unexpected number of containers when processing exit code for pod %s", pod.Name)
	}

	containerStatuses, err := getDeterminedContainersStatus(
		pod.Status.ContainerStatuses, containerNames)
	if err != nil {
		return 0, "", err
	}

	for _, containerStatus := range containerStatuses {
		terminationStatus := containerStatus.State.Terminated
		if terminationStatus != nil {
			return int(terminationStatus.ExitCode), terminationStatus.Message, nil
		}
	}

	return 0, "", errors.Errorf("unable to get exit code from pod %s", pod.Name)
}

func getDeterminedContainersStatus(
	statuses []k8sV1.ContainerStatus,
	containerNames map[string]bool,
) ([]*k8sV1.ContainerStatus, error) {
	containerStatuses := make([]*k8sV1.ContainerStatus, 0, len(statuses))
	for idx, containerStatus := range statuses {
		if _, match := containerNames[containerStatus.Name]; !match {
			continue
		}
		containerStatuses = append(containerStatuses, &statuses[idx])
	}

	if len(containerStatuses) != len(containerNames) {
		containerNamesFound := make([]string, 0, len(containerStatuses))
		for _, containerStatus := range containerStatuses {
			containerNamesFound = append(containerNamesFound, containerStatus.Name)
		}
		return nil, errors.Errorf("found container statuses only for: %v", containerNamesFound)
	}

	return containerStatuses, nil
}
