package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"

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
	if err = db.initAllocationSessions(); err != nil {
		return nil, err
	}
	return db, nil
}

// ValidateRPWorkspaceBindings checks if rp-workspace bindings pertain to valid resource pools.
// Bindings with resource pools that don't exist have the "valid" column set to false.
func ValidateRPWorkspaceBindings(ctx context.Context, pools []config.ResourcePoolConfig) error {
	var poolNames []string
	for _, pool := range pools {
		poolNames = append(poolNames, pool.PoolName)
	}

	tx, err := Bun().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error beginning transaction: %t", err)
	}
	defer func() {
		txErr := tx.Rollback()
		if txErr != nil && txErr != sql.ErrTxDone {
			log.WithError(txErr).Error("error rolling back transaction in AddExperiment")
		}
	}()

	_, err = tx.NewUpdate().Table("rp_workspace_bindings").
		Set("valid = false").
		Where("pool_name NOT IN (?)", bun.In(poolNames)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("error while invalidating rp-workspace bindings: %t", err)
	}

	_, err = tx.NewUpdate().Table("rp_workspace_bindings").
		Set("valid = true").
		Where("pool_name IN (?)", bun.In(poolNames)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("error while validating rp-workspace bindings: %t", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("error committing transaction: %t", err)
	}

	return nil
}
