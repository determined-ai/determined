package tasks

import (
	"encoding/json"
	"fmt"
)

func jsonify(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Errorf("unable to marshal to JSON: %T: %w", v, err))
	}
	return string(data)
}
