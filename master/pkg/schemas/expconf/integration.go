package expconf

// IntegrationConfigV0 configures experiment-level integrations.
type IntegrationConfigV0 struct {
	Pachyderm PachydermConfigV0 `json:"pachyderm"`
}

// PachydermConfigV0 configures experiments with fields relevant to the pachyderm integration.
type PachydermConfigV0 struct {
	Host    *string `json:"host"`
	Port    *int    `json:"port"`
	Project *string `json:"project"`
	Repo    *string `json:"repo"`
	Commit  *string `json:"previous_commit"`
	Branch  *string `json:"branch"`
}
