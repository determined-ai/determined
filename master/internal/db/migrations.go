package db

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/go-pg/migrations/v8"
	"github.com/go-pg/pg/v10"
	"github.com/jackc/pgconn"
	"github.com/jmoiron/sqlx"
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

func (db *PgDB) addDBCode(dbCodeDir string) error {
	files, err := os.ReadDir(dbCodeDir)
	if err != nil {
		return fmt.Errorf("reading '%s' directory for database views: %w", dbCodeDir, err)
	}

	if err := db.withTransaction("determined database views", func(tx *sqlx.Tx) error {
		for _, f := range files {
			if filepath.Ext(f.Name()) != ".sql" {
				continue
			}

			filePath := filepath.Join(dbCodeDir, f.Name())
			b, err := os.ReadFile(filePath) //nolint: gosec // We trust dbCodeDir.
			if err != nil {
				return fmt.Errorf("reading view definition file '%s': %w", filePath, err)
			}

			if _, err := tx.Exec(string(b)); err != nil {
				return fmt.Errorf("running database view file '%s': %w", filePath, err)
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf("adding determined database views: %w", err)
	}

	return nil
}

func (db *PgDB) dropDBCode() error {
	// SET search_path since the ALTER DATABASE ... SET SEARCH_PATH won't apply to this connection
	// since it was created after the migration ran.
	if _, err := db.sql.Exec(`
DROP SCHEMA IF EXISTS determined_code CASCADE;
CREATE SCHEMA determined_code;
SET search_path TO determined_code,public`); err != nil {
		return fmt.Errorf("removing determined database views so they can be created later: %w", err)
	}

	return nil
}

// Migrate runs the migrations from the specified directory URL.
func (db *PgDB) Migrate(
	migrationURL string, dbCodeDir string, actions []string,
) (isNew bool, err error) {
	if err := db.dropDBCode(); err != nil {
		return false, err
	}

	// go-pg/migrations uses go-pg/pg connection API, which is not compatible
	// with pgx, so we use a one-off go-pg/pg connection.
	pgOpts, err := makeGoPgOpts(db.URL)
	if err != nil {
		return false, err
	}

	pgConn := pg.Connect(pgOpts)
	defer func() {
		if errd := pgConn.Close(); errd != nil {
			log.Errorf("error closing pg connection: %s", errd)
		}
	}()

	tx, err := pgConn.Begin()
	if err != nil {
		return false, err
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
		return false, err
	}

	if err = ensureMigrationUpgrade(tx); err != nil {
		return false, errors.Wrap(err, "error upgrading migration metadata")
	}

	if err = tx.Commit(); err != nil {
		return false, err
	}

	log.Infof("running DB migrations from %s; this might take a while...", migrationURL)

	re := regexp.MustCompile(`file://(.+)`)
	match := re.FindStringSubmatch(migrationURL)
	if len(match) != 2 {
		return false, fmt.Errorf("failed to parse migrationsURL: %s", migrationURL)
	}

	collection := migrations.NewCollection()
	collection.DisableSQLAutodiscover(true)
	if err = collection.DiscoverSQLMigrations(match[1]); err != nil {
		return false, err
	}
	if len(collection.Migrations()) == 0 {
		return false, errors.New("failed to discover any migrations")
	}

	oldVersion, newVersion, err := collection.Run(pgConn, actions...)
	if err != nil {
		return false, errors.Wrap(err, "error applying migrations")
	}

	if oldVersion == newVersion {
		log.Infof("no migrations to apply; version: %d", newVersion)
	} else {
		log.Infof("migrated from %d to %d", oldVersion, newVersion)
	}

	if newVersion >= 20240502203516 { // Only comes up in testing old data.
		if err := db.addDBCode(dbCodeDir); err != nil {
			return false, err
		}
	}

	log.Info("DB migrations completed")
	return oldVersion == 0, nil
}
