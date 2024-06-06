# Internal Task Gateway


- limit of port range


## Configuration


### Master Configuraiton
config explanation.

an optional config.
Sitting under `internal_task_gateway` key under each resource manager config in master config.

represented by go package `config.InternalTaskGatewayConfig`

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


- limit of listeners in gateway CRD setup
    - max aux containers < min(this and port range)
- developer docs on how to test and use
- master config docs
- update setup guide on multirm 
- release note
- update k8s architecture docs? to include we will deploy services / routes / 
- include example doc for deploying a gateway controller
- update Carolina’s bug bash docs to include notebook testing
- supported controllers list
