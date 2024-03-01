package expconf

// IntegrationConfigV0 configures experiment-level integrations.
type IntegrationConfigV0 struct {
	Pachyderm *PachydermConfigV0 `json:"pachyderm"`
}

// PachydermConfigV0 configures experiments with fields relevant to the pachyderm integration.
type PachydermConfigV0 struct {
	PachdConfig   *PachydermPachdConfigV0   `json:"pachd"`
	ProxyConfig   *PachydermProxyConfigV0   `json:"proxy"`
	DatasetConfig *PachydermDatasetConfigV0 `json:"dataset"`
}

// PachydermPachdConfigV0 configures the fields relevant to the pachyderm daemon.
type PachydermPachdConfigV0 struct {
	Host  *string `json:"host"`
	Port  *int    `json:"port"`
	Token *string `json:"token"`
}

// PachydermProxyConfigV0 configures the fields relevant to the pachyderm console proxy.
type PachydermProxyConfigV0 struct {
	Scheme *string `json:"scheme"`
	Host   *string `json:"host"`
	Port   *int    `json:"port"`
}

// PachydermDatasetConfigV0 configures the fields relevant to the pachyderm dataset.
type PachydermDatasetConfigV0 struct {
	Project *string `json:"project"`
	Repo    *string `json:"repo"`
	Commit  *string `json:"commit"`
	Branch  *string `json:"branch"`
}
