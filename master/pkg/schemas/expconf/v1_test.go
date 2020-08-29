package expconf

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/assert"
)

type SchemaTestCase struct {
	Name    string               `json:"name"`
	Matches *[]string            `json:"matches"`
	Errors  *map[string][]string `json:"errors"`
	Case    JSON                 `json:"case"`
}

func errorIn(expect string, errors []string) bool {
	for _, msg := range errors {
		if strings.Contains(msg, expect) {
			return true
		}
	}
	return false
}

func (tc SchemaTestCase) CheckMatches(t *testing.T) {
	if tc.Matches == nil {
		return
	}
	byts, err := json.Marshal(tc.Case)
	assert.NilError(t, err)
	for _, url := range *tc.Matches {
		schema := getCompletenessValidatorV1(url)
		err := schema.Validate(bytes.NewReader(byts))
		if err == nil {
			continue
		}
		// Unexpected errors.
		rendered := getRenderedErrors(err, byts)
		t.Errorf("errors matching %v:\n%v", url, strings.Join(rendered, "\n"))
	}
}

func (tc SchemaTestCase) CheckErrors(t *testing.T) {
	if tc.Errors == nil {
		return
	}
	byts, err := json.Marshal(tc.Case)
	assert.NilError(t, err)
	for url, expectedErrors := range *tc.Errors {
		schema := getCompletenessValidatorV1(url)
		err := schema.Validate(bytes.NewReader(byts))
		if err == nil {
			t.Errorf("expected error matching %v but got none", url)
			continue
		}
		rendered := getRenderedErrors(err, byts)
		for _, expect := range expectedErrors {
			if !errorIn(expect, rendered) {
				t.Errorf(
					"while matching %v\ndid not find error:\n    %q\nin:\n    %v",
					url,
					expect,
					strings.Join(rendered, "\n    "),
				)
			}
		}
	}
}

func RunCasesFile(t *testing.T, path string) {
	// Ignore the security error about including files as variables; this is just a test.
	byts, err := ioutil.ReadFile(path) //nolint: gosec
	assert.NilError(t, err)

	jbyts, err := jsonFromYaml(byts)
	assert.NilError(t, err)

	var cases []SchemaTestCase
	err = json.Unmarshal(jbyts, &cases)
	assert.NilError(t, err)

	for _, testCase := range cases {
		tc := testCase
		t.Run(tc.Name, func(t *testing.T) {
			tc.CheckMatches(t)
			tc.CheckErrors(t)
		})
	}
}

func TestExperimentConfig(t *testing.T) {
	dir := "../../../../schemas/test_cases/v1"
	files, err := ioutil.ReadDir(dir)
	assert.NilError(t, err)

	for _, file := range files {
		path := filepath.Join(dir, file.Name())
		if strings.HasSuffix(path, ".yaml") {
			RunCasesFile(t, path)
		}
	}
}
