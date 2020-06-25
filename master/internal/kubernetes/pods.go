package kubernetes

import (
	"reflect"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/check"

	"github.com/labstack/echo"
	k8sclient "k8s.io/client-go/kubernetes"

	// Used to load all auth plugins.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

type pods struct {
	cluster        *actor.Ref
	namespace      string
	outOfCluster   bool
	kubeConfigPath string

	clientSet *k8sclient.Clientset
}

// Initialize creates a new global agent actor.
func Initialize(
	s *actor.System,
	_ *echo.Echo,
	c *actor.Ref,
	namespace string,
	outOfCluster bool,
	kubeConfigPath string,
) *actor.Ref {
	podsActor, ok := s.ActorOf(actor.Addr("pods"), &pods{
		cluster:        c,
		namespace:      namespace,
		outOfCluster:   outOfCluster,
		kubeConfigPath: kubeConfigPath,
	})
	check.Panic(check.True(ok, "pods address already taken"))

	// TODO (DET-3424) Configure endpoints.
	//e.Any("/agents*", api.Route(s))

	return podsActor
}

func (p *pods) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		p.startClientSet(ctx)

	default:
		ctx.Log().Error("Unexpected message: ", reflect.TypeOf(msg))
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (p *pods) startClientSet(ctx *actor.Context) {
	var config *rest.Config
	var err error

	// TODO: Remove out of cluster config
	if p.outOfCluster {
		config, err = clientcmd.BuildConfigFromFlags("", p.kubeConfigPath)
		if err != nil {
			ctx.Log().Error("Error building kubernetes config", err)
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			ctx.Log().Error("Error building kubernetes config", err)
		}
	}

	p.clientSet, err = k8sclient.NewForConfig(config)
	if err != nil {
		ctx.Log().Error("Error initializing kubernetes clientSet", err)
	}

	ctx.Log().Infof("kubernetes clientSet initialized")
}
