package api

import (
	"net/http"
	"reflect"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
)

var parsers = map[reflect.Kind]func(v string) (interface{}, error){
	reflect.String: func(v string) (interface{}, error) { return v, nil },
	reflect.Int:    func(v string) (interface{}, error) { return strconv.Atoi(v) },
	reflect.Bool:   func(v string) (interface{}, error) { return strconv.ParseBool(v) },
}

// BindArgs binds path and query parameters in the context to struct fields.
func BindArgs(i interface{}, c echo.Context) error {
	v := reflect.ValueOf(i).Elem()
	for index := 0; index < v.Type().NumField(); index++ {
		meta := v.Type().Field(index)
		if name, ok := meta.Tag.Lookup("path"); ok {
			if err := bindValue(name, meta.Type, v.Field(index), c.Param(name)); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error())
			}
		}
		if name, ok := meta.Tag.Lookup("query"); ok {
			if err := bindValue(name, meta.Type, v.Field(index), c.QueryParam(name)); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error())
			}
		}
	}
	return nil
}

func bindValue(name string, t reflect.Type, f reflect.Value, value string) error {
	if value == "" {
		// If the value is blank, handle defaulting pointer and non-pointer types appropriately.
		if t.Kind() == reflect.Ptr {
			return nil
		}
		return errors.Errorf("missing parameter: %s", name)
	}

	parser, ok := parsers[t.Kind()]
	if t.Kind() == reflect.Ptr {
		// Use the parser of the underlying type of the pointer.
		parser, ok = parsers[t.Elem().Kind()]
	}
	if !ok {
		return errors.Errorf("no parser found for kind: %v", t.Kind())
	}
	parsed, err := parser(value)
	if err != nil {
		return errors.Wrapf(err, "unable to parse to %v: %s", t.Kind(), value)
	}

	if t.Kind() == reflect.Ptr {
		// Create a pointer and set its value to the parsed value.
		f.Set(reflect.New(t.Elem()))
		f.Elem().Set(reflect.ValueOf(parsed))
	} else {
		f.Set(reflect.ValueOf(parsed))
	}
	return nil
}
