package kubernetesrm

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/docker/docker/pkg/stdcopy"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/maps"
	batchV1 "k8s.io/api/batch/v1"
	k8sV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sClient "k8s.io/client-go/kubernetes"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	gatewayTyped "sigs.k8s.io/gateway-api/apis/v1"
	alphaGatewayTyped "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/rm/rmevents"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/set"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

type gatewayProxyResource struct {
	serviceSpec     *k8sV1.Service
	tcpRouteSpec    *alphaGatewayTyped.TCPRoute
	gatewayListener gatewayTyped.Listener
}

func (g gatewayProxyResource) PodPort() int {
	return int(g.serviceSpec.Spec.Ports[0].Port)
}

func (g gatewayProxyResource) GWPort() int {
	return int(g.gatewayListener.Port)
}

func (g gatewayProxyResource) SetGWPort(port int) {
	gwPort := gatewayTyped.PortNumber(port)
	g.gatewayListener.Port = gwPort
	g.tcpRouteSpec.Spec.CommonRouteSpec.ParentRefs[0].Port = &gwPort
}

var successfulExit = exitReason{}

// describes why a job failed. empty value indicates success.
type exitReason struct {
	code        int
	msg         string
	failureType sproto.FailureType
}

func (r *exitReason) String() string {
	if isSuccessfulExit(r) {
		return "success"
	}
	return fmt.Sprintf("%s code=%d type=%s", r.msg, r.code, r.failureType)
}

type podNodeInfo struct {
	nodeName  string
	numSlots  int
	slotType  device.Type
	container *cproto.Container
}

// job manages the lifecycle of a Kubernetes Job that executes a
// Determined task.
type job struct {
	// Configuration details. Set in initialization (the `newJob` constructor) and never modified after.
	clusterID       string
	masterHost      string
	masterPort      int32
	masterScheme    string
	masterTLSConfig model.TLSClientConfig
	jobName         string
	configMapName   string

	gatewayProxyResources []gatewayProxyResource
	internalTaskGWConfig  *config.InternalTaskGatewayConfig
	gatewayService        *gatewayService

	allocationID model.AllocationID
	// req.State is mutated, we should change this.
	req *sproto.AllocateRequest
	// Kubernetes-specific request information.
	namespace            string
	slotsPerPod          int
	numPods              int
	containerNames       set.Set[string]
	scheduler            string
	slotType             device.Type
	slotResourceRequests config.PodSlotResourceRequests
	restore              bool

	// System dependencies. Also set in initialization and never modified after.
	syslog               *logrus.Entry
	clientSet            k8sClient.Interface
	podInterface         typedV1.PodInterface
	configMapInterface   typedV1.ConfigMapInterface
	resourceRequestQueue *requestQueue

	// Internal state. Access should be protected.
	mu                    sync.Mutex
	podKillSent           map[string]bool
	podLogStreamerStarted map[string]bool
	podNodeNames          map[string]string
	podStates             map[string]cproto.State
	podExits              map[string]bool
	jobExitCause          *exitReason
	sentStartingEvent     bool
	sentRunningEvent      bool
	sentTerminationEvent  bool
	// TODO(DET-10013) : Remove container field from pod struct. And get away from having several IDs, just use job name.
	container        cproto.Container
	resourcesDeleted atomic.Bool
}

