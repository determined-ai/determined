package ft

import (
	"github.com/determined-ai/determined/master/internal/sproto"
)

/*
write through cache the alerts. or nvm just use the db or in memory for first milestone.
*/

// NodeBlacklist returns a list of nodes that should be blacklisted for the given allocation
func NodeBlacklist(ar *sproto.AllocateRequest) ([]string, error) {
	// maybe just taskid is enough. go off of task id and GetAlerts
	return nil, nil
}

func UserOwnsTask(userID, taskID string) (bool, error) {
	return true, nil
}

// GetAlertsMerged mapping of action to relative alert(s)
func GetAlertsMerged(taskID string) (map[any]any, error) {
	return nil, nil
}
