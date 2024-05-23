package db

import (
	"context"
	"crypto/ed25519"
	"database/sql"
	"fmt"

	"github.com/go-pg/migrations/v8"
	"github.com/go-pg/pg/v10"
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

	return db, nil
}

// IsNew checks to see if the database's migration tracking tables have been
// created, and if so, if it's above version 0. It returns `false` if the current
// version exists and is higher than zero, `true` otherwise.
// This is not guaranteed to be accurate if the database is otherwise in a bad or
// incomplete state. If an error is returned, the bool should be ignored.
func IsNew(opts *config.DBConfig) (bool, error) {
	dbURL := fmt.Sprintf(cnxTpl, opts.User, opts.Password, opts.Host, opts.Port, opts.Name)
	dbURL += fmt.Sprintf(sslTpl, opts.SSLMode, opts.SSLRootCert)
	pgOpts, err := makeGoPgOpts(dbURL)
	if err != nil {
		return false, err
	}

	pgConn := pg.Connect(pgOpts)
	defer func() {
		if errd := pgConn.Close(); errd != nil {
			log.Errorf("error closing pg connection: %s", errd)
		}
	}()

	exist, err := tablesExist(pgConn, []string{"gopg_migrations", "schema_migrations"})
	if err != nil {
		return false, err
	}
	if !exist["gopg_migrations"] {
		return true, nil
	}

	collection := migrations.NewCollection()
	collection.DisableSQLAutodiscover(true)
	version, err := collection.Version(pgConn)
	if err != nil {
		return false, err
	}
	return version == 0, nil
}

// Setup connects to the database and run any necessary migrations.
func Setup(opts *config.DBConfig) (db *PgDB, err error) {
	db, err = Connect(opts)
	if err != nil {
		return db, err
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
