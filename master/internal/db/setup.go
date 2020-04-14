package db

import (
	"fmt"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const maxOpenConns = 48

const cnxTpl = "postgres://%s:%s@%s:%s/%s?sslmode=disable&application_name=determined-master"

// Setup connects to the database and run any necessary migrations.
func Setup(opts *Config) (*PgDB, error) {
	dbURL := fmt.Sprintf(cnxTpl, opts.User, opts.Password, opts.Host, opts.Port, opts.Name)
	log.Infof("connecting to database %s:%s", opts.Host, opts.Port)
	db, err := ConnectPostgres(dbURL)
	if err != nil {
		return nil, errors.Wrapf(err, "connecting to database: %s", dbURL)
	}

	db.sql.SetMaxOpenConns(maxOpenConns)

	log.Infof("running migrations from %v", opts.Migrations)
	if err = db.Migrate(opts.Migrations); err != nil {
		return nil, errors.Wrap(err, "running migrations")
	}
	return db, nil
}
