//go:build integration
// +build integration

package db

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"

	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
)

const (
	tableKey     = "cluster_messages"
	msgKey       = "message"
	startTimeKey = "start_time"
	endTimeKey   = "end_time"
	content      = "test msg"
	content2     = "test msg 2"
	content3     = "test msg 3"
)

func TestGetActiveClusterMessage(t *testing.T) {
	ctx := context.TODO()
	require.NoError(t, etc.SetRootPath(RootFromDB))

	db, closer := MustResolveTestPostgres(t)
	bunDB := bun.NewDB(db.sql.DB, pgdialect.New())

	defer func() {
		_, err := bunDB.NewTruncateTable().Table(tableKey).Exec(ctx)
		require.NoError(t, err)

		closer()
	}()

	// test no messages returns not found
	msg, err := GetActiveClusterMessage(ctx, bunDB)
	require.True(t, errors.Is(err, ErrNotFound))

	// test get active cluster message - infinite expiration
	// columns: message, created_by, start_time, end_time, created_time
	values := map[string]interface{}{"message": content}
	_, err = bunDB.NewInsert().Model(&values).Table(tableKey).Exec(ctx)
	require.NoError(t, err)

	msg, err = GetActiveClusterMessage(ctx, bunDB)
	require.NoError(t, err)
	require.Equal(t, content, msg.Message)

	// test get active cluster message - expiration in the future
	values[endTimeKey] = time.Now().Add(time.Hour)

	_, err = bunDB.NewUpdate().Model(&values).Table(tableKey).Where("message = ?", content).Exec(ctx)
	require.NoError(t, err)

	msg, err = GetActiveClusterMessage(ctx, bunDB)
	require.NoError(t, err)
	require.Equal(t, content, msg.Message)

	// test expired message returns not found
	values[endTimeKey] = time.Now().Add(time.Duration(-10) * time.Minute)

	_, err = bunDB.NewUpdate().Model(&values).Table(tableKey).Where("message = ?", content).Exec(ctx)
	require.NoError(t, err)

	msg, err = GetActiveClusterMessage(ctx, bunDB)
	require.True(t, errors.Is(err, ErrNotFound))

	// test future message returns not found
	values = map[string]interface{}{msgKey: content2, startTimeKey: time.Now().Add(time.Minute)}

	_, err = bunDB.NewInsert().Model(&values).Table(tableKey).Exec(ctx)
	require.NoError(t, err)

	msg, err = GetActiveClusterMessage(ctx, bunDB)
	require.True(t, errors.Is(err, ErrNotFound))
}

func TestGetClusterMessage(t *testing.T) {
	ctx := context.TODO()
	require.NoError(t, etc.SetRootPath(RootFromDB))

	db, closer := MustResolveTestPostgres(t)
	bunDB := bun.NewDB(db.sql.DB, pgdialect.New())
	defer func() {
		_, err := bunDB.NewTruncateTable().Table(tableKey).Exec(ctx)
		require.NoError(t, err)

		closer()
	}()

	// test no messages returns not found
	msg, err := GetClusterMessage(ctx, bunDB)
	require.True(t, errors.Is(err, ErrNotFound), "expected err not found, but got:", err)

	// test get cluster message - infinite expiration
	// fields: message, created_by, start_time, end_time, created_time
	values := map[string]interface{}{"message": content}
	_, err = bunDB.NewInsert().Model(&values).Table(tableKey).Exec(ctx)
	require.NoError(t, err)

	msg, err = GetClusterMessage(ctx, bunDB)
	require.NoError(t, err)
	require.Equal(t, content, msg.Message)

	// test expired cluster msg not found
	values = map[string]interface{}{
		"message":  content2,
		endTimeKey: time.Now().Add(time.Duration(-1) * time.Hour),
	}
	_, err = bunDB.NewInsert().Model(&values).Table(tableKey).Exec(ctx)
	require.NoError(t, err)

	msg, err = GetClusterMessage(ctx, bunDB)
	require.True(t, errors.Is(err, ErrNotFound), "expected err not found, but got:", err)

	// test get future cluster msg
	values = map[string]interface{}{
		"message":    content3,
		startTimeKey: time.Now().Add(time.Hour),
	}
	_, err = bunDB.NewInsert().Model(&values).Table(tableKey).Exec(ctx)
	require.NoError(t, err)

	msg, err = GetClusterMessage(ctx, bunDB)
	require.NoError(t, err)
	require.Equal(t, content3, msg.Message)
}

func TestSetClusterMessage(t *testing.T) {
	ctx := context.TODO()
	require.NoError(t, etc.SetRootPath(RootFromDB))

	db, closer := MustResolveTestPostgres(t)
	bunDB := bun.NewDB(db.sql.DB, pgdialect.New())
	defer func() {
		_, err := bunDB.NewTruncateTable().Table(tableKey).Exec(ctx)
		require.NoError(t, err)

		closer()
	}()

	// test get cluster message - infinite expiration
	content := "test msg"
	msg := model.ClusterMessage{
		Message: content,
	}

	err := SetClusterMessage(ctx, bunDB, msg)
	require.NoError(t, err)

	msg, err = GetClusterMessage(ctx, bunDB)
	require.NoError(t, err)
	require.Equal(t, content, msg.Message)

	// test cluster message > 250 runes
	// TODO (eliu): fix borked test
	//content = ""
	//for x := 0; x < 251; x++ {
	//	content += "a"
	//}
	//require.Equal(t, 251, utf8.RuneCountInString(msg.Message))
	//
	//msg = model.ClusterMessage{
	//	Message: content,
	//}
	//err = SetClusterMessage(ctx, bunDB, msg)
	//require.True(t, errors.Is(err, ErrInvalidInput))

	// test expiration time before start time

	// test expiration time before now
}
