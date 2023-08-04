package db

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/go-pg/migrations/v8"
	"github.com/go-pg/pg/v10"
	"github.com/jackc/pgconn"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func makeGoPgOpts(dbURL string) (*pg.Options, error) {
	// go-pg ParseURL doesn't support sslrootcert, so strip it and do manually.
	// TODO(DET-6084): make an upstream PR for this.
	re := regexp.MustCompile(`&sslrootcert=([^&]*)`)
	url := re.ReplaceAllString(dbURL, "")
	opts, err := pg.ParseURL(url)
	if err != nil {
		return nil, err
	}

	if opts.TLSConfig != nil {
		pgxConfig, err := pgconn.ParseConfig(dbURL)
		if err != nil {
			return nil, err
		}
		opts.TLSConfig = pgxConfig.TLSConfig
	}

	return opts, nil
}

func tablesExist(tx *pg.Tx, tableNames []string) (map[string]bool, error) {
	existingTables := []string{}
	result := map[string]bool{}
	for _, tn := range tableNames {
		result[tn] = false
	}

	_, err := tx.Query(
		&existingTables,
		`SELECT tablename FROM pg_tables WHERE schemaname = 'public' AND tablename IN (?)`,
		pg.In(tableNames),
	)
	if err != nil {
		return result, err
	}

	for _, tn := range existingTables {
		result[tn] = true
	}

	return result, nil
}

func ensureMigrationUpgrade(tx *pg.Tx) error {
	exist, err := tablesExist(tx, []string{"gopg_migrations", "schema_migrations"})
	if err != nil {
		return err
	}

	// On fresh installations, just run init.
	if !exist["gopg_migrations"] && !exist["schema_migrations"] {
		_, _, err = migrations.Run(tx, "init")
		return err
	}

	if exist["gopg_migrations"] || !exist["schema_migrations"] {
		return nil
	}

	log.Infof("upgrading migration metadata...")

	type GoMigrateEntry struct {
		Version string
		Dirty   bool
	}

	rows := []GoMigrateEntry{}
	if _, err = tx.Query(&rows, `SELECT version, dirty FROM schema_migrations`); err != nil {
		return err
	}

	// Unrecognized table state.
	if len(rows) != 1 {
		return fmt.Errorf("schema_migrations table has %d entries", len(rows))
	}

	goMigrateEntry := rows[0]

	if goMigrateEntry.Dirty {
		return fmt.Errorf("schema_migrations entry dirty, version %s", goMigrateEntry.Version)
	}
	goMigrateVersion, err := strconv.ParseInt(goMigrateEntry.Version, 10, 64)
	if err != nil {
		return err
	}

	// CREATE gopg_migrations table,
	// and INSERT the initial version from go-migrate.
	if _, _, err := migrations.Run(tx, "init"); err != nil {
		return err
	}
	if err := migrations.SetVersion(tx, goMigrateVersion); err != nil {
		return err
	}

	return nil
}

// Migrate runs the migrations from the specified directory URL.
func (db *PgDB) Migrate(migrationURL string, actions []string) error {
	// go-pg/migrations uses go-pg/pg connection API, which is not compatible
	// with pgx, so we use a one-off go-pg/pg connection.
	pgOpts, err := makeGoPgOpts(db.url)
	if err != nil {
		return err
	}

	pgConn := pg.Connect(pgOpts)
	defer func() {
		if errd := pgConn.Close(); errd != nil {
			log.Errorf("error closing pg connection: %s", errd)
		}
	}()

	tx, err := pgConn.Begin()
	if err != nil {
		return err
	}

	defer func() {
		// Rollback unless it has already been committed.
		if errd := tx.Close(); errd != nil {
			log.Errorf("failed to rollback pg transaction while migrating: %s", errd)
		}
	}()

	// In integration tests, multiple processes can be running this code at once, which can lead to
	// errors because PostgreSQL's CREATE TABLE IF NOT EXISTS is not great with concurrency.

	// Arbitrarily chosen unique consistent ID for the lock.
	const MigrationLockID = 0x33ad0708c9bed25b

	_, err = tx.Exec("SELECT pg_advisory_xact_lock(?)", MigrationLockID)
	if err != nil {
		return err
	}

	if err = ensureMigrationUpgrade(tx); err != nil {
		return errors.Wrap(err, "error upgrading migration metadata")
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	log.Infof("running DB migrations from %s; this might take a while...", migrationURL)

	re := regexp.MustCompile(`file://(.+)`)
	match := re.FindStringSubmatch(migrationURL)
	if len(match) != 2 {
		return fmt.Errorf("failed to parse migrationsURL: %s", migrationURL)
	}

	collection := migrations.NewCollection()
	collection.DisableSQLAutodiscover(true)
	if err = collection.DiscoverSQLMigrations(match[1]); err != nil {
		return err
	}
	if len(collection.Migrations()) == 0 {
		return errors.New("failed to discover any migrations")
	}

	oldVersion, newVersion, err := collection.Run(pgConn, actions...)
	if err != nil {
		return errors.Wrap(err, "error applying migrations")
	}

	if oldVersion == newVersion {
		log.Infof("no migrations to apply; version: %d", newVersion)
	} else {
		log.Infof("migrated from %d to %d", oldVersion, newVersion)
	}

	log.Info("DB migrations completed")
	return nil
}
