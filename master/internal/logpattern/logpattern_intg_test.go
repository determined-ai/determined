//go:build integration

package logpattern

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
)

var pgDB *db.PgDB

func TestMain(m *testing.M) {
	var err error
	pgDB, err = db.ResolveTestPostgres()
	if err != nil {
		log.Panicln(err)
	}

	err = db.MigrateTestPostgres(pgDB, "file://../../static/migrations", "up")
	if err != nil {
		log.Panicln(err)
	}

	err = etc.SetRootPath("../../static/srv")
	if err != nil {
		log.Panicln(err)
	}

	os.Exit(m.Run())
}

func TestRetryOnDifferentNode(t *testing.T) {
	ctx := context.Background()

	blocked, err := GetBlockedNodes(ctx, model.TaskID("fake task ID"))
	require.NoError(t, err)
	require.Len(t, blocked, 0)

	user := db.RequireMockUser(t, pgDB)
	exp := db.RequireMockExperiment(t, pgDB, user)
	_, task := db.RequireMockTrial(t, pgDB, exp)

	blocked, err = GetBlockedNodes(ctx, task.TaskID)
	require.NoError(t, err)
	require.Len(t, blocked, 0)

	require.NoError(t, addRetryOnDifferentNode(ctx, task.TaskID, "n0", "regexa", "loga"))
	require.NoError(t, addRetryOnDifferentNode(ctx, task.TaskID, "n1", "regexa", "logb"))
	require.NoError(t, addRetryOnDifferentNode(ctx, task.TaskID, "n0", "regexb", "logc"))

	require.NoError(t, addRetryOnDifferentNode(ctx, task.TaskID, "n0", "regexa", "dontappear"))
	require.NoError(t, addRetryOnDifferentNode(ctx, task.TaskID, "n0", "regexb", "dontappear"))

	// Check DB state is as expected.
	var actual []*retryOnDifferentNode
	require.NoError(t, db.Bun().NewSelect().Model(&actual).
		ExcludeColumn("id", "task_id").
		Where("task_id = ?", task.TaskID).
		Scan(ctx, &actual))

	require.ElementsMatch(t, []*retryOnDifferentNode{
		{
			NodeName:      "n0",
			Regex:         "regexa",
			TriggeringLog: "loga",
		},
		{
			NodeName:      "n1",
			Regex:         "regexa",
			TriggeringLog: "logb",
		},
		{
			NodeName:      "n0",
			Regex:         "regexb",
			TriggeringLog: "logc",
		},
	}, actual)

	blocked, err = GetBlockedNodes(ctx, task.TaskID)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"n0", "n1"}, blocked)
}

func TestShouldRetry(t *testing.T) {
	ctx := context.Background()

	resp, err := ShouldRetry(ctx, model.TaskID("fake task ID"))
	require.NoError(t, err)
	require.Len(t, resp, 0)

	user := db.RequireMockUser(t, pgDB)
	exp := db.RequireMockExperiment(t, pgDB, user)
	_, task := db.RequireMockTrial(t, pgDB, exp)

	resp, err = ShouldRetry(ctx, task.TaskID)
	require.NoError(t, err)
	require.Len(t, resp, 0)

	require.NoError(t, addDontRetry(ctx, task.TaskID, "n0", "regexa", "loga"))
	require.NoError(t, addDontRetry(ctx, task.TaskID, "n0", "regexb", "logb"))
	require.NoError(t, addDontRetry(ctx, task.TaskID, "n0", "regexa", "dontappear"))
	require.NoError(t, addDontRetry(ctx, task.TaskID, "n1", "regexb", "dontappear"))

	resp, err = ShouldRetry(ctx, task.TaskID)
	require.NoError(t, err)
	require.ElementsMatch(t, []DontRetryTrigger{
		{
			Regex:         "regexa",
			TriggeringLog: "loga",
		},
		{
			Regex:         "regexb",
			TriggeringLog: "logb",
		},
	}, resp)
}

func TestTaskLogsFromDontRetryTriggers(t *testing.T) {
	logs := TaskLogsFromDontRetryTriggers("id", []DontRetryTrigger{
		{
			Regex:         "regexa",
			TriggeringLog: "loga",
		},
		{
			Regex:         "regexb",
			TriggeringLog: "logb",
		},
	})
	totalLog := ""
	for _, l := range logs {
		require.Equal(t, "id", l.TaskID)
		totalLog += l.Log
	}

	require.Equal(t, `trial failed and matched logs to a don't retry policy
(log "loga" matched regex "regexa")
(log "logb" matched regex "regexb")
`, totalLog)
}
