package webhooks

import (
	"context"
	"testing"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/stretchr/testify/require"
)

func TestShipper(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pgDB := db.MustResolveTestPostgres(t)
	db.MustMigrateTestPostgres(t, pgDB, "file://../static/migrations")
	require.NoError(t, etc.SetRootPath("../static/srv"))

	t.Run("")
}
