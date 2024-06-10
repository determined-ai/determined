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

func stripIndexFromSharedName(n string) string {
	lastHyphenIndex := strings.LastIndex(n, "-")
	if lastHyphenIndex == -1 {
		return ""
	}

	return n[:lastHyphenIndex]
}

func generateListenerName(allocationID model.AllocationID, port int) string {
	return fmt.Sprintf("%d-determined-%s", port, allocationID)
}

func listenerIsDetermined(listenerName string) bool {
	return strings.Contains(listenerName, "determined-")
}

func getAllocationIDFromListenerName(listenerName string) string {
	if !listenerIsDetermined(listenerName) {
		return ""
	}

	lastHyphenIndex := strings.LastIndex(listenerName, "determined-")
	if lastHyphenIndex == -1 {
		return ""
	}

	return listenerName[lastHyphenIndex+len("determined-"):]
}
