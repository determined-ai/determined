package kubernetesrm

import (
	gatewayTyped "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/determined-ai/determined/master/pkg/ptrs"
)

func createListenerForPod(gwPort int) gatewayTyped.Listener {
	gatewayListener := gatewayTyped.Listener{
		Name:     gatewayTyped.SectionName(genSectionName(gwPort)),
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
