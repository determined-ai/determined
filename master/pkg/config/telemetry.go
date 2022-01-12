package config

// TelemetryConfig is the configuration for telemetry.
type TelemetryConfig struct {
	Enabled          bool   `json:"enabled"`
	SegmentMasterKey string `json:"segment_master_key"`
	SegmentWebUIKey  string `json:"segment_webui_key"`
}
