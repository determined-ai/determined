package db

import (
	"encoding/json"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// QueryProto returns the result of the query. Any placeholder parameters are replaced
// with supplied args. Enum values must be the full name of the enum.
func (db *PgDB) QueryProto(queryName string, v interface{}, args ...interface{}) error {
	parser := func(rows *sqlx.Rows, val interface{}) error {
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
	return db.queryRows(db.queries.getOrLoad(queryName), parser, v, args...)
}
