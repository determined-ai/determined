package kubernetesrm

import (
	"log"
	"os"
	"testing"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/etc"
)

func TestMain(m *testing.M) {
	pgDB, err := db.ResolveTestPostgres()
	if err != nil {
		log.Panicln(err)
	}

	err = db.MigrateTestPostgres(pgDB, "file://../../../static/migrations", "up")
	if err != nil {
		log.Panicln(err)
	}

	err = etc.SetRootPath("../../../static/srv")
	if err != nil {
		log.Panicln(err)
	}

	os.Exit(m.Run())
}
