package db

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v4/stdlib" // Import Postgres driver.
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/extra/bundebug"

	"github.com/determined-ai/determined/master/pkg/model"
)

var (
	bunMutex         sync.Mutex
	theOneBun        *bun.DB
	theOneDB         *PgDB
	modelsToRegister []interface{} // TODO (eliu): currently allows duplicate models
	tokenKeys        *model.AuthTokenKeypair
)

const id = "id"

// RegisterModel registers a model in Bun or, if theOneBun is not yet initialized,
// sets it up to be registered once initialized. It's generally best to pass a nil
// pointer of your model's type as argument m.
func RegisterModel(m interface{}) {
	bunMutex.Lock()
	defer bunMutex.Unlock()

	if theOneBun != nil {
		theOneBun.RegisterModel(m)
	}
	modelsToRegister = append(modelsToRegister, m)
}

func initTheOneBun(db *PgDB) {
	bunMutex.Lock()
	defer bunMutex.Unlock()
	if theOneBun != nil {
		log.Warn(
			"detected re-initialization of Bun that should never occur outside of tests",
		)
	}
	theOneBun = bun.NewDB(db.sql.DB, pgdialect.New())
	theOneDB = db

	for _, m := range modelsToRegister {
		theOneBun.RegisterModel(m)
	}

	// This will print every query that runs.
	// theOneBun.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))

	// This will print only the failed queries.
	theOneBun.AddQueryHook(bundebug.NewQueryHook())
}

func setTokenKeys(tk *model.AuthTokenKeypair) {
	bunMutex.Lock()
	defer bunMutex.Unlock()

	tokenKeys = tk
}

// GetTokenKeys returns tokenKeys.
func GetTokenKeys() *model.AuthTokenKeypair {
	return tokenKeys
}

// Bun returns the singleton database connection through the bun library. bun is the database
// library we have decided to use for new code in the future due to its superior composability
// over bare SQL, and its superior flexibility over e.g. gorm.  New code should not use the old bare
// SQL tooling.
func Bun() *bun.DB {
	if theOneBun == nil {
		panic("Bun is not yet initialized!  Did you use the database before initializing it?")
	}
	return theOneBun
}

// SingleDB returns a singleton database client. Bun() should be preferred over this for all new
// queries.
func SingleDB() *PgDB {
	if theOneDB == nil {
		panic("DB is not yet initialized!  Did you use the database before initializing it?")
	}
	return theOneDB
}

// SortDirection represents the order by in a query.
type SortDirection string

const (
	// SortDirectionAsc represents ordering by ascending.
	SortDirectionAsc SortDirection = "ASC"
	// SortDirectionDesc represents ordering by descending.
	SortDirectionDesc SortDirection = "DESC"
	// SortDirectionAscNullsFirst represents ordering by ascending with nulls first.
	SortDirectionAscNullsFirst SortDirection = "ASC NULLS FIRST"
	// SortDirectionDescNullsLast represents ordering by descending with nulls last.
	SortDirectionDescNullsLast SortDirection = "DESC NULLS LAST"
)

// PaginateBun adds sorting and pagination to the provided bun query, defaulting to certain values
// if they are not specified. By default, we order by ascending on the id column, with no limit.
func PaginateBun(
	query *bun.SelectQuery,
	orderColumn string,
	direction SortDirection,
	offset,
	limit int,
) *bun.SelectQuery {
	if orderColumn == "" {
		orderColumn = id
	}
	if len(direction) == 0 {
		direction = SortDirectionAsc
	}
	orderExp := fmt.Sprintf("%s %s", orderColumn, direction)

	query = query.Order(orderExp)

	query = query.Offset(offset)

	if limit > 0 {
		query = query.Limit(limit)
	}

	return query
}

// PaginateBunUnsafe is a version of PaginateBun that
// allows an arbitrary order expression like `metrics->>'loss'`.
func PaginateBunUnsafe(
	query *bun.SelectQuery,
	orderColumn string,
	direction SortDirection,
	offset,
	limit int,
) *bun.SelectQuery {
	if orderColumn == "" {
		orderColumn = id
	}
	if len(direction) == 0 {
		direction = SortDirectionAsc
	}
	orderExp := fmt.Sprintf("%s %s", orderColumn, direction)

	query = query.OrderExpr(orderExp)

	query = query.Offset(offset)

	if limit > 0 {
		query = query.Limit(limit)
	}

	return query
}

