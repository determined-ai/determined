package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/pkg/model"
)

// GetActiveClusterMessage returns the active cluster message if one is set and active, or
// ErrNotFound if not.
// Things to test:
// - When active, returns it
// - When not yet active, returns ErrNotFound
// - When expired, returns ErrNotFound
func GetActiveClusterMessage(ctx context.Context, db *bun.DB) (model.ClusterMessage, error) {
	var msg model.ClusterMessage
	err := db.NewRaw(`
		WITH newest_message AS (
			SELECT message, start_time, end_time, created_time
			FROM cluster_messages
			ORDER BY created_time DESC
			LIMIT 1
		)

		SELECT
			message, start_time,
			end_time, created_time
		FROM newest_message
		WHERE
			start_time < NOW()
			AND (end_time IS NULL OR end_time > NOW())
	`).Scan(ctx, &msg)
	if errors.Is(err, sql.ErrNoRows) {
		return model.ClusterMessage{}, ErrNotFound
	} else if err != nil {
		return model.ClusterMessage{}, err
	}

	return msg, nil
}

// GetClusterMessage returns the cluster message even if it's not yet active, or ErrNotFound if all
// cluster messages have expired.
func GetClusterMessage(ctx context.Context, db *bun.DB) (model.ClusterMessage, error) {
	var msg model.ClusterMessage
	err := db.NewRaw(`
		WITH newest_message AS (
			SELECT message, start_time, end_time, created_time
			FROM cluster_messages
			ORDER BY created_time DESC
			LIMIT 1
		)

		SELECT
			message, start_time,
			end_time, created_time
		FROM newest_message
		WHERE (end_time IS NULL OR end_time > NOW())
	`).Scan(ctx, &msg)
	if errors.Is(err, sql.ErrNoRows) {
		return model.ClusterMessage{}, ErrNotFound
	} else if err != nil {
		return model.ClusterMessage{}, err
	}

	return msg, nil
}

// ClusterMessageMaxLength caps the length of a cluster-wide message.
const ClusterMessageMaxLength = 250

// SetClusterMessage sets the cluster-wide message. Any existing message will be expired because
// only one cluster message is allowed at any time. Messages may be at most ClusterMessageMaxLength
// characters long. Returns a wrapped ErrInvalidInput when input is invalid.
// Stuff to test:
// - Max length of ClusterMessageMaxLength
func SetClusterMessage(ctx context.Context, db *bun.DB, msg model.ClusterMessage) error {
	if msgLen := utf8.RuneCountInString(msg.Message); msgLen > ClusterMessageMaxLength {
		return fmt.Errorf("%w: message must be at most %d characters; got %d",
			ErrInvalidInput, ClusterMessageMaxLength, msgLen)
	}

	if msg.EndTime.Time.Before(msg.StartTime) {
		return fmt.Errorf("%w, %s", ErrInvalidInput, "end time must be after start time")
	}

	if msg.EndTime.Time.Before(time.Now()) {
		return fmt.Errorf("%w, %s", ErrInvalidInput, "end time must be after current time")
	}

	return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewUpdate().
			Table("cluster_messages").
			Set("end_time = NOW()").
			Where("end_time >= NOW() OR end_time IS NULL").
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("%w: %s", err, "error clearing previous cluster-wide messages")
		}

		_, err = tx.NewInsert().
			Model(&msg).
			ExcludeColumn("created_time").
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("%w: %s", err, "error setting the cluster-wide message")
		}
		return nil
	})
}

// ClearClusterMessage clears the active cluster message.
func ClearClusterMessage(ctx context.Context, db *bun.DB) error {
	_, err := db.NewUpdate().
		Table("cluster_messages").
		Set("end_time = NOW()").
		Where("end_time >= NOW() OR end_time IS NULL").
		Exec(ctx)
	return err
}
