package tasks

import (
	"github.com/determined-ai/determined/master/pkg/model"
)

// Maybe think of better name.
type GenericTaskSpec struct {
	Base TaskSpec

	// CollectionID int //
	// RunID        int //
	// TrialRunID int // restartsID, should be somewhere else?

	WorkspaceID int

	GenericTaskConfig model.GenericTaskConfig

	// Keys ssh.PrivateAndPublicKeys
}
