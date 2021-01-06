package schemas

import (
	"bytes"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/santhosh-tekuri/jsonschema/v2"

	"github.com/determined-ai/determined/master/pkg/schemas/extensions"
)

// Schema defines some basic knowledge that helper functions like IsSane or
// IsComplete need to operate.
type Schema interface {
	ParsedSchema() interface{}
	SanityValidator() *jsonschema.Schema
	CompletenessValidator() *jsonschema.Schema
}

// IsSane ensure that the schema is valid, modulo some fields which are
// eventuallyRequired but not required.
func IsSane(schema Schema) (bool, []error) {
	byts, err := json.Marshal(schema)
	if err != nil {
		return false, []error{errors.Wrap(err, "json marshal failed")}
	}

	validator := schema.SanityValidator()
	err = validator.Validate(bytes.NewReader(byts))
	if err == nil {
		return true, nil
	}

	return false, GetRenderedErrors(err, byts)
}

// IsComplete ensures that the schema is totally valid, including
// eventuallyRequired fields.
func IsComplete(schema Schema) (bool, []error) {
	byts, err := json.Marshal(schema)
	if err != nil {
		return false, []error{errors.Wrap(err, "json marshal failed")}
	}

	validator := schema.CompletenessValidator()
	err = validator.Validate(bytes.NewReader(byts))
	if err == nil {
		return true, nil
	}

	return false, GetRenderedErrors(err, byts)
}

var sanityValidators = map[string]*jsonschema.Schema{}
var completenessValidators = map[string]*jsonschema.Schema{}

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

func GetSanityValidator(url string) *jsonschema.Schema {
	// Check if we have a pre-compiled validator already.
	if validator, ok := sanityValidators[url]; ok {
		return validator
	}

	compiler := newCompiler()

	// Sanity check means eventuallyRequired isn't required yet.
	compiler.Extensions["disallowProperties"] = extensions.DisallowPropertiesExtension()
	compiler.Extensions["union"] = extensions.UnionExtension()
	compiler.Extensions["checks"] = extensions.ChecksExtension()
	compiler.Extensions["compareProperties"] = extensions.ComparePropertiesExtension()
	compiler.Extensions["conditional"] = extensions.ConditionalExtension()
	compiler.Extensions["optionalRef"] = extensions.OptionalRefExtension()

	validator, err := compiler.Compile(url)
	if err != nil {
		panic("uncompilable schema: " + url)
	}

	// Remember this validator for later.
	sanityValidators[url] = validator

	return validator
}

func GetCompletenessValidator(url string) *jsonschema.Schema {
	if validator, ok := completenessValidators[url]; ok {
		return validator
	}

	compiler := newCompiler()

	// Completeness means eventuallyRequired is now required.
	compiler.Extensions["disallowProperties"] = extensions.DisallowPropertiesExtension()
	compiler.Extensions["union"] = extensions.UnionExtension()
	compiler.Extensions["checks"] = extensions.ChecksExtension()
	compiler.Extensions["compareProperties"] = extensions.ComparePropertiesExtension()
	compiler.Extensions["conditional"] = extensions.ConditionalExtension()
	compiler.Extensions["optionalRef"] = extensions.OptionalRefExtension()
	compiler.Extensions["eventuallyRequired"] = extensions.EventuallyRequiredExtension()

	validator, err := compiler.Compile(url)
	if err != nil {
		panic("uncompilable schema: " + url)
	}

	completenessValidators[url] = validator

	return validator
}
