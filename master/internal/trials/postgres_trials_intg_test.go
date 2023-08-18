//go:build integration
// +build integration

package trials

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
)

func TestMarkLostTrials(t *testing.T) {
	ctx := context.Background()
	pgDB := db.MustResolveTestPostgres(t)

	user := db.RequireMockUser(t, pgDB)
	// TODO(ilia): it'd be useful to cleanup the user, but we can't because of a foreign key
	// constraint with jobs owner table.
	// t.Cleanup(func() {
	// 	db.Bun().NewDelete().Table("users").Where("id = ?", user.ID).Exec(ctx)
	// })

	type TestCase struct {
		ExpectedExperimentState model.State
		StartingTrialStates     []model.State
		ExpectedTrialStates     []model.State
	}

	cases := []TestCase{
		{
			ExpectedExperimentState: model.ErrorState,
			StartingTrialStates:     []model.State{model.RunningState, model.RunningState},
			ExpectedTrialStates:     []model.State{model.ErrorState, model.ErrorState},
		},
		{
			ExpectedExperimentState: model.CompletedState,
			StartingTrialStates: []model.State{
				model.CompletedState, model.RunningState, model.ErrorState,
			},
			ExpectedTrialStates: []model.State{model.CompletedState, model.ErrorState, model.ErrorState},
		},
	}

	experiments := []*model.Experiment{}
	for i := 0; i < len(cases); i++ {
		experiments = append(experiments, db.RequireMockExperiment(t, pgDB, user))
	}
	experimentIds := []int{}
	for _, e := range experiments {
		experimentIds = append(experimentIds, e.ID)
	}

	_, err := db.Bun().NewUpdate().Model((*model.Experiment)(nil)).
		Where("id IN (?)", bun.In(experimentIds)).
		Set("unmanaged = true").
		Exec(ctx)

	require.NoError(t, err)

	t.Cleanup(func() {
		_, err := db.Bun().NewDelete().Model((*model.Experiment)(nil)).
			Where("id IN (?)", bun.In(experimentIds)).
			Exec(ctx)

		require.NoError(t, err)
	})

	trials := map[int][]int{}

	for i, exp := range experiments {
		for j := 0; j < len(cases[i].StartingTrialStates); j++ {
			trial, _ := db.RequireMockTrial(t, pgDB, exp)

			trialID := trial.ID
			lastActivity := time.Now().Add(-20 * time.Minute)
			_, err := db.Bun().NewUpdate().Model((*Trial)(nil)).
				Where("id = ?", trialID).
				Set("state = ?", cases[i].StartingTrialStates[j]).
				Set("last_activity = ?", lastActivity).
				Exec(ctx)

			require.NoError(t, err)

			trials[exp.ID] = append(trials[exp.ID], trial.ID)
		}
	}

	t.Cleanup(func() {
		trialIds := []int{}

		for _, trs := range trials {
			for _, trialID := range trs {
				trialIds = append(trialIds, trialID)
			}
		}

		_, err := db.Bun().NewDelete().Model((*Trial)(nil)).
			Where("id in (?)", bun.In(trialIds)).
			Exec(ctx)

		require.NoError(t, err)
	})

	t.Run("MarkLostTrials", func(t *testing.T) {
		err := MarkLostTrials(ctx)

		require.NoError(t, err)

		experimentsRes := []model.Experiment{}
		err = db.Bun().NewSelect().Model(&experimentsRes).
			Where("id in (?)", bun.In(experimentIds)).
			Order("id").
			Column("id", "state").
			Scan(ctx)
		require.NoError(t, err)
		for i, e := range experimentsRes {
			require.Equal(t, cases[i].ExpectedExperimentState, e.State)

			trialsRes := []Trial{}
			err = db.Bun().NewSelect().Model(&trialsRes).
				Where("id in (?)", bun.In(trials[e.ID])).
				Order("id").
				Scan(ctx)
			require.NoError(t, err)

			for j, tr := range trialsRes {
				require.Equal(t, cases[i].ExpectedTrialStates[j], tr.State)
			}
		}
	})
}
