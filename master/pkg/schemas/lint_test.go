package schemas

import (
	"encoding/json"
	"strings"
	"testing"

	"gotest.tools/assert"
)

type BadDefaultStruct struct {
	Val string `json:"val"`
}

func (b BadDefaultStruct) DefaultSource() interface{} {
	raw := `{
		"properties": {
			"val": {
				"default": "val-default"
			}
		}
	}`
	var out interface{}
	err := json.Unmarshal([]byte(raw), &out)
	if err != nil {
		panic(err.Error())
	}
	return out
}

func TestLintStructDefaults(t *testing.T) {
	var b BadDefaultStruct
	errs := LintStructDefaults(b)
	assert.Assert(t, len(errs) == 1)
	assert.Assert(t, strings.Contains(errs[0].Error(), "non-pointer type"))
}
