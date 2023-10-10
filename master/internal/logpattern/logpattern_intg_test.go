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

	require.NoError(t, Initialize(ctx))

	require.Len(t, *DisallowedNodes(model.TaskID("fake task ID")), 0)

	user := db.RequireMockUser(t, pgDB)
	exp := db.RequireMockExperiment(t, pgDB, user)
	trial, task := db.RequireMockTrial(t, pgDB, exp)

	require.Len(t, *DisallowedNodes(task.TaskID), 0)

	require.NoError(t, AddRetryOnDifferentNode(ctx, task.TaskID, "n0", "regexa", "loga"))
	require.NoError(t, AddRetryOnDifferentNode(ctx, task.TaskID, "n1", "regexa", "logb"))
	require.NoError(t, AddRetryOnDifferentNode(ctx, task.TaskID, "n0", "regexb", "logc"))

	require.NoError(t, AddRetryOnDifferentNode(ctx, task.TaskID, "n0", "regexa", "dontappear"))
	require.NoError(t, AddRetryOnDifferentNode(ctx, task.TaskID, "n0", "regexb", "dontappear"))

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
			TaskEnded:     false,
		},
		{
			NodeName:      "n1",
			Regex:         "regexa",
			TriggeringLog: "logb",
			TaskEnded:     false,
		},
		{
			NodeName:      "n0",
			Regex:         "regexb",
			TriggeringLog: "logc",
			TaskEnded:     false,
		},
	}, actual)

	// Safe across "restarts".
	require.ElementsMatch(t, []string{"n0", "n1"},
		DisallowedNodes(task.TaskID).ToSlice())

	require.NoError(t, Initialize(ctx))

	require.ElementsMatch(t, []string{"n0", "n1"},
		DisallowedNodes(task.TaskID).ToSlice())

	// Restarts won't load inactive alerts for perf.
	_, err := db.Bun().NewUpdate().Model(&retryOnDifferentNode{}).
		Set("task_ended = true").
		Where("task_id = ?", task.TaskID).
		Where("node_name = ?", "n0").
		Exec(ctx)
	require.NoError(t, err)

	require.NoError(t, Initialize(ctx))

	require.ElementsMatch(t, []string{"n1"},
		DisallowedNodes(task.TaskID).ToSlice())

	// Reporting task done won't update db state just the cache.
	ReportTaskDone(task.TaskID)
	require.Len(t, DisallowedNodes(task.TaskID).ToSlice(), 0)

	require.NoError(t, Initialize(ctx))

	require.ElementsMatch(t, []string{"n1"},
		DisallowedNodes(task.TaskID).ToSlice())

	// Moving trial to done will end it's task and have it no longer be in cache when restarted.
	require.NoError(t, pgDB.UpdateTrial(trial.ID, model.CompletedState))

	require.NoError(t, Initialize(ctx))

	require.Len(t, DisallowedNodes(task.TaskID).ToSlice(), 0)
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

	require.NoError(t, AddDontRetry(ctx, task.TaskID, "n0", "regexa", "loga"))
	require.NoError(t, AddDontRetry(ctx, task.TaskID, "n0", "regexb", "logb"))
	require.NoError(t, AddDontRetry(ctx, task.TaskID, "n0", "regexa", "dontappear"))
	require.NoError(t, AddDontRetry(ctx, task.TaskID, "n1", "regexb", "dontappear"))

	resp, err = ShouldRetry(ctx, task.TaskID)
	require.NoError(t, err)
	require.ElementsMatch(t, []RetryInfo{
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

func TestSendWebhook(t *testing.T) {
	ctx := context.Background()

	resp, err := ShouldRetry(ctx, model.TaskID("fake Task ID"))
	require.NoError(t, err)
	require.Len(t, resp, 0)

	user := db.RequireMockUser(t, pgDB)
	exp := db.RequireMockExperiment(t, pgDB, user)
	_, task := db.RequireMockTrial(t, pgDB, exp)

	require.NoError(t, AddWebhookAlert(ctx, task.TaskID, "hook0", "n0", "regexa", "loga"))
	require.NoError(t, AddWebhookAlert(ctx, task.TaskID, "hook0", "n0", "regexb", "logb"))
	require.NoError(t, AddWebhookAlert(ctx, task.TaskID, "hook1", "n0", "regexa", "logc"))

	require.NoError(t, AddWebhookAlert(ctx, task.TaskID, "hook0", "n1", "regexa", "dontappear"))
	require.NoError(t, AddWebhookAlert(ctx, task.TaskID, "hook0", "n1", "regexb", "dontappear"))
	require.NoError(t, AddWebhookAlert(ctx, task.TaskID, "hook1", "n1", "regexa", "dontappear"))

	require.NoError(t, AddWebhookAlert(ctx, task.TaskID, "hook0", "n0", "regexa", "dontappear"))
	require.NoError(t, AddWebhookAlert(ctx, task.TaskID, "hook0", "n0", "regexb", "dontappear"))
	require.NoError(t, AddWebhookAlert(ctx, task.TaskID, "hook1", "n0", "regexa", "dontappear"))

	var actual []*sendWebhook
	require.NoError(t, db.Bun().NewSelect().Model(&actual).
		ExcludeColumn("id", "task_id").
		Where("task_id = ?", task.TaskID).
		Scan(ctx, &actual))

	require.ElementsMatch(t, []*sendWebhook{
		{
			WebhookName:   "hook0",
			NodeName:      "n0",
			Regex:         "regexa",
			TriggeringLog: "loga",
		},
		{
			WebhookName:   "hook0",
			NodeName:      "n0",
			Regex:         "regexb",
			TriggeringLog: "logb",
		},
		{
			WebhookName:   "hook1",
			NodeName:      "n0",
			Regex:         "regexa",
			TriggeringLog: "logc",
		},
	}, actual)
}