func newJob(
	name string,
	msg startJob,
	clusterID string,
	clientSet k8sClient.Interface,
	namespace string,
	masterHost string,
	masterPort int32,
	masterScheme string,
	masterTLSConfig model.TLSClientConfig,
	podInterface typedV1.PodInterface,
	configMapInterface typedV1.ConfigMapInterface,
	resourceRequestQueue *requestQueue,
	slotType device.Type,
	slotResourceRequests config.PodSlotResourceRequests,
	scheduler string,
	internalTaskGWConfig *config.InternalTaskGatewayConfig,
	gatewayService *gatewayService,
) *job {
	// The lifecycle of the containers specified in this map will be monitored.
	// As soon as one or more of them exits, the pod will be terminated.
	containerNames := set.FromSlice([]string{model.DeterminedK8ContainerName})

	p := &job{
		req:                   msg.req,
		clusterID:             clusterID,
		allocationID:          msg.allocationID,
		clientSet:             clientSet,
		namespace:             namespace,
		masterHost:            masterHost,
		masterPort:            masterPort,
		masterScheme:          masterScheme,
		masterTLSConfig:       masterTLSConfig,
		numPods:               msg.numPods,
		slotsPerPod:           msg.slots,
		podInterface:          podInterface,
		configMapInterface:    configMapInterface,
		resourceRequestQueue:  resourceRequestQueue,
		jobName:               name,
		configMapName:         name,
		podNodeNames:          make(map[string]string),
		podStates:             make(map[string]cproto.State),
		podKillSent:           make(map[string]bool),
		podExits:              make(map[string]bool),
		podLogStreamerStarted: make(map[string]bool),
		container: cproto.Container{
			ID:          cproto.ID(msg.spec.ContainerID),
			State:       cproto.Assigned,
			Description: msg.spec.Description,
		},
		containerNames:       containerNames,
		scheduler:            scheduler,
		slotType:             slotType,
		slotResourceRequests: slotResourceRequests,
		internalTaskGWConfig: internalTaskGWConfig,
		gatewayService:       gatewayService,
		syslog: logrus.WithField("component", "job").WithFields(
			logger.MergeContexts(msg.logContext, logger.Context{
				"job": name,
			}).Fields(),
		),
	}
	return p
}

func (j *job) finalize() {
	j.mu.Lock()
	defer j.mu.Unlock()

	// If an error occurred during the lifecycle of the pods, we need to update the scheduler
	// and the task handler with new state.
	if j.container.State != cproto.Terminated {
		j.kill()
		j.syslog.Warnf("killed job after our handler exited unexpectedly")
		j.container.State = cproto.Terminated
		j.jobExitCause = &exitReason{
			failureType: sproto.TaskError,
			msg:         "job crashed",
		}
		j.informTaskResourcesStopped()
	}
}

func (j *job) exitCause() *sproto.ResourcesFailedError {
	if isSuccessfulExit(j.jobExitCause) {
		return nil
	}

	failureType := j.jobExitCause.failureType
	if failureType == "" {
		failureType = sproto.ResourcesFailed
	}
	var exitCode *sproto.ExitCode
	if j.jobExitCause.code > 0 {
		exitCode = (*sproto.ExitCode)(&j.jobExitCause.code)
	}
	return &sproto.ResourcesFailedError{
		FailureType: failureType,
		ErrMsg:      j.jobExitCause.msg,
		ExitCode:    exitCode,
	}
}

func isSuccessfulExit(cause *exitReason) bool {
	return cause == nil || *cause == successfulExit
}

func (j *job) jobUpdatedCallback(updatedJob *batchV1.Job) (cproto.State, error) {
	j.mu.Lock()
	defer j.mu.Unlock()

	if j.container.State == cproto.Terminated {
		return j.container.State, nil
	}

	conds := updatedJob.Status.Conditions
	if len(conds) == 0 {
		return j.container.State, nil
	}

	for _, cond := range conds {
		if cond.Status != k8sV1.ConditionTrue {
			continue
		}

		switch cond.Type {
		case batchV1.JobComplete:
			if j.jobExitCause == nil {
				j.jobExitCause = &successfulExit
			}
			j.syslog.Infof(
				"job %s completed and transitioned from %s to %s",
				updatedJob.Name, j.container.State, cproto.Terminated,
			)
			j.container.State = cproto.Terminated
			j.informTaskResourcesStopped()
			return cproto.Terminated, nil

		case batchV1.JobFailed:
			if j.jobExitCause == nil {
				j.jobExitCause = &exitReason{msg: fmt.Sprintf(
					"job exited with a failure but we don't have pod-level detail: %s",
					cond.Message,
				)}
			}
			j.syslog.Infof("job %s failed and transitioned from %s to %s", updatedJob.Name, j.container.State, cproto.Terminated)
			j.container.State = cproto.Terminated
			j.informTaskResourcesStopped()
			return cproto.Terminated, nil
		}
	}

	return j.container.State, nil
}

