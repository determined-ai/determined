package db

import (
	"fmt"
	"strings"

	"github.com/golang/protobuf/ptypes"

	"github.com/determined-ai/determined/master/internal/api"

	"github.com/determined-ai/determined/master/pkg/model"
	filters "github.com/determined-ai/determined/proto/pkg/filtersv1"
)

const (
	batchSize = 1000
)

// TrialLogs takes a trial ID and log offset, limit and filters and returns matching trial logs.
func (db *PgDB) TrialLogs(
	trialID, offset, limit int, fs []*filters.Filter,
) ([]*model.TrialLog, error) {
	parameters := []interface{}{trialID, offset, limit}
	parameterID := len(parameters) + 1
	var queryFilters []string
	for _, f := range fs {
		fragment, params, err := trialLogsFiltersToSQL(f, parameterID)
		if err != nil {
			return nil, err
		}
		queryFilters = append(queryFilters, fragment)
		parameters = append(parameters, params...)
		parameterID += len(params)
	}
	query := fmt.Sprintf(`
SELECT
    l.id,
    l.trial_id,
    encode(l.message, 'escape') as message,
    l.agent_id,
    l.container_id,
    l.rank_id,
    l.timestamp,
    l.level,
    l.std_type,
    l.source
FROM trial_logs l
WHERE l.trial_id = $1
%s
ORDER BY l.id ASC OFFSET $2 LIMIT $3
`, strings.Join(queryFilters, "\n"))

	var b []*model.TrialLog
	return b, db.queryRows(query, &b, parameters...)
}

// trialLogsFiltersToSQL takes a filter, the type for the field being filtered and the
// current paramID for the query it is being built into and returns a query fragment,
// the parameters for that fragment of the query and an error if it failed.
func trialLogsFiltersToSQL(f *filters.Filter, paramID int) (string, []interface{}, error) {
	switch f.Field {
	case "agent_id", "container_id", "level", "std_type", "source":
		return stringFilterToSQL(paramID, f)
	case "rank_id":
		return intFilterToSQL(f, paramID)
	case "timestamp":
		return timestampFilterToSQL(f, paramID)
	default:
		return "", nil, api.ErrUnsupportedFilterField{Filter: f}
	}
}

func stringFilterToSQL(paramID int, f *filters.Filter) (string, []interface{}, error) {
	stringValues, ok := f.Values.(*filters.Filter_StringValues)
	if !ok {
		return "", nil, api.ErrUnsupportedFilterValues{Filter: f}
	}
	values := stringValues.StringValues.Values
	if len(values) < 1 {
		return "", nil, api.ErrMissingFilterValues{Filter: f}
	}
	var params []interface{}
	for _, v := range values {
		params = append(params, v)
	}
	return filterToSQL(f, paramID, params)
}

func intFilterToSQL(f *filters.Filter, paramID int) (string, []interface{}, error) {
	intValues, ok := f.Values.(*filters.Filter_IntValues)
	if !ok {
		return "", nil, api.ErrUnsupportedFilterValues{Filter: f}
	}
	values := intValues.IntValues.Values
	var params []interface{}
	for _, v := range values {
		params = append(params, v)
	}
	return filterToSQL(f, paramID, params)
}

func timestampFilterToSQL(f *filters.Filter, paramID int) (string, []interface{}, error) {
	timestampValues, ok := f.Values.(*filters.Filter_TimestampValues)
	if !ok {
		return "", nil, api.ErrUnsupportedFilterValues{Filter: f}
	}
	values := timestampValues.TimestampValues.Values
	var params []interface{}
	for _, v := range values {
		tv, err := ptypes.Timestamp(v)
		if err != nil {
			return "", nil, fmt.Errorf("%s: %w", err.Error(), api.ErrUnsupportedFilter)
		}
		params = append(params, tv)
	}
	return filterToSQL(f, paramID, params)
}

