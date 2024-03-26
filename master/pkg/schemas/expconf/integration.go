package expconf

// IntegrationConfigV0 configures experiment-level integrations.
type IntegrationConfigV0 struct {
	Pachyderm *PachydermConfigV0 `json:"pachyderm"`
}

// PachydermConfigV0 configures experiments with fields relevant to the pachyderm integration.
type PachydermConfigV0 struct {
	PachdConfig *PachydermPachdConfigV0 `json:"pachd"`
	ProxyConfig *PachydermProxyConfigV0 `json:"proxy"`
	Project     *string                 `json:"project"`
	Repo        *string                 `json:"repo"`
	Commit      *string                 `json:"previous_commit"`
	Branch      *string                 `json:"branch"`
}

// PachydermPachdConfigV0 configures the fields relevant to the pachyderm daemon.
type PachydermPachdConfigV0 struct {
	Host  *string `json:"host"`
	Port  *string `json:"port"`
	Token *string `json:"token"`
}

// PachydermProxyConfigV0 configures the fields relevant to the pachyderm console proxy.
type PachydermProxyConfigV0 struct {
	Host *string `json:"host"`
	Port *string `json:"port"`
}