func (j *job) jobDeletedCallback() {
	j.mu.Lock()
	defer j.mu.Unlock()

	if j.container.State == cproto.Terminated {
		return
	}

	if j.jobExitCause == nil {
		j.jobExitCause = &exitReason{msg: "job was deleted"}
	}
	j.syslog.Info("job deleted")
	j.container.State = cproto.Terminated
	j.informTaskResourcesStopped()
}

func (j *job) makeGatewayComms(spec *tasks.TaskSpec) *gatewayResourceComm {
	if j.internalTaskGWConfig == nil {
		return nil
	}

	updateResources := func(resources []gatewayProxyResource) {
		j.mu.Lock()
		defer j.mu.Unlock()
		j.gatewayProxyResources = resources
	}

	return &gatewayResourceComm{
		resourceDescriptor: j.configureProxyResources(spec),
		reportResources:    updateResources,
		allocationID:       j.req.AllocationID,
		requestedPorts:     len(j.req.ProxyPorts),
	}
}

func (j *job) getGatewayAddresses() []cproto.Address {
	if j.internalTaskGWConfig == nil {
		return nil
	}

	var addresses []cproto.Address
	for _, g := range j.gatewayProxyResources {
		addresses = append(addresses, cproto.Address{
			ContainerIP:   j.internalTaskGWConfig.GatewayIP,
			HostIP:        j.internalTaskGWConfig.GatewayIP,
			ContainerPort: g.PodPort(),
			HostPort:      g.GWPort(),
		})
	}

	return addresses
}

func (j *job) podUpdatedCallback(updatedPod k8sV1.Pod) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	podName := updatedPod.Name
	updatedPodState, err := j.getPodState(updatedPod)
	if err != nil {
		return err
	}
	j.podStates[podName] = updatedPodState

	j.podNodeNames[podName] = updatedPod.Spec.NodeName

	// Jobs with pods in ImagePullBackOff get stuck (https://github.com/kubernetes/kubernetes/issues/101584).
	for _, s := range append(updatedPod.Status.InitContainerStatuses, updatedPod.Status.ContainerStatuses...) {
		// Only check for ImagePullBackOff, ErrImagePull could be an intermittent issue and we want to be sure.
		// Waiting for backoff doesn't take very long.
		if waiting := s.State.Waiting; waiting != nil && waiting.Reason == "ImagePullBackOff" {
			j.jobExitCause = &exitReason{msg: "job was stuck due to unrecoverable image pull errors"}
			j.syslog.WithField("detail", waiting.Message).Infof(j.jobExitCause.msg)
			j.kill()
		}
	}

	allPodsFound := len(j.podStates) == j.numPods
	allPodsAtLeastStarting := all(cproto.Starting.Before, maps.Values(j.podStates)...)
	if allPodsFound && allPodsAtLeastStarting && !j.sentStartingEvent {
		// Kubernetes does not have an explicit state for pulling container images.
		// We insert it here because our  current implementation of the trial actor requires it.
		j.syslog.WithField("pod-name", podName).Info("pod is pulling images and starting")
		j.container.State = cproto.Pulling
		j.informTaskResourcesState()

		j.container.State = cproto.Starting
		j.informTaskResourcesState()
		j.sentStartingEvent = true
	}

	if updatedPodState == cproto.Running && !j.podLogStreamerStarted[podName] {
		err := startPodLogStreamer(j.podInterface, podName, func(log []byte) {
			j.receiveContainerLog(sproto.ContainerLog{
				Timestamp: time.Now().UTC(),
				RunMessage: &aproto.RunMessage{
					Value:   string(log),
					StdType: stdcopy.Stdout,
				},
			})
		})
		if err != nil {
			return fmt.Errorf("starting pod logs streamer for %s: %w", podName, err)
		}
		j.podLogStreamerStarted[podName] = true
	}

	allPodsAtLeastRunning := all(cproto.Running.Before, maps.Values(j.podStates)...)
	if allPodsFound && allPodsAtLeastRunning && !j.sentRunningEvent {
		j.syslog.WithField("pod-name", podName).Info("pod is running")
		j.container.State = cproto.Running
		j.informTaskResourcesStarted(sproto.ResourcesStarted{
			NativeResourcesID: j.jobName,
			Addresses:         j.getGatewayAddresses(),
		})

		j.sentRunningEvent = true
	}

	if updatedPodState == cproto.Terminated && !j.podExits[podName] {
		j.syslog.WithField("pod-name", podName).Info("pod is terminated")
		exit, err := getExitCodeAndMessage(&updatedPod, j.containerNames)
		if err != nil {
			if updatedPod.ObjectMeta.DeletionTimestamp == nil {
				return err
			}
			// When a pod is deleted, it is possible that it will exit before the
			// determined containers generates an exit code. To check if this is
			// the case we check if a deletion timestamp has been set.
			exit = &exitReason{msg: "unable to get exit code or exit message from deleted pod"}
		}
		if !isSuccessfulExit(exit) {
			if j.jobExitCause == nil {
				j.jobExitCause = exit
			}
			j.syslog.
				WithField("code", exit.code).
				WithField("cause", j.jobExitCause).
				Infof("detected a determined containers crashed, cleaning up job: %s", exit.msg)
			j.killPod(podName)
		}
		j.podExits[podName] = true
	}

	if len(j.podExits) == j.numPods {
		if j.jobExitCause == nil {
			// Explicitly mark this case as a success before we delete the job.
			j.jobExitCause = &successfulExit
		}

		j.syslog.
			WithField("cause", j.jobExitCause).
			Infof("detected all determined containers exited, cleaning up job")
		j.kill()
	}

	return nil
}

