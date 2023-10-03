package config

// TelemetryConfig is the configuration for telemetry.
type TelemetryConfig struct {
	Enabled                  bool   `json:"enabled"`
	SegmentMasterKey         string `json:"segment_master_key"`
	OtelEnabled              bool   `json:"otel_enabled"`
	OtelExportedOtlpEndpoint string `json:"otel_endpoint"`
	SegmentWebUIKey          string `json:"segment_webui_key"`
	ClusterID                string `json:"cluster_id"`
}
