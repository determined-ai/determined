package scheduler

// View is a partial view of the current state of the tasks and agents of the cluster.
type View interface {
	Update(rp *DefaultRP) (ViewSnapshot, bool)
}

// ViewSnapshot is an immutable snapshot of a View.
type ViewSnapshot struct {
	Tasks           []*TaskSummary
	ConnectedAgents []*AgentSummary
	IdleAgents      []*AgentSummary
}
