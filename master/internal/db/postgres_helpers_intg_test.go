//go:build integration
// +build integration

package db

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/commonv1"
	"github.com/determined-ai/determined/proto/pkg/modelv1"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

func pprintedExpect(expected, got interface{}) string {
	return fmt.Sprintf("expected \n\t%s\ngot\n\t%s", spew.Sdump(expected), spew.Sdump(got))
}

func requireMockModel(t *testing.T, db *PgDB, user model.User) *modelv1.Model {
	now := time.Now()
	m := model.Model{
		Name:            uuid.NewString(),
		Description:     "",
		Metadata:        map[string]interface{}{},
		CreationTime:    now,
		LastUpdatedTime: now,
		Labels:          []string{},
		Username:        user.Username,
	}
	b, err := json.Marshal(m.Metadata)
	require.NoError(t, err)

	var pmdl modelv1.Model
	err = db.QueryProto("insert_model", &pmdl, m.Name, m.Description, b, m.Labels, "", user.ID)
	require.NoError(t, err)
	return &pmdl
}

func requireMockMetrics(
	t *testing.T, db *PgDB, tr *model.Trial, stepsCompleted int, metricValue float64,
) *trialv1.TrialMetrics {
	m := trialv1.TrialMetrics{
		TrialId:        int32(tr.ID),
		StepsCompleted: int32(stepsCompleted),
		Metrics: &commonv1.Metrics{
			AvgMetrics: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					defaultSearcherMetric: {
						Kind: &structpb.Value_NumberValue{
							NumberValue: metricValue,
						},
					},
				},
			},
			BatchMetrics: []*structpb.Struct{},
		},
	}
	err := db.AddValidationMetrics(context.TODO(), &m)
	require.NoError(t, err)
	return &m
}