func (j *job) podDeletedCallback(deleted *k8sV1.Pod) {
	j.mu.Lock()
	defer j.mu.Unlock()

	j.syslog.WithField("pod-name", deleted.Name).Info("pod deleted")
	if j.jobExitCause == nil {
		j.jobExitCause = &exitReason{
			failureType: sproto.TaskError,
			msg:         fmt.Sprintf("pod %s deleted", deleted.Name),
		}
	}
}

func (j *job) newEventCallback(event *k8sV1.Event) {
	j.mu.Lock()
	defer j.mu.Unlock()

	msgText := j.preparePodUpdateMessage(event.Message)
	message := fmt.Sprintf("%s %s: %s", event.InvolvedObject.Kind, event.InvolvedObject.Name, msgText)
	j.insertLog(event.CreationTimestamp.Time, message)
}

func (j *job) preemptionCallback() {
	j.syslog.Info("received preemption command")
	rmevents.Publish(j.allocationID, &sproto.ReleaseResources{Reason: "preempted by the scheduler"})
}

func (j *job) changePriority() {
	j.syslog.Info("interrupting job to change priorities")
	rmevents.Publish(j.allocationID, &sproto.ReleaseResources{Reason: "priority changed"})
}

func (j *job) Kill() {
	j.mu.Lock()
	defer j.mu.Unlock()

	j.syslog.Info("received request to stop job")
	if j.jobExitCause == nil {
		j.jobExitCause = &exitReason{msg: "killed"}
	}
	j.kill()
}

func (j *job) kill() {
	if !j.resourcesDeleted.CompareAndSwap(false, true) {
		return
	}

	var serviceNames, tcpRouteNames []string
	var gatewayPortsToFree []int
	for _, g := range j.gatewayProxyResources {
		serviceNames = append(serviceNames, g.serviceSpec.Name)
		tcpRouteNames = append(tcpRouteNames, g.tcpRouteSpec.Name)
		gatewayPortsToFree = append(gatewayPortsToFree, int(g.gatewayListener.Port))
	}

	j.syslog.Infof("requesting to delete kubernetes resources %s", j.jobName)
	j.resourceRequestQueue.deleteKubernetesResources(deleteKubernetesResources{
		namespace:          j.namespace,
		jobName:            j.jobName,
		configMapName:      j.configMapName,
		serviceNames:       serviceNames,
		tcpRouteNames:      tcpRouteNames,
		gatewayPortsToFree: gatewayPortsToFree,
	})
}

