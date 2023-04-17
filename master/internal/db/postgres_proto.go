package db

import (
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"

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
	return fmt.Errorf("error running query: %v: %w", queryName, err)
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
	if err := db.queryRowsWithParser(query, protoParser, v, params...); err != nil {
		return fmt.Errorf("error running query: %v: %w", queryName, err)
	}

	return nil
}

func protoParser(rows *sqlx.Rows, val interface{}) error {
	message, ok := val.(proto.Message)
	if !ok {
		return fmt.Errorf("invalid type conversion: %T is not a Protobuf message", val)
	}
	dest := make(map[string]interface{})
	if err := rows.MapScan(dest); err != nil {
		return fmt.Errorf("error reading row from database: %w", err)
	}
	for key, value := range dest {
		switch parsed := value.(type) {
		case float64:
			dest[key] = model.ExtendedFloat64(parsed)
		case []byte:
			var marshaled interface{}
			if err := json.Unmarshal(parsed, &marshaled); err != nil {
				return fmt.Errorf("error parsing field: %s: %w", key, err)
			}
			dest[key] = marshaled
		}
	}
	bytes, err := json.Marshal(dest)
	if err != nil {
		return fmt.Errorf("error converting row to json bytes: %s: %w", dest, err)
	}

	if err := protojson.Unmarshal(bytes, message); err != nil {
		return fmt.Errorf("error converting row to Protobuf struct: %w", err)
	}

	return nil
}
