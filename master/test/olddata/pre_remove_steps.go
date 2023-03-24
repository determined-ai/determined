//go:build integration
// +build integration

package olddata

import (
	// embed is only used in comments.
	_ "embed"
	"testing"

	"github.com/determined-ai/determined/master/internal/db"
)

//go:embed pre_remove_steps.sql
var preRemoveStepsSQL string

// migration001213 is the migration number as of v0.12.13.
const migration001213 = 20200630141158

// PreRemoveStepsExperimentsData holds the migration and useful constants for pre_remove_steps.sql.
type PreRemoveStepsExperimentsData struct {
	MustMigrate                  MustMigrateFn
	CompletedSingleExpID         int32
	CompletedAdaptiveSimpleExpID int32
	CompletedPBTExpID            int32
	PausedPBTExpID               int32
	PausedSingleExpID            int32
}

// PreRemoveStepsExperiments returns a PreRemoveStepsExperimentsData.
func PreRemoveStepsExperiments() PreRemoveStepsExperimentsData {
	mustMigrate := func(t *testing.T, pgdb *db.PgDB, migrationsPath string) {
		extra := migrationExtra{
			When: migration001213,
			SQL:  preRemoveStepsSQL,
		}
		mustMigrateWithExtras(t, pgdb, migrationsPath, extra)
	}
	return PreRemoveStepsExperimentsData{
		MustMigrate:                  mustMigrate,
		CompletedSingleExpID:         1,
		CompletedAdaptiveSimpleExpID: 2,
		CompletedPBTExpID:            3,
		PausedPBTExpID:               4,
		PausedSingleExpID:            5,
	}
}
