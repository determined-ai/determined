package kubernetesrm

import (
	"fmt"
	"strings"

	gatewayTyped "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
)

func createListenerForPod(allocationID model.AllocationID, gwPort int) gatewayTyped.Listener {
	gatewayListener := gatewayTyped.Listener{
		Name:     gatewayTyped.SectionName(generateListenerName(allocationID, gwPort)),
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

func generateListenerName(allocationID model.AllocationID, port int) string {
	return fmt.Sprintf("%d-det-%s", port, allocationID)
}

func getAllocationIDFromListenerName(listenerName string) string {
	if !strings.Contains(listenerName, "det") {
		return ""
	}

	lastHyphenIndex := strings.LastIndex(listenerName, "det-")
	if lastHyphenIndex == -1 {
		return ""
	}

	return listenerName[lastHyphenIndex+len("det-"):]
}
