package db

import (
	"crypto/sha256"
	"encoding/hex"
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

func tablesExist(tx pg.DBI, tableNames []string) (map[string]bool, error) {
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

func (db *PgDB) readDBCodeAndCheckIfDifferent(
	dbCodeDir string,
) (dbCodeFiles map[string]string, hash string, needToUpdateDBCode bool, err error) {
	upDir := filepath.Join(dbCodeDir, "up")
	files, err := os.ReadDir(upDir)
	if err != nil {
		return nil, "", false, fmt.Errorf("reading '%s' directory for database views: %w", dbCodeDir, err)
	}

	allCode := ""
	fileNamesToSQL := make(map[string]string)
	for _, f := range files {
		if filepath.Ext(f.Name()) != ".sql" {
			continue
		}

		filePath := filepath.Join(upDir, f.Name())
		b, err := os.ReadFile(filePath) //nolint: gosec // We trust dbCodeDir.
		if err != nil {
			return nil, "", false, fmt.Errorf("reading view definition file '%s': %w", filePath, err)
		}

		fileNamesToSQL[f.Name()] = string(b)
		allCode += string(b)
	}

	// I didn't want to get into deciding when to apply database or code or not but integration
	// tests make it really hard to not do this.
	hashSHA := sha256.Sum256([]byte(allCode))
	ourHash := hex.EncodeToString(hashSHA[:])

	// Check if the views_and_triggers_hash table exists. If it doesn't return that we need to create db code.
	var tableExists bool
	if err = db.sql.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'views_and_triggers_hash')").
		Scan(&tableExists); err != nil {
		return nil, "", false, fmt.Errorf("checking views_and_triggers_hash exists: %w", err)
	}
	if !tableExists {
		return fileNamesToSQL, ourHash, true, nil
	}

	// Check if our hashes match. If they do we can just return we don't need to do anything.
	var databaseHash string
	if err := db.sql.QueryRow("SELECT hash FROM views_and_triggers_hash").Scan(&databaseHash); err != nil {
		return nil, "", false, fmt.Errorf("getting hash from views_and_triggers_hash: %w", err)
	}
	if databaseHash == ourHash {
		return fileNamesToSQL, ourHash, false, nil
	}

	// Update our hash and return we need to create views and triggers.
	if err := db.dropDBCode(dbCodeDir); err != nil {
		return nil, "", false, err
	}

	return fileNamesToSQL, ourHash, true, nil
}

func (db *PgDB) addDBCode(fileNamesToSQL map[string]string, hash string) error {
	if err := db.withTransaction("determined database views", func(tx *sqlx.Tx) error {
		for filePath, sql := range fileNamesToSQL {
			if _, err := tx.Exec(sql); err != nil {
				return fmt.Errorf("running database view file '%s': %w", filePath, err)
			}
		}

		if _, err := tx.Exec("UPDATE views_and_triggers_hash SET hash = $1", hash); err != nil {
			return fmt.Errorf("updating our database hash: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("adding determined database views: %w", err)
	}

	return nil
}

func (db *PgDB) dropDBCode(dbCodeDir string) error {
	b, err := os.ReadFile(filepath.Join(dbCodeDir, "down.sql")) //nolint: gosec // We trust dbCodeDir.
	if err != nil {
		return fmt.Errorf("reading down db code migration: %w", err)
	}

	if _, err := db.sql.Exec(string(b)); err != nil {
		return fmt.Errorf("removing determined database views so they can be created later: %w", err)
	}

	return nil
}

// This is set in an init in postgres_test_utils.go behind the intg feature flag.
// For normal usages this won't build. For tests we need to serialize access to
// run migrations.
var testOnlyDBLock func(sql *sqlx.DB) (unlock func())

// Migrate runs the migrations from the specified directory URL.
func (db *PgDB) Migrate(
	migrationURL string, dbCodeDir string, actions []string,
) error {
	if testOnlyDBLock != nil {
		// In integration tests, multiple processes can be running this code at once, which can lead to
		// errors because PostgreSQL's CREATE TABLE IF NOT EXISTS is not great with concurrency.
		cleanup := testOnlyDBLock(db.sql)
		defer cleanup()
	}

	dbCodeFiles, hash, needToUpdateDBCode, err := db.readDBCodeAndCheckIfDifferent(dbCodeDir)
	if err != nil {
		return err
	}

	// go-pg/migrations uses go-pg/pg connection API, which is not compatible
	// with pgx, so we use a one-off go-pg/pg connection.
	pgOpts, err := makeGoPgOpts(db.URL)
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

	if newVersion >= 20240502203516 { // Only comes up in testing old data.
		if needToUpdateDBCode {
			log.Info("database views changed")
			if err := db.addDBCode(dbCodeFiles, hash); err != nil {
				return err
			}
		} else {
			log.Info("database views unchanged, will not updated")
		}
	}

	log.Info("DB migrations completed")
	return nil
}
