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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"

	// Used to load all auth plugins.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

type pods struct {
	cluster           *actor.Ref
	namespace         string
	masterServiceName string

	clientSet  *k8sclient.Clientset
	masterIP   string
	masterPort int32

	informer            *actor.Ref
	podNameToPodHandler map[string]*actor.Ref

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
) *actor.Ref {
	podsActor, ok := s.ActorOf(actor.Addr("pods"), &pods{
		cluster:             c,
		namespace:           namespace,
		masterServiceName:   masterServiceName,
		podNameToPodHandler: make(map[string]*actor.Ref),
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
		p.startPodInformer(ctx)

	case sproto.StartPod:
		if err := p.receiveStartPod(ctx, msg); err != nil {
			return err
		}

	case podStatusUpdate:
		p.receivePodStatusUpdate(ctx, msg)

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

	p.clientSet, err = k8sclient.NewForConfig(config)
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
		p.masterServiceName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get master service")
	}

	p.masterIP = masterService.Spec.ClusterIP
	p.masterPort = masterService.Spec.Ports[0].Port
	ctx.Log().Infof("master URL set to %s:%d", p.masterIP, p.masterPort)
	return nil
}

func (p *pods) startPodInformer(ctx *actor.Context) {
	p.informer, _ = ctx.ActorOf("pod-informer", newInformer(p.podInterface, p.namespace, ctx.Self()))
	ctx.Tell(p.informer, startInformer{})
}

func (p *pods) receiveStartPod(ctx *actor.Context, msg sproto.StartPod) error {
	newPodHandler := newPod(
		p.cluster, msg.TaskHandler, p.clientSet, p.namespace, p.masterIP,
		p.masterPort, msg.Spec, msg.Slots, msg.Rank,
		p.podInterface, p.configMapInterface,
	)
	ref, ok := ctx.ActorOf(fmt.Sprintf("pod-%s", msg.Spec.TaskID), newPodHandler)
	if !ok {
		return errors.Errorf("pod actor %s already exists", ref.Address().String())
	}

	p.podNameToPodHandler[newPodHandler.podName] = ref
	return nil
}

func (p *pods) receivePodStatusUpdate(ctx *actor.Context, msg podStatusUpdate) {
	ref, ok := p.podNameToPodHandler[msg.podName]
	if !ok {
		ctx.Log().Errorf("received pod status update for un-registered pod: %s", msg.podName)
		return
	}

	ctx.Tell(ref, msg)
}
