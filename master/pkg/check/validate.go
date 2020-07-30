package check

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/pkg/errors"
)

// Validator returns an error if the validation fails and nil otherwise.
type Validator func() error

// Validatable is implemented by anything that has fields that should be validated.
type Validatable interface {
	Validate() []error
}

type validationError struct {
	errs []error
}

func (v validationError) Error() string {
	errStrings := make([]string, 0, len(v.errs))
	for _, err := range v.errs {
		errStrings = append(errStrings, err.Error())
	}
	sort.Strings(errStrings)
	joined := strings.Join(errStrings, "\n\t")
	return fmt.Sprintf("Check Failed! %d errors found:\n\t%s", len(v.errs), joined)
}

// Validate returns an error if any of the provided validators have failed. The errors of all
// failed validators are combined into a single returned error.
func Validate(v interface{}) error {
	errs := validate(reflect.ValueOf(v), "root")
	if len(errs) == 0 {
		return nil
	}
	return validationError{errs: errs}
}

func validate(v reflect.Value, path interface{}) []error {
	var errs []error
	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			return nil
		}
		errs = append(errs, validate(v.Elem(), path)...)
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			errs = append(errs, validate(v.Index(i), fmt.Sprintf("%s[%d]", path, i))...)
		}
	case reflect.Map:
		for _, key := range v.MapKeys() {
			errs = append(errs, validate(v.MapIndex(key),
				fmt.Sprintf("%s[%s]", path, key.Interface()))...)
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if !v.Field(i).CanInterface() {
				continue
			}

			errs = append(errs, validate(v.Field(i),
				fmt.Sprintf("%s.%s", path, v.Type().Field(i).Name))...)
		}
	}

	if v.Kind() != reflect.Ptr {
		vp := reflect.New(v.Type())
		vp.Elem().Set(v)
		if validatable, ok := vp.Interface().(Validatable); ok {
			for _, err := range validatable.Validate() {
				if err != nil {
					errs = append(errs, errors.Wrapf(err, "error found at %s", path))
				}
			}
		}
	}

	return errs
}
