package db

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/determined-ai/determined/proto/pkg/apiv1"

	"github.com/determined-ai/determined/master/internal/api"
)

var validField = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

// filtersToSQL takes a slice of api.Filter and the params for the current state of the
// returned fragment will be added to and constructs a query fragment representing
// the provided filters and a full list of parameters.
//
// The user input to the filters should always be contained in api.Filter.Values and
// never the field. If the field is taken from user input, SQL injection is possible.
func filtersToSQL(
	fs []api.Filter, params []interface{}, fieldMap map[string]string,
) (string, []interface{}) {
	paramID := len(params) + 1
	var fragments []string
	for _, f := range fs {
		if !validField.MatchString(f.Field) {
			panic(fmt.Sprintf("field in filter %s contains possible SQL injection", f.Field))
		}
		filterParams := filterToParams(f)
		fragments = append(fragments, filterToSQL(f, filterParams, paramID, fieldMap))
		params = append(params, filterParams...)
		paramID += len(filterParams)
	}
	return strings.Join(fragments, "\n"), params
}

func filterToSQL(
	f api.Filter, values []interface{}, paramID int, fieldMap map[string]string,
) string {
	var field string
	if fm, ok := fieldMap[f.Field]; ok {
		field = fm
	} else {
		field = f.Field
	}
	switch f.Operation {
	case api.FilterOperationIn:
		var fragment strings.Builder
		_, _ = fragment.WriteString("AND %s IN (")
		var paramFragments []string
		for i := range values {
			paramFragments = append(paramFragments, fmt.Sprintf("$%d", paramID+i))
		}
		_, _ = fragment.WriteString(strings.Join(paramFragments, ","))
		_, _ = fragment.WriteString(")")
		return fmt.Sprintf(fragment.String(), field)
	case api.FilterOperationInOrNull:
		var fragment strings.Builder
		_, _ = fragment.WriteString("AND %s IS NULL OR %s IN (")
		var paramFragments []string
		for i := range values {
			paramFragments = append(paramFragments, fmt.Sprintf("$%d", paramID+i))
		}
		_, _ = fragment.WriteString(strings.Join(paramFragments, ","))
		_, _ = fragment.WriteString(")")
		return fmt.Sprintf(fragment.String(), field, field)
	case api.FilterOperationGreaterThan:
		return fmt.Sprintf("AND %s > $%d", field, paramID)
	case api.FilterOperationLessThanEqual:
		return fmt.Sprintf("AND %s <= $%d", field, paramID)
	case api.FilterOperationStringContainment:
		// Works for both bytea and text fields
		return fmt.Sprintf("AND encode(%s::bytea, 'escape') ILIKE  ('%%%%' || $%d || '%%%%')",
			field,
			paramID)
	default:
		panic(fmt.Sprintf("cannot convert operation %d to SQL", f.Operation))
	}
}

func filterToParams(f api.Filter) []interface{} {
	var params []interface{}
	switch vs := f.Values.(type) {
	case []string:
		for _, v := range vs {
			params = append(params, v)
		}
	case string:
		params = append(params, vs)
	case []int64:
		for _, v := range vs {
			params = append(params, v)
		}
	case []int32:
		for _, v := range vs {
			params = append(params, v)
		}
	case time.Time:
		params = append(params, vs)
	default:
		panic(fmt.Sprintf("cannot convert filter values to params: %T", f.Values))
	}
	return params
}

// OrderByToSQL computes the SQL keyword corresponding to the given ordering type.
func OrderByToSQL(order apiv1.OrderBy) string {
	switch order {
	case apiv1.OrderBy_ORDER_BY_UNSPECIFIED:
		return asc
	case apiv1.OrderBy_ORDER_BY_ASC:
		return asc
	case apiv1.OrderBy_ORDER_BY_DESC:
		return desc
	default:
		panic(fmt.Sprintf("unexpected order by: %s", order))
	}
}
