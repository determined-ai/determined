package agentrm

// TODO(!!!): unexport (remove, too)

type (
	// AddAgent adds the agent to the cluster.
	AddAgent struct {
		Agent *agent
		Slots int
	}
	// RemoveAgent removes the agent from the cluster.
	RemoveAgent struct {
		Agent *agent
	}
	// UpdateAgent notifies the RP on scheduling-related changes in the agent.
	UpdateAgent struct {
		Agent *agent
	}
)
