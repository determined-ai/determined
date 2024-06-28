package kubernetesrm

import (
	"fmt"

	gatewayTyped "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/determined-ai/determined/master/pkg/ptrs"
)

func generateListenerName(p int) string {
	return fmt.Sprintf("det-%d", p)
}

func createListenerForPod(gwPort int) gatewayTyped.Listener {
	gatewayListener := gatewayTyped.Listener{
		Name:     gatewayTyped.SectionName(generateListenerName(gwPort)),
		Port:     gatewayTyped.PortNumber(gwPort),
		Protocol: "TCP",
		AllowedRoutes: &gatewayTyped.AllowedRoutes{
			Namespaces: &gatewayTyped.RouteNamespaces{
				From: ptrs.Ptr(gatewayTyped.NamespacesFromAll),
			},
		},
	}
	return gatewayListener
}

func portToAnnotationKey(port int) string {
	return fmt.Sprintf("determined.ai/det-gateway-port-%d", port)
}
