package tasks

import (
	"encoding/json"

	"github.com/pkg/errors"
)

func jsonify(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(errors.Wrapf(err, "unable to marshal to JSON: %T", v))
	}
	return string(data)
}
