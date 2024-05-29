package kubernetesrm

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	gatewayTyped "sigs.k8s.io/gateway-api/apis/v1"
	gateway "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/typed/apis/v1"

	alphaGateway "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/typed/apis/v1alpha2"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/sergi/go-diff/diffmatchpatch"
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
	reportResources    func([]gatewayProxyResource)
}

func genSectionName(gwPort int) string {
	return fmt.Sprintf("section-%d", gwPort)
}

func newGatewayService(
	gatewayInterface gateway.GatewayInterface,
	tcpRouteInterfaces map[string]alphaGateway.TCPRouteInterface,
	gatewayName string,
) (*gatewayService, error) {
	// TODO: make port range configurable by user. We currently assume we own the controller and
	// the service.
	// DOCS: note this limit on number of active proxied tasks.
	// TODO: validate existing port on the gateway on startup
	g := &gatewayService{
		gatewayInterface:   gatewayInterface,
		tcpRouteInterfaces: tcpRouteInterfaces,
		gatewayName:        gatewayName,
		portRangeStart:     49152,
		portRangeEnd:       65535,
	}
	return g, nil
}

// PortMap is a map of internal pod ports to external gateway ports.
type PortMap map[int]int

func (g *gatewayService) generateAndAddListeners(count int) ([]int, error) {
	var ports []int
	if err := g.updateGateway(func(gateway *gatewayTyped.Gateway) error {
		var err error
		listeners := make([]gatewayTyped.Listener, count)
		ports, err = g.pickNFreePorts(gateway, count)
		if err != nil {
			return err
		}
		for i := 0; i < count; i++ {
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
		var newListeners []gatewayTyped.Listener
		for _, l := range gateway.Spec.Listeners {
			if !slices.Contains(ports, int(l.Port)) {
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

// getDeployedPortMap returns a mapping of ports based on gw config in the cluster.
func (g *gatewayService) getDeployedPortMap() (map[model.AllocationID]PortMap, error) {
	rv := make(map[model.AllocationID]PortMap)

	for namespace, tcpRouteInterface := range g.tcpRouteInterfaces {
		tcpRoutes, err := tcpRouteInterface.List(context.TODO(), metaV1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("listing TCPRoutes in namespace '%s': %w", namespace, err)
		}

		for _, route := range tcpRoutes.Items {
			allocIDStr, ok := route.Labels[determinedLabel]
			if !ok {
				continue
			}
			allocID := model.AllocationID(allocIDStr)
			for _, rule := range route.Spec.Rules {
				for _, backendRef := range rule.BackendRefs {
					if _, ok := rv[allocID]; !ok {
						rv[allocID] = make(PortMap)
					}
					if backendRef.Port == nil {
						return nil, fmt.Errorf("TCPRoute '%s' has a nil port", route.Name)
					}
					if len(route.Spec.ParentRefs) != 1 {
						return nil, fmt.Errorf("TCPRoute '%s' has %d parent refs, expected 1",
							route.Name, len(route.Spec.ParentRefs))
					}
					rv[allocID][int(*backendRef.Port)] = int(*route.Spec.ParentRefs[0].Port)
				}
			}
		}
	}
	return rv, nil
}

func diffGateways(old, new *gatewayTyped.Gateway) (string, error) {
	oldJSON, err := json.MarshalIndent(old, "", "  ")
	if err != nil {
		return "", err
	}

	newJSON, err := json.MarshalIndent(new, "", "  ")
	if err != nil {
		return "", err
	}

	return diff(oldJSON, newJSON), nil
}

func diff(a, b []byte) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(string(a), string(b), false)
	return dmp.DiffPrettyText(diffs)
}

func isConflictError(err error) bool {
	// statusErr, ok := err.(metaV1.StatusReason)
	// if !ok {
	//     return false
	// }
	// return statusErr == metaV1.StatusReasonConflict
	msg := "the object has been modified; please apply your changes to the latest version"
	return strings.Contains(err.Error(), msg)
}

func (g *gatewayService) updateGateway(update func(*gatewayTyped.Gateway) error) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	backoff := wait.Backoff{
		Duration: 1 * time.Second,
		Factor:   2.0,
		Jitter:   0.1,
		Steps:    2,
	}

	return retry.OnError(backoff, isConflictError, func() error {
		gateway, err := g.gatewayInterface.Get(context.TODO(), g.gatewayName, metaV1.GetOptions{})
		if err != nil {
			return fmt.Errorf("getting gateway with name '%s': %w", g.gatewayName, err)
		}

		err = update(gateway)
		if err != nil {
			return err
		}

		if _, err := g.gatewayInterface.Update(context.TODO(), gateway, metaV1.UpdateOptions{}); err != nil {
			fmt.Printf("gwservice: ran into error: %v\n", err)
			return fmt.Errorf("updating gateway with name '%s': %w", g.gatewayName, err)
		}

		return nil
	})
	// gateway, err := g.gatewayInterface.Get(context.TODO(), g.gatewayName, metaV1.GetOptions{})
	// if err != nil {
	// 	return fmt.Errorf("getting gateway with name '%s': %w", g.gatewayName, err)
	// }

	// err = update(gateway)
	// if err != nil {
	// 	return err
	// }

	// if updatedGateway, err := g.gatewayInterface.
	// 	Update(context.TODO(), gateway, metaV1.UpdateOptions{}); err != nil {
	// 	fmt.Printf("received gateway: %v\n", updatedGateway)
	// 	gwDiff, diffErr := diffGateways(gateway, updatedGateway)
	// 	if diffErr == nil {
	// 		fmt.Printf("diff: %s\n", gwDiff)
	// 	}
	// 	return fmt.Errorf("updating gateway with name '%s': %w", g.gatewayName, err)
	// }

	// return nil
}
