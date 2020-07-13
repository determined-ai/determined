// Package kubernetes handles all interaction with the Kubernetes API including starting
// and stopping tasks, monitoring their status, and fetching logs.
package kubernetes

import (
	"fmt"

	"github.com/labstack/echo"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/check"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sClient "k8s.io/client-go/kubernetes"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"

	// Used to load all auth plugins.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

type podMetadata struct {
	podName     string
	containerID string
}

type pods struct {
	cluster                  *actor.Ref
	namespace                string
	masterServiceName        string
	leaveKubernetesResources bool

	clientSet  *k8sClient.Clientset
	masterIP   string
	masterPort int32

	informer                *actor.Ref
	podNameToPodHandler     map[string]*actor.Ref
	containerIDToPodHandler map[string]*actor.Ref
	podHandlerToMetadata    map[*actor.Ref]podMetadata

	podInterface       typedV1.PodInterface
	configMapInterface typedV1.ConfigMapInterface
}

// Initialize creates a new global agent actor.
func Initialize(
	s *actor.System,
	_ *echo.Echo,
	c *actor.Ref,
	namespace string,
	masterServiceName string,
	leaveKubernetesResources bool,
) *actor.Ref {
	podsActor, ok := s.ActorOf(actor.Addr("pods"), &pods{
		cluster:                  c,
		namespace:                namespace,
		masterServiceName:        masterServiceName,
		podNameToPodHandler:      make(map[string]*actor.Ref),
		containerIDToPodHandler:  make(map[string]*actor.Ref),
		podHandlerToMetadata:     make(map[*actor.Ref]podMetadata),
		leaveKubernetesResources: leaveKubernetesResources,
	})
	check.Panic(check.True(ok, "pods address already taken"))

	// TODO (DET-3424): Configure endpoints.
	//e.Any("/agents*", api.Route(s))

	return podsActor
}

func (p *pods) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		if err := p.startClientSet(ctx); err != nil {
			return err
		}
		if err := p.getMasterIPAndPort(ctx); err != nil {
			return err
		}
		if err := p.deleteExistingKubernetesResources(ctx); err != nil {
			return err
		}
		p.startPodInformer(ctx)

	case sproto.StartPod:
		if err := p.receiveStartPod(ctx, msg); err != nil {
			return err
		}

	case podStatusUpdate:
		p.receivePodStatusUpdate(ctx, msg)

	case sproto.StopPod:
		p.receiveStopPod(ctx, msg)

	case actor.ChildStopped:
		if err := p.cleanUpPodHandler(ctx, msg.Child); err != nil {
			return err
		}

	case actor.ChildFailed:
		if msg.Child == p.informer {
			return errors.Errorf("pod informer failed")
		}

		if err := p.cleanUpPodHandler(ctx, msg.Child); err != nil {
			return err
		}

	default:
		ctx.Log().Errorf("unexpected message %T", msg)
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (p *pods) startClientSet(ctx *actor.Context) error {
	config, err := rest.InClusterConfig()
	if err != nil {
		return errors.Wrap(err, "error building kubernetes config")
	}

	p.clientSet, err = k8sClient.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "failed to initialize kubernetes clientSet")
	}

	p.podInterface = p.clientSet.CoreV1().Pods(p.namespace)
	p.configMapInterface = p.clientSet.CoreV1().ConfigMaps(p.namespace)

	ctx.Log().Infof("kubernetes clientSet initialized")
	return nil
}

func (p *pods) getMasterIPAndPort(ctx *actor.Context) error {
	masterService, err := p.clientSet.CoreV1().Services(p.namespace).Get(
		p.masterServiceName, metaV1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get master service")
	}

	p.masterIP = masterService.Spec.ClusterIP
	p.masterPort = masterService.Spec.Ports[0].Port
	ctx.Log().Infof("master URL set to %s:%d", p.masterIP, p.masterPort)
	return nil
}

