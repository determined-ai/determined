package db

import (
	"context"
	"crypto/ed25519"
	"database/sql"
	"fmt"
	"regexp"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/pkg/model"
)

const maxOpenConns = 48

const (
	cnxTpl = "postgres://%s:%s@%s:%s/%s?application_name=determined-master"
	sslTpl = "&sslmode=%s&sslrootcert=%s"
)

// authTokenKeypair gets the existing auth token keypair.
func authTokenKeypair(ctx context.Context) (*model.AuthTokenKeypair, error) {
	var tokenKeypair model.AuthTokenKeypair
	switch err := Bun().NewSelect().Table("auth_token_keypair").Scan(ctx, &tokenKeypair); {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	case errors.Is(err, ErrNotFound):
		return nil, nil
	case err != nil:
		return nil, err
	default:
		return &tokenKeypair, nil
	}
}

// addAuthTokenKeypair adds the new auth token keypair.
func addAuthTokenKeypair(ctx context.Context, tokenKeypair *model.AuthTokenKeypair) error {
	_, err := Bun().NewInsert().
		Model(&model.AuthTokenKeypair{
			PublicKey:  tokenKeypair.PublicKey,
			PrivateKey: tokenKeypair.PrivateKey,
		}).
		Exec(ctx)
	return err
}

// InitAuthKeys initializes auth token keypairs.
func InitAuthKeys() error {
	switch storedKeys, err := authTokenKeypair(context.TODO()); {
	case err != nil:
		return fmt.Errorf("error retrieving auth token keypair: %s", err)
	case storedKeys == nil:
		publicKey, privateKey, err := ed25519.GenerateKey(nil)
		if err != nil {
			return fmt.Errorf("error creating auth token keypair: %s", err)
		}
		tokenKeypair := model.AuthTokenKeypair{PublicKey: publicKey, PrivateKey: privateKey}
		err = addAuthTokenKeypair(context.TODO(), &tokenKeypair)
		if err != nil {
			return fmt.Errorf("error saving auth token keypair: %s", err)
		}
		SetTokenKeys(&tokenKeypair)
	default:
		SetTokenKeys(storedKeys)
	}
	return nil
}

// checkPostgresVersion checks the version of the connected Postgres database,
// and logs a warning if the version is unsupported.
func checkPostgresVersion(db *PgDB) error {
	var dbVersion string
	err := Bun().NewSelect().ColumnExpr("version()").Scan(context.TODO(), &dbVersion)
	if err != nil {
		return err
	}
	versionRegex := regexp.MustCompile(`PostgreSQL (\d+)(?:\.\d+)?`)

	matches := versionRegex.FindStringSubmatch(dbVersion)
	if len(matches) < 2 {
		return fmt.Errorf("could not parse Postgres version: %s", dbVersion)
	}
	version := matches[1]
	// TODO (CM-443): Bump this to 12 once Postgres 12 has reached EOL.
	if version <= "11" {
		log.Errorf(
			"Postgres %s has reached it's end of life. Upgrading to a supported version is strongly recommended.",
			version,
		)
	} else if version == "12" {
		// TODO (CM-443): Remove above warning once Postgres 12 has reached EOL.
		log.Warnf(
			"Postgres %s will reach it's end of life on November 14, 2024. Please upgrade to a supported version.",
			version)
	}

	return nil
}

// Connect connects to the database, but doesn't run migrations & inits.
func Connect(opts *config.DBConfig) (*PgDB, error) {
	dbURL := fmt.Sprintf(cnxTpl, opts.User, opts.Password, opts.Host, opts.Port, opts.Name)
	dbURL += fmt.Sprintf(sslTpl, opts.SSLMode, opts.SSLRootCert)
	log.Infof("connecting to database %s:%s", opts.Host, opts.Port)
	db, err := ConnectPostgres(dbURL)
	if err != nil {
		return nil, fmt.Errorf("%s: error connecting to database: %s:%s", err, opts.Host, opts.Port)
	}

	db.sql.SetMaxOpenConns(maxOpenConns)

	err = checkPostgresVersion(db)
	if err != nil {
		log.Errorf("error checking Postgres version: %s", err)
	}

	return db, nil
}

// Setup connects to the database and run any necessary migrations.
func Setup(
	opts *config.DBConfig, postConnectHooks ...func(*PgDB) error,
) (db *PgDB, err error) {
	db, err = Connect(opts)
	if err != nil {
		return db, err
	}

	for _, hook := range postConnectHooks {
		if err := hook(db); err != nil {
			return nil, err
		}
	}

	err = db.Migrate(opts.Migrations, opts.ViewsAndTriggers, []string{"up"})
	if err != nil {
		return nil, fmt.Errorf("error running migrations: %s", err)
	}

	if err = InitAuthKeys(); err != nil {
		return nil, err
	}

	if err = initAllocationSessions(context.TODO()); err != nil {
		return nil, err
	}
	return db, nil
}