func filterToSQL(
	f *filters.Filter, paramID int, params []interface{},
) (string, []interface{}, error) {
	if len(params) < 1 {
		return "", nil, api.ErrMissingFilterValues{Filter: f}
	}
	switch f.Operation {
	case filters.Filter_OPERATION_IN, filters.Filter_OPERATION_NOT_IN:
		return setOpFilterToSQL(f, paramID, params)
	case filters.Filter_OPERATION_EQUAL, filters.Filter_OPERATION_NOT_EQUAL,
		filters.Filter_OPERATION_GREATER, filters.Filter_OPERATION_LESS,
		filters.Filter_OPERATION_LESS_EQUAL, filters.Filter_OPERATION_GREATER_EQUAL:
		return binaryOpFilterToSQL(f, paramID, params)
	default:
		return "", nil, api.ErrUnsupportedFilterOperation{Filter: f}
	}
}

func setOpFilterToSQL(
	f *filters.Filter, paramID int, params []interface{},
) (string, []interface{}, error) {
	var fragment strings.Builder
	var paramFragments []string
	_, _ = fragment.WriteString("AND %s %s (")
	for i := range params {
		paramFragments = append(paramFragments, fmt.Sprintf("$%d", paramID+i))
	}
	_, _ = fragment.WriteString(strings.Join(paramFragments, ","))
	_, _ = fragment.WriteString(")")
	return fmt.Sprintf(fragment.String(), f.Field, setOpToSQL(f.Operation)), params, nil
}

func binaryOpFilterToSQL(
	f *filters.Filter, paramID int, params []interface{},
) (string, []interface{}, error) {
	if len(params) > 1 {
		return "", nil, api.ErrTooManyFilterValues{Filter: f}
	}
	return fmt.Sprintf("AND %s %s $%d", f.Field, binaryOpToSQL(f.Operation), paramID), params, nil
}

func binaryOpToSQL(op filters.Filter_Operation) string {
	switch op {
	case filters.Filter_OPERATION_EQUAL:
		return "="
	case filters.Filter_OPERATION_NOT_EQUAL:
		return "!="
	case filters.Filter_OPERATION_LESS:
		return "<"
	case filters.Filter_OPERATION_LESS_EQUAL:
		return "<="
	case filters.Filter_OPERATION_GREATER:
		return ">"
	case filters.Filter_OPERATION_GREATER_EQUAL:
		return ">="
	default:
		panic(fmt.Sprintf("invalid binary operation: %s", op))
	}
}

func setOpToSQL(op filters.Filter_Operation) string {
	switch op {
	case filters.Filter_OPERATION_IN:
		return "IN"
	case filters.Filter_OPERATION_NOT_IN:
		return "NOT IN"
	default:
		panic(fmt.Sprintf("invalid set operation: %s", op))
	}
}

// TrialLogsFetcher is a fetcher for postgres-backed trial logs.
type TrialLogsFetcher struct {
	db      *PgDB
	trialID int
	offset  int
	filters []*filters.Filter
}

// NewTrialLogsFetcher returns a new TrialLogsFetcher.
func NewTrialLogsFetcher(
	db *PgDB, trialID, offset int, fs []*filters.Filter,
) (*TrialLogsFetcher, error) {
	fetcher, err := validateTrialLogsFilters(fs)
	if err != nil {
		return fetcher, err
	}
	return &TrialLogsFetcher{
		db:      db,
		trialID: trialID,
		offset:  offset,
		filters: fs,
	}, nil
}

// validateTrialLogsFilters tries to construct filters using trialLogsFiltersToSQL to ensure
// validation is one to one with functionality, which avoids validations allowing invalid filters
// and vice versa.
func validateTrialLogsFilters(fs []*filters.Filter) (*TrialLogsFetcher, error) {
	for _, f := range fs {
		if _, _, err := trialLogsFiltersToSQL(f, 0); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

// Fetch implements api.Fetcher
func (f *TrialLogsFetcher) Fetch(limit int, unlimited bool) (api.Batch, error) {
	switch {
	case unlimited || limit > batchSize:
		limit = batchSize
	case limit <= 0:
		return nil, nil
	}

	b, err := f.db.TrialLogs(f.trialID, f.offset, limit, f.filters)
	if err != nil {
		return nil, err
	}

	if len(b) != 0 {
		f.offset = b[len(b)-1].ID
	}

	return model.TrialLogBatch(b), err
}