func (p *pods) deleteExistingKubernetesResources(ctx *actor.Context) error {
	listOptions := metaV1.ListOptions{LabelSelector: determinedLabel}
	var gracePeriod int64 = 15
	deleteOptions := &metaV1.DeleteOptions{GracePeriodSeconds: &gracePeriod}

	configMaps, err := p.configMapInterface.List(listOptions)
	if err != nil {
		return errors.Wrap(err, "error listing existing config maps")
	}
	for _, configMap := range configMaps.Items {
		if configMap.Namespace != p.namespace {
			continue
		}

		ctx.Log().WithField("name", configMap.Name).Info("deleting configMap")
		if err = p.configMapInterface.Delete(configMap.Name, deleteOptions); err != nil {
			return errors.Wrapf(err, "error deleting configMap: %s", configMap.Name)
		}
	}

	pods, err := p.podInterface.List(listOptions)
	if err != nil {
		return errors.Wrap(err, "error listing existing pod")
	}
	for _, pod := range pods.Items {
		if pod.Namespace != p.namespace {
			continue
		}

		ctx.Log().WithField("name", pod.Name).Info("deleting pod")
		if err = p.podInterface.Delete(pod.Name, deleteOptions); err != nil {
			return errors.Wrapf(err, "error deleting pod: %s", pod.Name)
		}
	}

	return nil
}

func (p *pods) startPodInformer(ctx *actor.Context) {
	p.informer, _ = ctx.ActorOf("pod-informer", newInformer(p.podInterface, p.namespace, ctx.Self()))
	ctx.Tell(p.informer, startInformer{})
}

func (p *pods) receiveStartPod(ctx *actor.Context, msg sproto.StartPod) error {
	newPodHandler := newPod(
		p.cluster, msg.TaskHandler, p.clientSet, p.namespace, p.masterIP,
		p.masterPort, msg.Spec, msg.Slots, msg.Rank, p.podInterface,
		p.configMapInterface, p.leaveKubernetesResources,
	)
	ref, ok := ctx.ActorOf(fmt.Sprintf("pod-%s-%d", msg.Spec.TaskID, msg.Rank), newPodHandler)
	if !ok {
		return errors.Errorf("pod actor %s already exists", ref.Address().String())
	}

	ctx.Log().WithField("pod", newPodHandler.podName).WithField(
		"handler", ref.Address()).Infof("registering pod handler")

	if _, alreadyExists := p.podNameToPodHandler[newPodHandler.podName]; alreadyExists {
		return errors.Errorf(
			"attempting to register same pod name: %s multiple times", newPodHandler.podName)
	}

	p.podNameToPodHandler[newPodHandler.podName] = ref
	p.containerIDToPodHandler[msg.Spec.ContainerID] = ref
	p.podHandlerToMetadata[ref] = podMetadata{
		podName:     newPodHandler.podName,
		containerID: msg.Spec.ContainerID,
	}
	return nil
}

func (p *pods) receivePodStatusUpdate(ctx *actor.Context, msg podStatusUpdate) {
	ref, ok := p.podNameToPodHandler[msg.updatedPod.Name]
	if !ok {
		ctx.Log().WithField("pod-name", msg.updatedPod.Name).Warn(
			"received pod status update for un-registered pod")
		return
	}

	ctx.Tell(ref, msg)
}

func (p *pods) receiveStopPod(ctx *actor.Context, msg sproto.StopPod) {
	ref, ok := p.containerIDToPodHandler[msg.ContainerID]
	if !ok {
		// For multi-pod tasks, when the the chief pod exits,
		// the scheduler will request to terminate pods all other pods
		// that have notified the scheduler that they have exited.
		ctx.Log().WithField("container-id", msg.ContainerID).Info(
			"received stop pod command for unregistered container id")
		return
	}

	ctx.Tell(ref, msg)
}

func (p *pods) cleanUpPodHandler(ctx *actor.Context, podHandler *actor.Ref) error {
	podInfo, ok := p.podHandlerToMetadata[podHandler]
	if !ok {
		return errors.Errorf("unknown pod handler being deleted %s", podHandler.Address())
	}

	ctx.Log().WithField("pod", podInfo.podName).WithField(
		"handler", podHandler.Address()).Infof("de-registering pod handler")
	delete(p.podNameToPodHandler, podInfo.podName)
	delete(p.containerIDToPodHandler, podInfo.containerID)
	delete(p.podHandlerToMetadata, podHandler)

	return nil
}