// PgDB represents a Postgres database connection.  The type definition is needed to define methods.
type PgDB struct {
	tokenKeys *model.AuthTokenKeypair
	sql       *sqlx.DB
	queries   *StaticQueryMap
	url       string
}

// ConnectPostgres connects to a Postgres database.
func ConnectPostgres(url string) (*PgDB, error) {
	numTries := 0
	for {
		sql, err := sqlx.Connect("pgx", url)
		if err == nil {
			db := &PgDB{sql: sql, queries: &StaticQueryMap{}, url: url}
			initTheOneBun(db)
			return db, nil
		}
		numTries++
		if numTries >= 15 {
			return nil, errors.Wrapf(err, "could not connect to database after %v tries", numTries)
		}
		toWait := 4 * time.Second
		time.Sleep(toWait)
		log.WithError(err).Warnf("failed to connect to postgres, trying again in %s", toWait)
	}
}

const (
	// CodeUniqueViolation is the error code that Postgres uses to indicate that an attempted
	// insert/update violates a uniqueness constraint.  Obtained from:
	// https://www.postgresql.org/docs/10/errcodes-appendix.html
	CodeUniqueViolation = "23505"
	// CodeForeignKeyViolation is the error code that Postgres uses to indicate that an attempted
	// insert/update violates a foreign key constraint.  Obtained from:
	// https://www.postgresql.org/docs/10/errcodes-appendix.html
	CodeForeignKeyViolation = "23503"
)

// Close closes the underlying pq connection.
func (db *PgDB) Close() error {
	return db.sql.Close()
}

// namedGet is a convenience method for a named query for a single value.
func (db *PgDB) namedGet(dest interface{}, query string, arg interface{}) error {
	nstmt, err := db.sql.PrepareNamed(query)
	if err != nil {
		return errors.Wrapf(err, "error preparing query %s", query)
	}

	defer nstmt.Close()

	if sErr := nstmt.QueryRowx(arg).Scan(dest); sErr != nil {
		err = errors.Wrapf(sErr, "error scanning query %s", query)
	}
	if cErr := nstmt.Close(); cErr != nil && err != nil {
		err = errors.Wrap(cErr, "error closing named DB statement")
	}

	return err
}

// namedExecOne is a convenience method for a NamedExec that should affect only one row.
func (db *PgDB) namedExecOne(query string, arg interface{}) error {
	res, err := db.sql.NamedExec(query, arg)
	if err != nil {
		return errors.Wrapf(err, "error in query %v \narg %v", query, arg)
	}
	num, err := res.RowsAffected()
	if err != nil {
		return errors.Wrapf(
			err,
			"error checking rows affected for query %v\n arg %v",
			query, arg)
	}
	if num != 1 {
		return errors.Errorf("error: %v rows affected on query %v \narg %v", num, query, arg)
	}
	return nil
}

// namedExecOne is a convenience method for a NamedExec that should affect only one row.
func namedExecOne(tx *sqlx.Tx, query string, arg interface{}) error {
	res, err := tx.NamedExec(query, arg)
	if err != nil {
		return errors.Wrapf(err, "error in query %v \narg %v", query, arg)
	}
	num, err := res.RowsAffected()
	if err != nil {
		return errors.Wrapf(
			err,
			"error checking rows affected for query %v\n arg %v",
			query, arg)
	}
	if num != 1 {
		return errors.Errorf("error: %v rows affected on query %v \narg %v", num, query, arg)
	}
	return nil
}

func queryBinds(fields []string) []string {
	binds := make([]string, 0, len(fields))
	for _, field := range fields {
		binds = append(binds, ":"+field)
	}
	return binds
}

func setClause(fields []string) string {
	sets := make([]string, 0, len(fields))
	binds := queryBinds(fields)
	for i, field := range fields {
		sets = append(sets, fmt.Sprintf("%v = %v", field, binds[i]))
	}
	return fmt.Sprintf("SET\n%v", strings.Join(sets, ",\n"))
}

func (db *PgDB) rawQuery(q string, args ...interface{}) ([]byte, error) {
	var ret []byte
	if err := db.sql.QueryRowx(q, args...).Scan(&ret); err == sql.ErrNoRows {
		return nil, errors.WithStack(ErrNotFound)
	} else if err != nil {
		return nil, errors.WithStack(err)
	}
	return ret, nil
}

