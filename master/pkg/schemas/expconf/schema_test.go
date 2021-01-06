package expconf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/schemas"
)

type JSON = interface{}

type SchemaTestCase struct {
	Name       string                  `json:"name"`
	Matches    *[]string               `json:"matches"`
	Errors     *map[string][]string    `json:"errors"`
	Defaulted  *JSON                   `json:"defaulted"`
	Case       JSON                    `json:"case"`
}

func errorIn(expect string, errors []error) bool {
	for _, msg := range errors {
		matched, err := regexp.MatchString(expect, msg.Error())
		if err != nil {
			// Regex error.
			panic(err)
		}
		if matched {
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
		schema := schemas.GetCompletenessValidator(url)
		err := schema.Validate(bytes.NewReader(byts))
		if err == nil {
			continue
		}
		// Unexpected errors.
		rendered := schemas.GetRenderedErrors(err, byts)
		t.Errorf("errors matching %v:\n%v", url, schemas.JoinErrors(rendered, "\n"))
	}
}

func (tc SchemaTestCase) CheckErrors(t *testing.T) {
	if tc.Errors == nil {
		return
	}
	byts, err := json.Marshal(tc.Case)
	assert.NilError(t, err)
	for url, expectedErrors := range *tc.Errors {
		schema := schemas.GetCompletenessValidator(url)
		err := schema.Validate(bytes.NewReader(byts))
		if err == nil {
			t.Errorf("expected error matching %v but got none", url)
			continue
		}
		rendered := schemas.GetRenderedErrors(err, byts)
		for _, expect := range expectedErrors {
			if !errorIn(expect, rendered) {
				t.Errorf(
					"while validating %v\ndid not find a match to the pattern:\n    %q\nin:\n    %v",
					url,
					expect,
					schemas.JoinErrors(rendered, "\n    "),
				)
			}
		}
	}
}

// Reverse-lookup of which object to umarshal into for a url, needed for default testing.
func objectForUrl(url string) interface{} {
	switch url {
	case "http://determined.ai/schemas/expconf/v0/experiment.json":
		return &ExperimentConfigV0{}
	case "http://determined.ai/schemas/expconf/v0/bind-mount.json":
		return &BindMountV0{}
	// Many core values are only in the core union type
	case "http://determined.ai/schemas/expconf/v0/searcher.json",
		"http://determined.ai/schemas/expconf/v0/searcher-adaptive-asha.json",
		"http://determined.ai/schemas/expconf/v0/searcher-adaptive.json",
		"http://determined.ai/schemas/expconf/v0/searcher-adaptive-simple.json",
		"http://determined.ai/schemas/expconf/v0/searcher-async-halving.json",
		"http://determined.ai/schemas/expconf/v0/searcher-grid.json",
		"http://determined.ai/schemas/expconf/v0/searcher-pbt.json",
		"http://determined.ai/schemas/expconf/v0/searcher-random.json",
		"http://determined.ai/schemas/expconf/v0/searcher-single.json",
		"http://determined.ai/schemas/expconf/v0/searcher-sync-halving.json":
		return &SearcherConfig{}
	case "http://determined.ai/schemas/expconf/v0/hyperparameter.json",
		"http://determined.ai/schemas/expconf/v0/hyperparameter-int.json":
		return &Hyperparameter{}
	default:
		panic(fmt.Sprintf("No object to match %v, maybe you need to add one?", url))
	}
}

// clearRuntimeDefaults recurses through a pair of nested json objects, looking for instances of "*"
// in the defaulted object.  If the matching element in the object is not null, clearRuntimeDefaults
// will set that element to also be "*" so that an assert.DeepEqual will pass.
func clearRuntimeDefaults(obj *interface{}, defaulted interface{}) {
	// If defaulted is a "*" and obj is not nil, set obj to be "*" too so they match.
	if s, ok := defaulted.(string); ok && s == "*" {
		if *obj != nil {
			fmt.Fprintf(os.Stderr, "%v matches %v\n", *obj, defaulted)
			*obj = "*"
		}
	}

	// Recurse into json objects and arrays.
	switch tObj := (*obj).(type) {
	case map[string]interface{}:
		tDefaulted, ok := defaulted.(map[string]interface{})
		if !ok {
			return
		}
		for key, objValue := range tObj {
			if defaultedValue, ok := tDefaulted[key]; ok {
				clearRuntimeDefaults(&objValue, defaultedValue)
				// Update the modified value in the original map.
				tObj[key] = objValue
			}
		}
	case []interface{}:
		tDefaulted, ok := defaulted.([]interface{})
		if !ok {
			return
		}
		for i := range tObj {
			if i == len(tDefaulted) {
				break
			}
			clearRuntimeDefaults(&tObj[i], tDefaulted[i])
		}
	}
}


func (tc SchemaTestCase) CheckDefaulted(t *testing.T) {
	if tc.Defaulted == nil {
		return
	}

	byts, err := json.Marshal(tc.Case)
	assert.NilError(t, err)

	// Unmarshal against the first item in "matches".
	assert.Assert(t, tc.Matches != nil)
	url := (*tc.Matches)[0]

	// Get an empty object to marshal into.
	obj := objectForUrl(url)

	testName := fmt.Sprintf("defaulted %T", obj)
	t.Run(testName, func(t *testing.T) {

		err = json.Unmarshal(byts, &obj)
		assert.NilError(t, err)

		// XXX: this fails with &obj... but I do not understand why at all
		schemas.FillDefaults(obj)

		// Compare json-to-json.
		defaultedBytes, err := json.Marshal(obj)
		assert.NilError(t, err)
		var rawObj interface{}
		err = json.Unmarshal(defaultedBytes, &rawObj)
		assert.NilError(t, err)

		// Just before the comparison, allow any instances of "*" in the defaulted value to match
		// any non-nil value.
		clearRuntimeDefaults(&rawObj, *tc.Defaulted)

		assert.DeepEqual(t, *tc.Defaulted, rawObj)
	})
}

func (tc SchemaTestCase) CheckRoundTrip(t *testing.T) {
	if tc.Defaulted == nil {
		return
	}

	byts, err := json.Marshal(tc.Case)
	assert.NilError(t, err)

	assert.Assert(t, tc.Matches != nil)
	url := (*tc.Matches)[0]

	// Unmarshal into an object once.
	obj := objectForUrl(url)
	err = json.Unmarshal(byts, &obj)
	assert.NilError(t, err)

	// Round-trip through json.
	jByts, err := json.Marshal(obj)
	assert.NilError(t, err)
	cpy := objectForUrl(url)
	err = json.Unmarshal(jByts, &cpy)
	assert.NilError(t, err)
	assert.DeepEqual(t, obj, cpy)

	// Round-trip again after defaults.
	schemas.FillDefaults(obj)
	jByts, err = json.Marshal(obj)
	assert.NilError(t, err)
	cpy = objectForUrl(url)
	schemas.FillDefaults(cpy)
	err = json.Unmarshal(jByts, &cpy)
	assert.NilError(t, err)
	assert.DeepEqual(t, obj, cpy)
}

func RunCasesFile(t *testing.T, path string, displayPath string) {
	// Ignore the security error about including files as variables; this is just a test.
	byts, err := ioutil.ReadFile(path) //nolint: gosec
	assert.NilError(t, err)

	jbyts, err := schemas.JsonFromYaml(byts)
	assert.NilError(t, err)

	var cases []SchemaTestCase
	err = json.Unmarshal(jbyts, &cases)
	assert.NilError(t, err)

	for _, testCase := range cases {
		tc := testCase
		testName := fmt.Sprintf("%v::%v", displayPath, tc.Name)
		t.Run(testName, func(t *testing.T) {
			tc.CheckMatches(t)
			tc.CheckErrors(t)
			tc.CheckDefaulted(t)
			tc.CheckRoundTrip(t)
		})
	}
}

func TestExperimentConfig(t *testing.T) {
	// Call RunCasesFile on every .yaml file in the test_cases directory tree.
	dir := "../../../../schemas/test_cases"
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if !strings.HasSuffix(path, ".yaml") {
			return nil
		}
		displayPath, err := filepath.Rel(dir, path)
		assert.NilError(t, err)
		RunCasesFile(t, path, displayPath)
		return nil
	})
	assert.NilError(t, err)
}
