package expconf

import (
	"bytes"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v2"

	"github.com/determined-ai/determined/master/pkg/schemas/extensions"
)

var sanityValidatorsV1 = map[string]*jsonschema.Schema{}
var completenessValidatorsV1 = map[string]*jsonschema.Schema{}

// Create a jsonschema.Compiler with all the schemas preloaded.
func newCompiler() *jsonschema.Compiler {
	compiler := jsonschema.NewCompiler()

	for url, byts := range schemaBytesMap() {
		if err := compiler.AddResource(url, bytes.NewReader(byts)); err != nil {
			panic("invalid schema: " + url)
		}
	}

	return compiler
}

func getSanityValidatorV1(url string) *jsonschema.Schema {
	if url == "" {
		// By default, return the validator for the experiment config object.
		url = parsedExperimentConfigV1().(JSONObject)["$id"].(string)
	}

	// Check if we have a pre-compiled validator already.
	if validator, ok := sanityValidatorsV1[url]; ok {
		return validator
	}

	compiler := newCompiler()

	// Sanity check means eventuallyRequired isn't required yet.
	compiler.Extensions["disallowProperties"] = extensions.DisallowPropertiesExtension()
	compiler.Extensions["union"] = extensions.UnionExtension()
	compiler.Extensions["checks"] = extensions.ChecksExtension()
	compiler.Extensions["compareProperties"] = extensions.ComparePropertiesExtension()
	compiler.Extensions["conditional"] = extensions.ConditionalExtension()

	validator, err := compiler.Compile(url)
	if err != nil {
		panic("uncompilable schema: " + url)
	}

	// Remember this validator for later.
	sanityValidatorsV1[url] = validator

	return validator
}

func getCompletenessValidatorV1(url string) *jsonschema.Schema {
	if url == "" {
		// By default, return the validator for the experiment config object.
		url = parsedExperimentConfigV1().(JSONObject)["$id"].(string)
	}

	if validator, ok := completenessValidatorsV1[url]; ok {
		return validator
	}

	compiler := newCompiler()

	// Completeness means eventuallyRequired is now required.
	compiler.Extensions["disallowProperties"] = extensions.DisallowPropertiesExtension()
	compiler.Extensions["union"] = extensions.UnionExtension()
	compiler.Extensions["checks"] = extensions.ChecksExtension()
	compiler.Extensions["compareProperties"] = extensions.ComparePropertiesExtension()
	compiler.Extensions["conditional"] = extensions.ConditionalExtension()
	compiler.Extensions["eventuallyRequired"] = extensions.EventuallyRequiredExtension()

	validator, err := compiler.Compile(url)
	if err != nil {
		panic("uncompilable schema: " + url)
	}

	completenessValidatorsV1[url] = validator

	return validator
}

// SaneYamlV1 validates yaml-formatted bytes, return if it was valid and a list of errors.
// "Sane" means it might not satisfied the "eventuallyRequired" keywords but that everything else
// is valid.
func SaneYamlV1(byts []byte) (bool, []string) {
	byts, err := jsonFromYaml(byts)
	if err != nil {
		return false, []string{fmt.Sprintf("%v", err)}
	}
	return SaneJSONV1(byts)
}

// SaneJSONV1 validates yaml-formatted bytes, return if it was valid and a list of errors.
// "Sane" means it might not satisfied the "eventuallyRequired" keywords but that everything else
// is valid.
func SaneJSONV1(byts []byte) (bool, []string) {
	schema := getSanityValidatorV1("")
	err := schema.Validate(bytes.NewReader(byts))
	if err == nil {
		return true, nil
	}

	return false, getRenderedErrors(err, byts)
}

// CompleteYamlV1 validates yaml-formatted bytes, return if it was valid and a list of errors.
func CompleteYamlV1(byts []byte) (bool, []string) {
	byts, err := jsonFromYaml(byts)
	if err != nil {
		return false, []string{fmt.Sprintf("%v", err)}
	}
	return CompleteJSONV1(byts)
}

// CompleteJSONV1 validates json-formatted bytes, return if it was valid and a list of errors.
func CompleteJSONV1(byts []byte) (bool, []string) {
	schema := getCompletenessValidatorV1("")
	err := schema.Validate(bytes.NewReader(byts))
	if err == nil {
		return true, nil
	}

	return false, getRenderedErrors(err, byts)
}
