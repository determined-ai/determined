//go:build integration
// +build integration

package db

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

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
	_, err := GetActiveClusterMessage(ctx, bunDB)
	require.True(t, errors.Is(err, ErrNotFound))

	// test get active cluster message - infinite expiration
	// columns: message, created_by, start_time, end_time, created_time
	values := map[string]interface{}{"message": content}
	_, err = bunDB.NewInsert().Model(&values).Table(tableKey).Exec(ctx)
	require.NoError(t, err)

	msg, err := GetActiveClusterMessage(ctx, bunDB)
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
	_, err := GetClusterMessage(ctx, bunDB)
	require.True(t, errors.Is(err, ErrNotFound), "expected err not found, but got:", err)

	// test get cluster message - infinite expiration
	// fields: message, created_by, start_time, end_time, created_time
	values := map[string]interface{}{"message": content}
	_, err = bunDB.NewInsert().Model(&values).Table(tableKey).Exec(ctx)
	require.NoError(t, err)

	msg, err := GetClusterMessage(ctx, bunDB)
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

	// test set cluster message - infinite expiration
	msg := model.ClusterMessage{
		Message:   content,
		CreatedBy: 1,
	}

	err := SetClusterMessage(ctx, bunDB, msg)
	require.NoError(t, err)

	msg, err = GetClusterMessage(ctx, bunDB)
	require.NoError(t, err)
	require.Equal(t, content, msg.Message)

	// test cluster message > 250 runes
	runeContent := ""
	for x := 0; x < 251; x++ {
		runeContent += "a"
	}
	require.Equal(t, 251, utf8.RuneCountInString(runeContent))

	msg = model.ClusterMessage{
		Message: runeContent,
	}
	err = SetClusterMessage(ctx, bunDB, msg)
	require.True(t, errors.Is(err, ErrInvalidInput))

	// test expiration time before start time
	msg = model.ClusterMessage{
		Message:   content,
		CreatedBy: 1,
		EndTime: sql.NullTime{
			Time:  time.Now().Add(time.Duration(1) * time.Minute),
			Valid: true,
		},
		StartTime: time.Now().Add(time.Duration(1) * time.Hour),
	}

	err = SetClusterMessage(ctx, bunDB, msg)
	require.True(t, errors.Is(err, ErrInvalidInput))

	// test expiration time before now
	msg = model.ClusterMessage{
		Message:   content,
		CreatedBy: 1,
		EndTime: sql.NullTime{
			Time:  time.Now().Add(time.Duration(-1) * time.Hour),
			Valid: true,
		},
	}

	err = SetClusterMessage(ctx, bunDB, msg)
	require.True(t, errors.Is(err, ErrInvalidInput))
}

func TestSetClusterMessageLength(t *testing.T) {
	ctx := context.TODO()
	require.NoError(t, etc.SetRootPath(RootFromDB))

	db, closer := MustResolveTestPostgres(t)
	bunDB := bun.NewDB(db.sql.DB, pgdialect.New())
	defer func() {
		_, err := bunDB.NewTruncateTable().Table(tableKey).Exec(ctx)
		require.NoError(t, err)

		closer()
	}()

	testCases := []struct {
		name        string
		msg         string
		expectedErr error
	}{{
		name:        "Message is accepted when short enough and single-byte",
		msg:         "Test message 123!",
		expectedErr: nil,
	}, {
		name:        "Message is rejected when a letter-only message is too long",
		msg:         strings.Repeat("a", 251),
		expectedErr: ErrInvalidInput,
	}, {
		name:        "Message containing multi-byte characters is rejected when too long",
		msg:         strings.Repeat("😱👍", 126),
		expectedErr: ErrInvalidInput,
	}, {
		name:        "Message containing multi-byte characters is accepted when not too long",
		msg:         strings.Repeat("😱👍", 120),
		expectedErr: nil,
	}}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			actualErr := SetClusterMessage(ctx, bunDB, model.ClusterMessage{
				CreatedBy: 1, // admin
				Message:   c.msg,
			})
			require.ErrorIs(t, actualErr, c.expectedErr, "SetClusterMessage behaved unexpectedly based on message length")
		})
	}

	// Clean up after ourselves
	cleanupErr := ClearClusterMessage(ctx, bunDB)
	require.NoError(t, cleanupErr, "expected no error during cleanup")
}

func TestClearClusterMessage(t *testing.T) {
	ctx := context.TODO()
	require.NoError(t, etc.SetRootPath(RootFromDB))

	db, closer := MustResolveTestPostgres(t)
	bunDB := bun.NewDB(db.sql.DB, pgdialect.New())
	defer func() {
		_, err := bunDB.NewTruncateTable().Table(tableKey).Exec(ctx)
		require.NoError(t, err)

		closer()
	}()

	msg := model.ClusterMessage{
		Message:   content,
		CreatedBy: 1,
	}

	err := SetClusterMessage(ctx, bunDB, msg)
	require.NoError(t, err)

	msg, err = GetClusterMessage(ctx, bunDB)
	require.NoError(t, err)
	require.Equal(t, content, msg.Message)

	err = ClearClusterMessage(ctx, bunDB)
	require.NoError(t, err)

	msg, err = GetClusterMessage(ctx, bunDB)
	require.True(t, errors.Is(err, ErrNotFound))

	msg, err = GetActiveClusterMessage(ctx, bunDB)
	require.True(t, errors.Is(err, ErrNotFound))
}