// query executes a query returning a single row and unmarshals the result into an obj.
func (db *PgDB) query(q string, obj interface{}, args ...interface{}) error {
	if err := db.sql.QueryRowx(q, args...).StructScan(obj); err == sql.ErrNoRows {
		return errors.WithStack(ErrNotFound)
	} else if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// query executes a query returning a single row and unmarshals the result into a slice.
func (db *PgDB) queryRows(query string, v interface{}, args ...interface{}) error {
	parser := func(rows *sqlx.Rows, val interface{}) error { return rows.StructScan(val) }
	return db.queryRowsWithParser(query, parser, v, args...)
}

func (db *PgDB) queryRowsWithParser(
	query string, p func(*sqlx.Rows, interface{}) error, v interface{}, args ...interface{},
) error {
	rows, err := db.sql.Queryx(query, args...)
	if err != nil {
		return err
	}

	// Defer once now, ignoring errors, to ensure cleanup occurs.  Also close at the end, capturing
	// the error, to ensure we don't drop any errors.
	defer rows.Close()

	vType := reflect.TypeOf(v).Elem()
	switch kind := vType.Kind(); kind {
	case reflect.Slice:
		vValue := reflect.ValueOf(v).Elem()
		vValue.Set(reflect.MakeSlice(vValue.Type(), 0, 0))
		for rows.Next() {
			switch k := vValue.Type().Elem().Kind(); k {
			case reflect.Ptr:
				sValue := reflect.New(vValue.Type().Elem().Elem())
				if err = p(rows, sValue.Interface()); err != nil {
					return errors.Wrap(err, "queryRowsWithParser[ptr]")
				}
				vValue = reflect.Append(vValue, sValue)
			case reflect.Struct:
				sValue := reflect.New(vValue.Type().Elem())
				if err = p(rows, sValue.Interface()); err != nil {
					return errors.Wrap(err, "queryRowsWithParser[struct]")
				}
				vValue = reflect.Append(vValue, sValue.Elem())
			default:
				return errors.Errorf("unexpected type: %s", k)
			}
		}
		reflect.ValueOf(v).Elem().Set(vValue)
	case reflect.Struct:
		if !rows.Next() {
			return ErrNotFound
		}
		if err = p(rows, v); err != nil {
			return err
		}
	default:
		panic(fmt.Sprintf("unsupported query type: %s", kind))
	}

	if err := rows.Close(); err != nil { //nolint: sqlclosecheck
		return errors.Wrapf(err, "rows.Close()")
	}

	return nil
}

// Query returns the result of the query. Any placeholder parameters are replaced
// with supplied params.
func (db *PgDB) Query(queryName string, v interface{}, params ...interface{}) error {
	parser := func(rows *sqlx.Rows, val interface{}) error { return rows.StructScan(val) }
	return db.queryRowsWithParser(db.queries.GetOrLoad(queryName), parser, v, params...)
}

// QueryF returns the result of the formatted query. Any placeholder parameters are replaced
// with supplied params.
func (db *PgDB) QueryF(
	queryName string, args []interface{}, v interface{}, params ...interface{},
) error {
	parser := func(rows *sqlx.Rows, val interface{}) error { return rows.StructScan(val) }
	query := db.queries.GetOrLoad(queryName)
	if len(args) > 0 {
		query = fmt.Sprintf(query, args...)
	}
	return db.queryRowsWithParser(query, parser, v, params...)
}

// RawQuery returns the result of the query as a raw byte string. Any placeholder parameters are
// replaced with supplied params.
func (db *PgDB) RawQuery(queryName string, params ...interface{}) ([]byte, error) {
	return db.rawQuery(db.queries.GetOrLoad(queryName), params...)
}

// withTransaction executes a function with a transaction.
func (db *PgDB) withTransaction(name string, exec func(tx *sqlx.Tx) error) error {
	tx, err := db.sql.Beginx()
	if err != nil {
		return errors.Wrapf(err, "failed to start transaction (%s)", name)
	}
	defer func() {
		if tx == nil {
			return
		}

		if rErr := tx.Rollback(); rErr != nil {
			log.Errorf("failed to rollback transaction (%s): %v", name, rErr)
		}
	}()

	if err = exec(tx); err != nil {
		return errors.Wrapf(err, "failed to exec transaction (%s)", name)
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrapf(err, "failed to commit transaction: (%s)", name)
	}

	tx = nil
	return nil
}
