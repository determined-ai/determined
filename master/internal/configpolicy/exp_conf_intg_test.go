package configpolicy

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/stretchr/testify/require"
)

// Use a test to find out if we need a refactor of the psudo-interfacce, or not.
func TestExpConfUnmarshal(t *testing.T) {

	// 0. set up db
	ctx := context.Background()
	require.NoError(t, etc.SetRootPath(db.RootFromDB))
	pgDB, cleanup := db.MustResolveNewPostgresDatabase(t)
	defer cleanup()
	db.MustMigrateTestPostgres(t, pgDB, db.MigrationsFromDB)

	user := db.RequireMockUser(t, pgDB)

	// 1. add test data to db
	tcps := model.TaskConfigPolicies{
		WorkspaceID:     nil,
		WorkloadType:    model.ExperimentType,
		LastUpdatedBy:   user.ID,
		LastUpdatedTime: time.Now().UTC().Truncate(time.Second),
		InvariantConfig: DefaultInvariantConfigImage(),
		Constraints:     DefaultConstraints(),
	}

	err := SetTaskConfigPolicies(ctx, &tcps)
	require.NoError(t, err)

	// 2. read back test exp conf
	res, err := GetTaskConfigPolicies(ctx, nil, model.ExperimentType)
	require.NoError(t, err)

	// 3. create user exp conf
	userExpConf := expconf.ExperimentConfig{}
	userExpConf = schemas.WithDefaults(userExpConf)
	require.Equal(t, "determinedai/environments:rocm-5.6-pytorch-1.3-tf-2.10-rocm-mpich-0736b6d", *userExpConf.Environment().Image().RawROCM)

	// 4. apply invariant config
	err = json.Unmarshal([]byte(*res.InvariantConfig), &userExpConf)
	require.NoError(t, err)

	// 5. does it work?
	require.Equal(t, "bogus", *userExpConf.Environment().Image().RawROCM)

}
