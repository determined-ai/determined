//nolint:exhaustivestruct
package expconf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/santhosh-tekuri/jsonschema/v2"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/schemas"
)

type JSON = interface{}

type SchemaTestCase struct {
	Name               string               `json:"name"`
	SaneAs             *[]string            `json:"sane_as"`
	CompleteAs         *[]string            `json:"complete_as"`
	SanityErrors       *map[string][]string `json:"sanity_errors"`
	CompletenessErrors *map[string][]string `json:"completeness_errors"`
	DefaultAs          *string              `json:"default_as"`
	Defaulted          *JSON                `json:"defaulted"`
	Case               JSON                 `json:"case"`
	MergeAs            *string              `json:"merge_as"`
	MergeSrc           *JSON                `json:"merge_src"`
	Merged             *JSON                `json:"merged"`
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

func (tc SchemaTestCase) CheckSaneAs(t *testing.T) {
	if tc.SaneAs == nil {
		return
	}
	byts, err := json.Marshal(tc.Case)
	assert.NilError(t, err)
	for _, url := range *tc.SaneAs {
		schema := schemas.GetSanityValidator(url)
		err := schema.Validate(bytes.NewReader(byts))
		if err == nil {
			continue
		}
		// Unexpected errors.
		rendered := schemas.GetRenderedErrors(err, byts)
		t.Errorf("errors matching %v:\n%v", url, schemas.JoinErrors(rendered, "\n"))
	}
}

func (tc SchemaTestCase) CheckCompleteAs(t *testing.T) {
	if tc.CompleteAs == nil {
		return
	}
	byts, err := json.Marshal(tc.Case)
	assert.NilError(t, err)
	for _, url := range *tc.CompleteAs {
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

func (tc SchemaTestCase) CheckSanityErrors(t *testing.T) {
	if tc.SanityErrors == nil {
		return
	}
	tc.checkErrors(t, tc.SanityErrors, "sanity")
}

func (tc SchemaTestCase) CheckCompletenessErrors(t *testing.T) {
	if tc.CompletenessErrors == nil {
		return
	}
	tc.checkErrors(t, tc.CompletenessErrors, "completeness")
}

func (tc SchemaTestCase) checkErrors(t *testing.T, errors *map[string][]string, testType string) {
	byts, err := json.Marshal(tc.Case)
	assert.NilError(t, err)
	for url, expectedErrors := range *errors {
		var schema *jsonschema.Schema
		if testType == "sanity" {
			schema = schemas.GetSanityValidator(url)
		} else {
			schema = schemas.GetCompletenessValidator(url)
		}
		err := schema.Validate(bytes.NewReader(byts))
		if err == nil {
			t.Errorf("expected error %v validating %v but got none", testType, url)
			continue
		}
		rendered := schemas.GetRenderedErrors(err, byts)
		for _, expect := range expectedErrors {
			if !errorIn(expect, rendered) {
				t.Errorf(
					"while %v validating %v\ndid not find a match to the pattern:\n    %q\nin:\n    %v",
					testType,
					url,
					expect,
					schemas.JoinErrors(rendered, "\n    "),
				)
			}
		}
	}
}

// Reverse-lookup of which object to umarshal into for a url, needed for default testing.
func objectForURL(url string) interface{} {
	switch url {
	case "http://determined.ai/schemas/expconf/v0/experiment.json":
		return &ExperimentConfigV0{}
	case "http://determined.ai/schemas/expconf/v0/bind-mount.json":
		return &BindMountV0{}
	case "http://determined.ai/schemas/expconf/v0/bind-mounts.json":
		return &BindMountsConfigV0{}
	case "http://determined.ai/schemas/expconf/v0/devices.json":
		return &DevicesConfigV0{}
	case "http://determined.ai/schemas/expconf/v0/environment.json":
		return &EnvironmentConfigV0{}
	case "http://determined.ai/schemas/expconf/v0/resources.json":
		return &ResourcesConfigV0{}
	// For union member schemas, just return the union type.
	case "http://determined.ai/schemas/expconf/v0/searcher.json",
		"http://determined.ai/schemas/expconf/v0/searcher-adaptive-asha.json",
		"http://determined.ai/schemas/expconf/v0/searcher-async-halving.json",
		"http://determined.ai/schemas/expconf/v0/searcher-custom.json",
		"http://determined.ai/schemas/expconf/v0/searcher-grid.json",
		"http://determined.ai/schemas/expconf/v0/searcher-random.json",
		"http://determined.ai/schemas/expconf/v0/searcher-single.json":
		return &SearcherConfigV0{}
	case "http://determined.ai/schemas/expconf/v0/checkpoint-storage.json":
		return &CheckpointStorageConfigV0{}
	case "http://determined.ai/schemas/expconf/v0/hyperparameter.json",
		"http://determined.ai/schemas/expconf/v0/hyperparameter-int.json":
		return &HyperparameterV0{}

	// Test-related structs.
	case "http://determined.ai/schemas/expconf/v0/test-root.json":
		return &TestRootV0{}
	case "http://determined.ai/schemas/expconf/v0/test-union.json",
		"http://determined.ai/schemas/expconf/v0/test-union-a.json",
		"http://determined.ai/schemas/expconf/v0/test-union-b.json":
		return &TestUnionV0{}
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
				cpy := objValue
				clearRuntimeDefaults(&cpy, defaultedValue)
				// Update the modified value in the original map.
				tObj[key] = cpy
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
	if tc.Defaulted == nil && tc.DefaultAs == nil {
		return
	}

	if tc.Defaulted == nil || tc.DefaultAs == nil {
		assert.NilError(t, errors.New(
			"if either of default_as, or defaulted are set in a test case, "+
				"they must both be set",
		))
	}

	byts, err := json.Marshal(tc.Case)
	assert.NilError(t, err)

	// Get an empty object to marshal into.
	obj := objectForURL(*tc.DefaultAs)

	testName := fmt.Sprintf("defaulted %T", obj)
	t.Run(testName, func(t *testing.T) {
		err = json.Unmarshal(byts, &obj)
		assert.NilError(t, err)

		obj = schemas.WithDefaults(obj)

		// Compare json-to-json.
		defaultedBytes, err := json.Marshal(obj)
		assert.NilError(t, err)
		var rawObj interface{}
		err = json.Unmarshal(defaultedBytes, &rawObj)
		assert.NilError(t, err)

		// Just before the comparison, allow any instances of "*" in the defaulted value to match
		// any non-nil value.
		clearRuntimeDefaults(&rawObj, *tc.Defaulted)

		assert.DeepEqual(t, rawObj, *tc.Defaulted)
	})
}

func (tc SchemaTestCase) CheckMerged(t *testing.T) {
	if tc.MergeAs == nil && tc.MergeSrc == nil && tc.Merged == nil {
		return
	}

	if tc.MergeAs == nil || tc.MergeSrc == nil || tc.Merged == nil {
		assert.NilError(t, errors.New(
			"if any of merge_as, merge_src, or merged are set in a test case, "+
				"they must all be set",
		))
	}

	objBytes, err := json.Marshal(tc.Case)
	assert.NilError(t, err)

	srcBytes, err := json.Marshal(*tc.MergeSrc)
	assert.NilError(t, err)

	url := (*tc.MergeAs)

	// Get an empty objects to unmarshal into.
	obj := objectForURL(url)
	src := objectForURL(url)

	testName := fmt.Sprintf("merged %T", obj)
	t.Run(testName, func(t *testing.T) {
		assert.NilError(t, schemas.SaneBytes(obj.(schemas.Schema), objBytes))
		assert.NilError(t, schemas.SaneBytes(src.(schemas.Schema), srcBytes))

		err = json.Unmarshal(objBytes, &obj)
		assert.NilError(t, err)

		err = json.Unmarshal(srcBytes, &src)
		assert.NilError(t, err)

		merged := schemas.Merge(obj, src)

		// Compare json-to-json.
		mergedBytes, err := json.Marshal(merged)
		assert.NilError(t, err)
		var rawMerged interface{}
		err = json.Unmarshal(mergedBytes, &rawMerged)
		assert.NilError(t, err)

		assert.DeepEqual(t, rawMerged, *tc.Merged)
	})
}

func (tc SchemaTestCase) CheckRoundTrip(t *testing.T) {
	if tc.Defaulted == nil {
		return
	}

	byts, err := json.Marshal(tc.Case)
	assert.NilError(t, err)

	assert.Assert(t, tc.SaneAs != nil)
	url := (*tc.SaneAs)[0]

	// Unmarshal into an object once.
	obj := objectForURL(url)
	err = json.Unmarshal(byts, &obj)
	assert.NilError(t, err)

	// Round-trip through json.
	jByts, err := json.Marshal(obj)
	assert.NilError(t, err)
	cpy := objectForURL(url)
	err = json.Unmarshal(jByts, &cpy)
	assert.NilError(t, err)
	assert.DeepEqual(t, obj, cpy)

	// Round-trip again after defaults.
	obj = schemas.WithDefaults(obj)

	jByts, err = json.Marshal(obj)
	assert.NilError(t, err)

	cpy = objectForURL(url)
	err = json.Unmarshal(jByts, &cpy)
	assert.NilError(t, err)

	assert.DeepEqual(t, obj, cpy)
}

func RunCasesFile(t *testing.T, path string, displayPath string) {
	// Ignore the security error about including files as variables; this is just a test.
	byts, err := os.ReadFile(path) //nolint: gosec
	assert.NilError(t, err)

	jbyts, err := schemas.JSONFromYaml(byts)
	assert.NilError(t, err)

	var cases []SchemaTestCase
	err = json.Unmarshal(jbyts, &cases)
	assert.NilError(t, err)

	for _, testCase := range cases {
		tc := testCase
		testName := fmt.Sprintf("%v::%v", displayPath, tc.Name)
		t.Run(testName, func(t *testing.T) {
			tc.CheckSaneAs(t)
			tc.CheckCompleteAs(t)
			tc.CheckSanityErrors(t)
			tc.CheckCompletenessErrors(t)
			tc.CheckDefaulted(t)
			tc.CheckRoundTrip(t)
			tc.CheckMerged(t)
		})
	}
}

func TestExperimentConfig(t *testing.T) {
	// Call RunCasesFile on every .yaml file in the test_cases directory tree.
	dir := "../../../../schemas/test_cases"
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		assert.NilError(t, err)
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
