package kubernetes

import (
	"github.com/determined-ai/determined/master/pkg/actor"

	k8sV1 "k8s.io/api/core/v1"
	k8Informers "k8s.io/client-go/informers"
	k8sClient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// Messages sent by the nodeInformer to itself.
type (
	startNodeInformer struct{}
)

// Messages sent by the nodeInformer to the podsHandler.
type (
	nodeStatusUpdate struct {
		updatedNode *k8sV1.Node
		deletedNode *k8sV1.Node
	}
)

type nodeInformer struct {
	informer    k8Informers.SharedInformerFactory
	podsHandler *actor.Ref
	stop        chan struct{}
}

func newNodeInformer(clientSet k8sClient.Interface, podsHandler *actor.Ref) *nodeInformer {
	return &nodeInformer{
		informer: k8Informers.NewSharedInformerFactoryWithOptions(
			clientSet, 0, []k8Informers.SharedInformerOption{}...),
		podsHandler: podsHandler,
		stop:        make(chan struct{}),
	}
}

func (n *nodeInformer) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		ctx.Tell(ctx.Self(), startNodeInformer{})

	case startNodeInformer:
		if err := n.startNodeInformer(ctx); err != nil {
			return err
		}

	case actor.PostStop:
		ctx.Log().Infof("shutting down node informer")
		close(n.stop)

	default:
		ctx.Log().Errorf("unexpected message %T", msg)
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

func (n *nodeInformer) startNodeInformer(ctx *actor.Context) error {
	nodeInformer := n.informer.Core().V1().Nodes().Informer()
	nodeInformer.AddEventHandler(&cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			node, ok := obj.(*k8sV1.Node)
			if ok {
				ctx.Log().Debugf("node added %s", node.Name)
				ctx.Tell(n.podsHandler, nodeStatusUpdate{updatedNode: node})
			} else {
				ctx.Log().Warnf("error converting event of type %T to *k8sV1.Node", obj)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			node, ok := newObj.(*k8sV1.Node)
			if ok {
				ctx.Log().Debugf("node updated %s", node.Name)
				ctx.Tell(n.podsHandler, nodeStatusUpdate{updatedNode: node})
			} else {
				ctx.Log().Warnf("error converting event of type %T to *k8sV1.Node", newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			node, ok := obj.(*k8sV1.Node)
			if ok {
				ctx.Log().Debugf("node stopped %s", node.Name)
				ctx.Tell(n.podsHandler, nodeStatusUpdate{deletedNode: node})
			} else {
				ctx.Log().Warnf("error converting event of type %T to *k8sV1.Node", obj)
			}
		},
	})

	ctx.Log().Debug("starting node informer")
	n.informer.Start(n.stop)
	for !nodeInformer.HasSynced() {
	}
	ctx.Log().Info("node informer has started")

	return nil
}