func (j *job) killPod(name string) {
	if j.podKillSent[name] {
		return
	}

	j.syslog.Infof("requesting to delete kubernetes resources %s", j.jobName)
	j.resourceRequestQueue.deleteKubernetesResources(deleteKubernetesResources{
		namespace: j.namespace,
		podName:   name,
	})
	j.podKillSent[name] = true
}

func (j *job) getNodeInfoForPods() []podNodeInfo {
	j.mu.Lock()
	defer j.mu.Unlock()

	var infos []podNodeInfo
	for _, nodeName := range j.podNodeNames {
		infos = append(infos, podNodeInfo{
			nodeName:  nodeName,
			numSlots:  j.slotsPerPod,
			slotType:  j.slotType,
			container: j.container.DeepCopy(),
		})
	}
	return infos
}

func (j *job) startPodLogStreamers() error {
	podList, err := j.podInterface.List(context.TODO(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", determinedLabel, j.req.AllocationID),
	})
	if err != nil {
		return fmt.Errorf("listing job pods to reattach log streamers: %w", err)
	}
	for _, pod := range podList.Items {
		if pod.Status.Phase != k8sV1.PodRunning {
			j.syslog.Warnf("skipped reattaching pod log streamer for pod %s in phase %s", pod.Name, pod.Status.Phase)
			continue
		}

		err := startPodLogStreamer(j.podInterface, pod.Name, func(log []byte) {
			j.receiveContainerLog(sproto.ContainerLog{
				Timestamp: time.Now().UTC(),
				RunMessage: &aproto.RunMessage{
					Value:   string(log),
					StdType: stdcopy.Stdout,
				},
			})
		})
		if err != nil {
			return fmt.Errorf("starting pod logs streamer for %s: %w", pod.Name, err)
		}
	}
	return nil
}

func (j *job) createSpecAndSubmit(spec *tasks.TaskSpec) error {
	jobSpec, configMapSpec, err := j.createSpec(j.scheduler, spec)
	if err != nil {
		return err
	}

	j.resourceRequestQueue.createKubernetesResources(jobSpec, configMapSpec, j.makeGatewayComms(spec))
	return nil
}

func (j *job) receiveResourceCreationFailed(msg resourceCreationFailed) {
	j.syslog.WithError(msg.err).Error("pod handler notified that resource creation failed")
	j.insertLog(time.Now().UTC(), msg.err.Error())
}

func (j *job) receiveResourceCreationCancelled() {
	j.syslog.Info("pod creation canceled")
	j.resourcesDeleted.Store(true)
}

func (j *job) receiveResourceDeletionFailed(msg resourceDeletionFailed) {
	j.syslog.WithError(msg.err).Error("pod handler notified that resource deletion failed")
}

func (j *job) informTaskResourcesState() {
	rmevents.Publish(j.allocationID, &sproto.ResourcesStateChanged{
		ResourcesID:    sproto.FromContainerID(j.container.ID),
		ResourcesState: sproto.FromContainerState(j.container.State),
	})
}

func (j *job) informTaskResourcesStarted(rs sproto.ResourcesStarted) {
	rmevents.Publish(j.allocationID, &sproto.ResourcesStateChanged{
		ResourcesID:      sproto.FromContainerID(j.container.ID),
		ResourcesState:   sproto.FromContainerState(j.container.State),
		ResourcesStarted: &rs,
	})
}

func (j *job) informTaskResourcesStopped() {
	if j.sentTerminationEvent {
		return
	}

	rmevents.Publish(j.allocationID, &sproto.ResourcesStateChanged{
		ResourcesID:      sproto.FromContainerID(j.container.ID),
		ResourcesState:   sproto.FromContainerState(j.container.State),
		ResourcesStopped: &sproto.ResourcesStopped{Failure: j.exitCause()},
	})
	j.sentTerminationEvent = true
}

