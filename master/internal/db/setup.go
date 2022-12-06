package db

import (
	"fmt"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/config"
)

const maxOpenConns = 48

const (
	cnxTpl = "postgres://%s:%s@%s:%s/%s?application_name=determined-master"
	sslTpl = "&sslmode=%s&sslrootcert=%s"
)

// Connect connects to the database, but doesn't run migrations & inits.
func Connect(opts *config.DBConfig) (*PgDB, error) {
	dbURL := fmt.Sprintf(cnxTpl, opts.User, opts.Password, opts.Host, opts.Port, opts.Name)
	dbURL += fmt.Sprintf(sslTpl, opts.SSLMode, opts.SSLRootCert)
	log.Infof("connecting to database %s:%s", opts.Host, opts.Port)
	db, err := ConnectPostgres(dbURL)
	if err != nil {
		return nil, errors.Wrapf(err, "error connecting to database: %s:%s", opts.Host, opts.Port)
	}

	db.sql.SetMaxOpenConns(maxOpenConns)

	return db, nil
}

// Setup connects to the database and run any necessary migrations.
func Setup(opts *config.DBConfig) (*PgDB, error) {
	db, err := Connect(opts)
	if err != nil {
		return db, err
	}

	if err = db.Migrate(opts.Migrations, []string{"up"}); err != nil {
		return nil, errors.Wrap(err, "running migrations")
	}
	if err = db.initAuthKeys(); err != nil {
		return nil, err
	}
	return db, nil
}
