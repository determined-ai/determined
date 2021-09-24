package db

import (
	"github.com/golang-migrate/migrate"
	postgresM "github.com/golang-migrate/migrate/database/postgres"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Migrate runs the migrations from the specified directory URL.
func (db *PgDB) Migrate(migrationURL string) error {
	log.Infof("running DB migrations from %s; this might take a while...", migrationURL)
	driver, err := postgresM.WithInstance(db.sql.DB, &postgresM.Config{})
	if err != nil {
		return errors.Wrap(err, "error constructing Postgres migration driver")
	}
	m, err := migrate.NewWithDatabaseInstance(migrationURL, "postgres", driver)
	if err != nil {
		return errors.Wrapf(err, "error constructing Postgres migration using %s", migrationURL)
	}

	migrateVersion, _, merr := m.Version()
	if merr != nil {
		if merr != migrate.ErrNilVersion {
			return errors.Wrap(merr, "error loading golang-migrate version")
		}
		log.Info("unable to find golang-migrate version")
	} else {
		log.Infof("found golang-migrate version %v", migrateVersion)
	}

	if err = m.Up(); err != nil && err != migrate.ErrNoChange {
		return errors.Wrap(err, "error applying migrations")
	}
	log.Info("DB migrations completed")
	return nil
}
