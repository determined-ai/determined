package schemas

import (
	"bytes"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/santhosh-tekuri/jsonschema/v2"

	"github.com/determined-ai/determined/master/pkg/schemas/extensions"
)

// Schema defines some basic knowledge needed by helper functions like SaneBytes or IsComplete.
// Outside of testing, the Schema interface should always be defined in generated code.
type Schema interface {
	ParsedSchema() interface{}
	SanityValidator() *jsonschema.Schema
	CompletenessValidator() *jsonschema.Schema
}

// SaneBytes will ensure that bytes for a given schema object are valid.  Unlike IsComplete,
// SaneBytes operates on a byte array, because if you unmarshal into the object and then check, you
// will already have silently dropped the unrecognized fields.
//
// The Schema object is only used for its type, and it may be nil.
func SaneBytes(schema Schema, byts []byte) error {
	validator := schema.SanityValidator()
	err := validator.Validate(bytes.NewReader(byts))
	if err != nil {
		err = errors.New(JoinErrors(GetRenderedErrors(err, byts), "\n"))
		return errors.Wrap(err, "config is invalid")
	}
	return nil
}

// IsComplete ensures that the schema is totally valid, including
// eventuallyRequired fields.
func IsComplete(schema Schema) error {
	byts, err := json.Marshal(schema)
	if err != nil {
		return errors.Wrap(err, "json marshal failed")
	}

	validator := schema.CompletenessValidator()
	err = validator.Validate(bytes.NewReader(byts))
	if err != nil {
		err = errors.New(JoinErrors(GetRenderedErrors(err, byts), "\n"))
		return errors.Wrap(err, "config is invalid or incomplete")
	}

	return nil
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

// GetSanityValidator returns a jsonschema validator for bytes from a particular URL.
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

// GetCompletenessValidator returns a jsonschema validator for bytes from a particular URL.
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
