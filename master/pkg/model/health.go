package model

// HealthStatus is the up or down informational status.
type HealthStatus string

const (
	// Healthy indicates passing the health check.
	Healthy HealthStatus = "up"
	// Unhealthy indicates failing the health check.
	Unhealthy HealthStatus = "down"
)

// HealthCheck is the response to the health check request.
type HealthCheck struct {
	Status           HealthStatus            `json:"status"`
	Database         HealthStatus            `json:"database"`
	ResourceManagers []ResourceManagerHealth `json:"resource_managers"`
}

// ResourceManagerHealth is a pair of resource manager name and health status.
type ResourceManagerHealth struct {
	ClusterName string       `json:"cluster_name"`
	Status      HealthStatus `json:"status"`
}
