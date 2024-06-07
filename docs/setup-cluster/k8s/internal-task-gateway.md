# Internal Task Gateway


## Controller Support - Requirements
supported controllers list

### sample setup
include example doc for deploying a gateway controller

## Configuration

- total active proxies will be limited by: min(maxItems, portRange) (not exhaustive).

### Gateway
In the CRD `gateways.gateway.networking.k8s.io`
`schema.openAPIV3Schema.properties.spec.properties.listeners.maxItems` defines a max limit of how many
listeners can be active on a single gateway. This limit sets the upper bound on how many tasks can be actively proxied.

note k8s validation complexity cost estimates.

- limit of listeners in gateway CRD setup
### Master Configuration
config explanation.

an optional config.
Sitting under `internal_task_gateway` key under each resource manager config in master config.

represented by Go package `config.InternalTaskGatewayConfig` probably defined in `master/internal/config/resource_manager_config.go`

```go
// InternalTaskGatewayConfig is config for exposing Determined tasks to outside of the cluster.
// Useful for multirm when we can only be running in a single cluster.
type InternalTaskGatewayConfig struct {
	// GatewayName as defined in the k8s cluster.
	GatewayName string `json:"gateway_name"`
	// GatewayNamespace as defined in the k8s cluster.
	GatewayNamespace string `json:"gateway_namespace"`
	GatewayIP        string `json:"gateway_ip"`
	// GWPortStart denotes the inclusive start of the available and exclusive port range to
	// MLDE for InternalTaskGateway.
	GWPortStart int `json:"gateway_port_range_start"`
	// GWPortEnd denotes the inclusive end of the available and exclusive port range to
	// MLDE for InternalTaskGateway.
	GWPortEnd int `json:"gateway_port_range_end"`
}
```

valid port range starts from 1025 to 65535 inclusive.

- CHECK: might wanna set max aux containers < min(this and port range)

## Dev Docs
- developer docs on how to test and use

## Release Notes
- mention docs
- mention current limits if any


## TODO
- update setup guide on multirm 
- update k8s architecture docs? to include we will deploy services / routes / 
- update Carolina’s bug bash docs to include notebook testing