func (j *job) receiveContainerLog(msg sproto.ContainerLog) {
	msg.ContainerID = j.container.ID
	rmevents.Publish(j.allocationID, &msg)
}

func (j *job) insertLog(timestamp time.Time, msg string) {
	j.receiveContainerLog(sproto.ContainerLog{
		Timestamp:  timestamp,
		AuxMessage: &msg,
	})
}

// Converts k8s message to be more understandable.
func (j *job) preparePodUpdateMessage(msgText string) string {
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
			required := strconv.Itoa(j.slotsPerPod)
			var resourceName string
			switch j.slotType {
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

// PodScheduled checks pod conditions to determine if a pod has been scheduled onto a node.
func podScheduled(pod k8sV1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == k8sV1.PodScheduled {
			return condition.Status == k8sV1.ConditionTrue
		}
	}
	return false
}

func (j *job) getPodState(pod k8sV1.Pod) (cproto.State, error) {
	switch pod.Status.Phase {
	case k8sV1.PodPending:
		// When pods are deleted, Kubernetes sometimes transitions pod statuses to pending
		// prior to deleting them. In these cases we have observed that we do not always
		// receive a PodFailed or a PodSucceeded message. We check if pods have a set pod
		// deletion timestamp to see if this is the case.
		if pod.ObjectMeta.DeletionTimestamp != nil {
			j.syslog.Warn("marking pod as terminated due to deletion timestamp")
			return cproto.Terminated, nil
		}

		if podScheduled(pod) {
			return cproto.Starting, nil
		}
		return cproto.Assigned, nil

	case k8sV1.PodRunning:
		// Pods are in a running state as long as at least one container has not terminated.
		// We check the status of the Determined containers directly to determine if they
		// are still running.
		containerStatuses, err := getDeterminedContainersStatus(
			pod.Status.ContainerStatuses, j.containerNames)
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
		return "", fmt.Errorf("unexpected pod status %s for pod %s", pod.Status.Phase, pod.Name)
	}
}

func getExitCodeAndMessage(pod *k8sV1.Pod, containerNames set.Set[string]) (*exitReason, error) {
	if len(pod.Status.InitContainerStatuses) == 0 {
		return nil, fmt.Errorf("unexpected number of init containers when processing exit code for pod %s", pod.Name)
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
			return &exitReason{
				code: int(exitCode),
				msg:  errMessage,
			}, nil
		}
	}

	if len(pod.Status.ContainerStatuses) < len(containerNames) {
		return nil, fmt.Errorf("unexpected number of containers when processing exit code for pod %s", pod.Name)
	}

	containerStatuses, err := getDeterminedContainersStatus(
		pod.Status.ContainerStatuses, containerNames)
	if err != nil {
		return nil, err
	}

	for _, containerStatus := range containerStatuses {
		terminationStatus := containerStatus.State.Terminated
		if terminationStatus != nil {
			return &exitReason{
				code: int(terminationStatus.ExitCode),
				msg:  terminationStatus.Message,
			}, nil
		}
	}

	return nil, fmt.Errorf("unable to get exit code from pod %s", pod.Name)
}

func getDeterminedContainersStatus(
	statuses []k8sV1.ContainerStatus,
	containerNames set.Set[string],
) ([]*k8sV1.ContainerStatus, error) {
	containerStatuses := make([]*k8sV1.ContainerStatus, 0, len(statuses))
	for idx, containerStatus := range statuses {
		if !containerNames.Contains(containerStatus.Name) {
			continue
		}
		containerStatuses = append(containerStatuses, &statuses[idx])
	}

	if len(containerStatuses) != len(containerNames) {
		containerNamesFound := make([]string, 0, len(containerStatuses))
		for _, containerStatus := range containerStatuses {
			containerNamesFound = append(containerNamesFound, containerStatus.Name)
		}
		return nil, fmt.Errorf("found container statuses only for: %v", containerNamesFound)
	}

	return containerStatuses, nil
}
