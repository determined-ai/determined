package ft

import (
	"fmt"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/set"
)

/*
write through cache the alerts. or nvm just use the db or in memory for first milestone.
*/

// DisallowedNodes returns a list of nodes that should be blacklisted for the given allocation
func DisallowedNodes(taskID model.TaskID) *set.Set[string] {
	fmt.Println(taskID)

	s := set.New[string]()
	s.Insert("agent1")
	return &s

	// maybe just taskid is enough. go off of task id and GetAlerts
	return nil
}

/*
func CanTaskBeOnNode(taskID model.TaskID, agentID string) bool {
	fmt.Println("CAN TASK BE ON NODE", taskID, agentID)
	// TODO write through cache.
	return true
}
*/

func UserOwnsTask(userID, taskID string) (bool, error) {
	return true, nil
}

// GetAlertsMerged mapping of action to relative alert(s)
func GetAlertsMerged(taskID string) (map[any]any, error) {
	return nil, nil
}
