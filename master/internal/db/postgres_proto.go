package db

import (
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/determined-ai/determined/master/pkg/model"
)

// QueryProto returns the result of the query. Any placeholder parameters are replaced
// with supplied args. Enum values must be the full name of the enum.
func (db *PgDB) QueryProto(queryName string, v interface{}, args ...interface{}) error {
	err := db.queryRowsWithParser(db.queries.getOrLoad(queryName), protoParser, v, args...)
	if err == ErrNotFound {
		return err
	}
	return errors.Wrapf(err, "error running query: %v", queryName)
}

// QueryProtof returns the result of the formated query. Any placeholder parameters are replaced
// with supplied params.
func (db *PgDB) QueryProtof(
	queryName string, args []interface{}, v interface{}, params ...interface{},
) error {
	query := db.queries.getOrLoad(queryName)
	if len(args) > 0 {
		query = fmt.Sprintf(query, args...)
	}
	return errors.Wrapf(
		db.queryRowsWithParser(query, protoParser, v, params...),
		"error running query: %v", queryName,
	)
}

func protoParser(rows *sqlx.Rows, val interface{}) error {
	message, ok := val.(proto.Message)
	if !ok {
		return errors.Errorf("invalid type conversion: %T is not a Protobuf message", val)
	}
	dest := make(map[string]interface{})
	if err := rows.MapScan(dest); err != nil {
		return errors.Wrap(err, "error reading row from database")
	}
	for key, value := range dest {
		switch parsed := value.(type) {
		case float64:
			dest[key] = model.ExtendedFloat64(parsed)
		case []byte:
			var marshaled interface{}
			if err := json.Unmarshal(parsed, &marshaled); err != nil {
				return errors.Wrapf(err, "error parsing field: %s", key)
			}
			dest[key] = marshaled
		}
	}
	bytes, err := json.Marshal(dest)
	if err != nil {
		return errors.Wrapf(err, "error converting row to json bytes: %s", dest)
	}
	return errors.Wrapf(protojson.Unmarshal(bytes, message),
		"error converting row to Protobuf struct")
}
