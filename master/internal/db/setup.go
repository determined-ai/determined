package db

import (
	"fmt"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const maxOpenConns = 48

const (
	cnxTpl         = "postgres://%s:%s@%s:%s/%s?application_name=determined-master"
	sslTpl         = "&sslmode=%s&sslrootcert=%s"
	sslModeDisable = "disable"
)

// Setup connects to the database and run any necessary migrations.
func Setup(opts *Config) (*PgDB, error) {
	dbURL := fmt.Sprintf(cnxTpl, opts.User, opts.Password, opts.Host, opts.Port, opts.Name)
	dbURL += fmt.Sprintf(sslTpl, opts.SSLMode, opts.SSLRootCert)
	log.Infof("connecting to database %s:%s", opts.Host, opts.Port)
	db, err := ConnectPostgres(dbURL)
	if err != nil {
		return nil, errors.Wrapf(err, "error connecting to database: %s:%s", opts.Host, opts.Port)
	}

	db.sql.SetMaxOpenConns(maxOpenConns)

	if err = db.Migrate(opts.Migrations); err != nil {
		return nil, errors.Wrap(err, "running migrations")
	}
	if err = db.initAuthKeys(); err != nil {
		return nil, err
	}
	if err = db.initAllocationSessions(); err != nil {
		return nil, err
	}
	return db, nil
}
