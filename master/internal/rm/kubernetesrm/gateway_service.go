package kubernetesrm

import (
	"context"
	"fmt"
	"slices"
	"sync"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	gatewayTyped "sigs.k8s.io/gateway-api/apis/v1"
	gateway "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/typed/apis/v1"

	alphaGateway "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/typed/apis/v1alpha2"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/pkg/model"
)

// I wanted to do this all in patches, but Gateways don't yet support strategic merge patch.
// Its pretty easy to use json patch to add the port, but removing the port is a lot harder
// as I don't think you can remove by value. Instead just serialize reading then submitting
// updates on the gateway.
type gatewayService struct {
	mu                 sync.Mutex
	gatewayInterface   gateway.GatewayInterface
	tcpRouteInterfaces map[string]alphaGateway.TCPRouteInterface
	gatewayName        string
	portRangeStart     int
	portRangeEnd       int
}

// chat: these probably could use a better names or packaging.
type gatewayResourceComm struct {
	requestedPorts     int
	resourceDescriptor proxyResourceGenerator
	allocationID       model.AllocationID
	reportResources    func([]gatewayProxyResource)
}

func newGatewayService(
	gatewayInterface gateway.GatewayInterface,
	tcpRouteInterfaces map[string]alphaGateway.TCPRouteInterface,
	taskGWConfig config.InternalTaskGatewayConfig,
) (*gatewayService, error) {
	// DOCS: note this limit on number of active proxied tasks.
	// TODO: validate existing port on the gateway on startup. Maybe? Restore could cover this?
	g := &gatewayService{
		gatewayInterface:   gatewayInterface,
		tcpRouteInterfaces: tcpRouteInterfaces,
		gatewayName:        taskGWConfig.GatewayName,
		portRangeStart:     taskGWConfig.GWPortStart,
		portRangeEnd:       taskGWConfig.GWPortEnd,
	}
	return g, nil
}

// PortMap is a map of internal pod ports to external gateway ports.
type PortMap map[int]int

func (g *gatewayService) generateAndAddListeners(allocationID model.AllocationID, count int) ([]int, error) {
	var ports []int
	if err := g.updateGateway(func(gateway *gatewayTyped.Gateway) error {
		var err error
		listeners := make([]gatewayTyped.Listener, count)
		ports, err = g.pickNFreePorts(gateway, len(listeners))
		if err != nil {
			return err
		}

		if gateway.Annotations == nil {
			gateway.Annotations = make(map[string]string)
		}

		for i := 0; i < count; i++ {
			gateway.Annotations[portToAnnotationKey(ports[i])] = string(allocationID)
			listeners[i] = createListenerForPod(ports[i])
		}
		gateway.Spec.Listeners = append(gateway.Spec.Listeners, listeners...)

		return nil
	}); err != nil {
		return nil, fmt.Errorf("adding %d listeners to gateway: %w", count, err)
	}
	return ports, nil
}

func (g *gatewayService) pickNFreePorts(gateway *gatewayTyped.Gateway, count int) ([]int, error) {
	usedPorts := make(map[int]struct{}, len(gateway.Spec.Listeners))
	for _, listener := range gateway.Spec.Listeners {
		usedPorts[int(listener.Port)] = struct{}{}
	}
	ports := make([]int, 0, count)
	for port := g.portRangeStart; port <= g.portRangeEnd; port++ {
		if _, used := usedPorts[port]; !used {
			ports = append(ports, port)
			if len(ports) == count {
				return ports, nil
			}
		}
	}
	return nil, fmt.Errorf("not enough free ports in range %d-%d", g.portRangeStart, g.portRangeEnd)
}

func (g *gatewayService) freePorts(ports []int) error {
	if err := g.updateGateway(func(gateway *gatewayTyped.Gateway) error {
		if gateway.Annotations == nil {
			gateway.Annotations = make(map[string]string)
		}

		var newListeners []gatewayTyped.Listener
		for _, l := range gateway.Spec.Listeners {
			if slices.Contains(ports, int(l.Port)) {
				delete(gateway.Annotations, portToAnnotationKey(int(l.Port)))
			} else {
				newListeners = append(newListeners, l)
			}
		}

		gateway.Spec.Listeners = newListeners
		return nil
	}); err != nil {
		return fmt.Errorf("freeing ports %v from gateway: %w", ports, err)
	}
	return nil
}

func (g *gatewayService) getProxyPorts(allocationID *model.AllocationID) ([]int, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	gateway, err := g.gatewayInterface.Get(context.TODO(), g.gatewayName, metaV1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting gateway for proxy ports: %w", err)
	}

	var ports []int
	for _, listener := range gateway.Spec.Listeners {
		portAllocationID := gateway.Annotations[portToAnnotationKey(int(listener.Port))]
		if portAllocationID == "" {
			continue // Don't touch ports that we didn't add.
		}
		if allocationID != nil && string(*allocationID) != portAllocationID {
			continue
		}

		ports = append(ports, int(listener.Port))
	}

	return ports, nil
}

func (g *gatewayService) updateGateway(update func(*gatewayTyped.Gateway) error) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		gateway, err := g.gatewayInterface.Get(context.TODO(), g.gatewayName, metaV1.GetOptions{})
		if err != nil {
			return err
		}

		if err = update(gateway); err != nil {
			return err
		}

		if _, err := g.gatewayInterface.
			Update(context.TODO(), gateway, metaV1.UpdateOptions{}); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return fmt.Errorf("updating gateway with name '%s': %w", g.gatewayName, err)
	}

	return nil
}
