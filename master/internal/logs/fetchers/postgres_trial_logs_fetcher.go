package fetchers

import (
	"fmt"
	"github.com/golang/protobuf/ptypes"
	"strconv"
	"strings"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/logs"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/filters"
)

const (
	batchSize = 1000
)

// PostgresTrialLogsFetcher is a fetcher for postgres-backed trial logs.
type PostgresTrialLogsFetcher struct {
	db      *db.PgDB
	trialID int
	offset  int
	filters []*filters.Filter
}

// NewPostgresTrialLogsFetcher returns a new PostgresTrialLogsFetcher.
func NewPostgresTrialLogsFetcher(
	db *db.PgDB, trialID, offset int, fs []*filters.Filter,
) (*PostgresTrialLogsFetcher, error) {
	fetcher, err := validateFilters(fs)
	if err != nil {
		return fetcher, err
	}
	return &PostgresTrialLogsFetcher{
		db:      db,
		trialID: trialID,
		offset:  offset,
		filters: fs,
	}, nil
}

// validateFilters tries to construct filters using filterToPostgres to ensure validation
// is one to one with functionality, which avoids validations allowing invalid filters and vice versa.
func validateFilters(fs []*filters.Filter) (*PostgresTrialLogsFetcher, error) {
	for _, f := range fs {
		if _, _, err := filterToPostgres(f, 0); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

// Fetch implements logs.Fetcher
func (p *PostgresTrialLogsFetcher) Fetch(limit int, unlimited bool) (logs.Batch, error) {
	switch {
	case unlimited || limit > batchSize:
		limit = batchSize
	case limit <= 0:
		return nil, nil
	}

	parameters := []interface{}{p.trialID, p.offset, limit}
	parameterID := len(parameters) + 1
	var queryFilters []string
	for _, f := range p.filters {
		filterFragment, filterParameter, err := filterToPostgres(f, parameterID)
		if err != nil {
			return nil, err
		}
		queryFilters = append(queryFilters, filterFragment)
		parameters = append(parameters, filterParameter)
		parameterID += 1
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
	err := p.db.QueryStr(query, &b, parameters...)

	if len(b) != 0 {
		p.offset = b[len(b)-1].ID
	}

	return model.TrialLogBatch(b), err
}

func filterToPostgres(f *filters.Filter, paramID int) (string, interface{}, error) {
	switch f.Field {
	case "agent_id", "container_id", "level", "std_type", "source":
		stringValues, ok := f.Values.(*filters.Filter_StringValues)
		if !ok {
			return "", "", fmt.Errorf(
				"unsupported values %T for filter %s", f.Values, f.Field)
		}
		values := stringValues.StringValues.Values
		if len(values) < 1 {
			return "", nil, fmt.Errorf("operation %s in filter %s requires arguments", f.Operation, f.Field)
		}
		switch f.Operation {
		case filters.Filter_OPERATION_IN, filters.Filter_OPERATION_NOT_IN:
			return fmt.Sprintf("AND l.%s %s (SELECT unnest($%d::text[])::text)",
				f.Field, operationToPostgres(f.Operation), paramID), strValuesToSQL(values), nil
		case filters.Filter_OPERATION_EQUAL, filters.Filter_OPERATION_NOT_EQUAL:
			if len(values) > 1 {
				return "", nil, fmt.Errorf("filter %s didn't expect multiple values", f.Field)
			}
			values := stringValues.StringValues.Values
			return fmt.Sprintf("AND l.%s %s $%d",
				f.Field, operationToPostgres(f.Operation), paramID), values[0], nil
		default:
			return "", nil, fmt.Errorf("unsupported operation %s in filter %s", f.Operation, f.Field)
		}
	case "rank_id":
		stringValues, ok := f.Values.(*filters.Filter_IntValues)
		if !ok {
			return "", "", fmt.Errorf(
				"unsupported values %T for filter %s", f.Values, f.Field)
		}
		values := stringValues.IntValues.Values
		if len(values) < 1 {
			return "", nil, fmt.Errorf("operation %s in filter %s requires arguments", f.Operation, f.Field)
		}
		switch f.Operation {
		case filters.Filter_OPERATION_IN, filters.Filter_OPERATION_NOT_IN:
			return fmt.Sprintf("AND l.%s %s (SELECT unnest($%d::smallint[])::smallint)",
				f.Field, operationToPostgres(f.Operation), paramID), intValuesToSQL(values), nil
		case filters.Filter_OPERATION_EQUAL, filters.Filter_OPERATION_NOT_EQUAL:
			if len(values) > 1 {
				return "", nil, fmt.Errorf("filter %s didn't expect multiple values", f.Field)
			}
			return fmt.Sprintf("AND l.%s %s $%d::smallint",
				f.Field, operationToPostgres(f.Operation), paramID), values[0], nil
		default:
			return "", "", fmt.Errorf("unsupported operation %s in filter %s", f.Operation, f.Field)
		}
	case "timestamp":
		stringValues, ok := f.Values.(*filters.Filter_TimestampValues)
		if !ok {
			return "", "", fmt.Errorf(
				"unsupported values %T for filter %s", f.Values, f.Field)
		}
		values := stringValues.TimestampValues.Values
		if len(values) < 1 {
			return "", "", fmt.Errorf("operation %s in filter %s requires arguments", f.Operation, f.Field)
		}
		switch f.Operation {
		case filters.Filter_OPERATION_LESS_EQUAL, filters.Filter_OPERATION_LESS, filters.Filter_OPERATION_GREATER,
			filters.Filter_OPERATION_GREATER_EQUAL:
			if len(values) > 1 {
				return "", nil, fmt.Errorf("filter %s didn't expect multiple values", f.Field)
			}
			value, err := ptypes.Timestamp(values[0])
			if err != nil {
				return "", nil, fmt.Errorf("could not convert timestamp: %w", err)
			}
			return fmt.Sprintf("AND l.%s %s $%d::timestamp",
				f.Field, operationToPostgres(f.Operation), paramID), value, nil
		default:
			return "", nil, fmt.Errorf("unsupported operation %s in filter %s", f.Operation, f.Field)
		}
	default:
		return "", nil, fmt.Errorf("unsupported field in filter %s", f.Field)
	}
}

func strValuesToSQL(vals []string) string {
	return "{" + strings.Join(vals, ",") + "}"
}

func intValuesToSQL(vals []int32) string {
	var strVals []string
	for _, val := range vals {
		strVals = append(strVals, strconv.Itoa(int(val)))
	}
	return "{" + strings.Join(strVals, ",") + "}"
}

func operationToPostgres(op filters.Filter_Operation) string {
	switch op {
	case filters.Filter_OPERATION_EQUAL:
		return "="
	case filters.Filter_OPERATION_NOT_EQUAL:
		return "!="
	case filters.Filter_OPERATION_LESS:
		return "<"
	case filters.Filter_OPERATION_LESS_EQUAL:
		return "<="
	case filters.Filter_OPERATION_GREATER_EQUAL:
		return ">="
	case filters.Filter_OPERATION_GREATER:
		return ">"
	case filters.Filter_OPERATION_IN:
		return "IN"
	case filters.Filter_OPERATION_NOT_IN:
		return "NOT IN"
	default:
		panic(fmt.Sprintf("invalid operation: %s", op))
	}
}
