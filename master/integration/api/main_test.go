// +build integration

package api

import (
	"fmt"
	"os"
	"testing"

	"github.com/determined-ai/determined/master/integration/testutils"
	"github.com/determined-ai/determined/master/internal/db"
)

var pgDB *db.PgDB

func TestMain(m *testing.M) {
	var err error
	pgDB, err = testutils.ResolvePostgres()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}
