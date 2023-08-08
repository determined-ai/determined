package internal

import (
	"fmt"
	"strings"

	"github.com/uptrace/bun"

	"golang.org/x/exp/slices"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/proto/pkg/projectv1"
)

const (
	and                filterConjunction = "and"
	or                 filterConjunction = "or"
	field              filterType        = "field"
	group              filterType        = "group"
	equal              operator          = "="
	notEqual           operator          = "!="
	lessThan           operator          = "<"
	lessThanOrEqual    operator          = "<="
	greaterThan        operator          = ">"
	greaterThanOrEqual operator          = ">="
	contains           operator          = "contains"
	doesNotContain     operator          = "notContains"
	empty              operator          = "isEmpty"
	notEmpty           operator          = "notEmpty"
)

type (
	filterConjunction string
	filterType        string
	operator          string
)

type experimentFilter struct {
	Children    []*experimentFilter
	Conjunction *filterConjunction
	Operator    *operator
	Value       *interface{}
	Kind        filterType
	ColumnName  string
	Location    *string
	Type        *string
}

type experimentFilterRoot struct {
	FilterGroup  experimentFilter
	ShowArchived bool
}

func (o *operator) toSQL() (string, error) {
	var s string
	switch *o {
	case equal:
		s = "="
	case notEqual:
		s = "!="
	case lessThan:
		s = "<"
	case lessThanOrEqual:
		s = "<="
	case greaterThan:
		s = ">"
	case greaterThanOrEqual:
		s = ">="
	case empty:
		s = "IS NULL"
	case notEmpty:
		s = "IS NOT NULL"
	case contains:
		return s, nil
	case doesNotContain:
		return s, nil
	default:
		return "", fmt.Errorf("invalid operator %v", *o)
	}
	return s, nil
}

func expColumnNameToSQL(columnName string) (string, error) {
	// To prevent SQL injection this function should never
	// return a user generated field name

	filterExperimentColMap := map[string]string{
		"id":              "e.id",
		"description":     "e.config->>'description'",
		"name":            "e.config->>'name'",
		"tags":            "e.config->>'labels'",
		"searcherType":    "e.config->'searcher'->>'name'",
		"searcherMetric":  "e.config->'searcher'->>'metric'",
		"startTime":       "e.start_time",
		"endTime":         "e.end_time",
		"duration":        "extract(epoch FROM coalesce(e.end_time, now()) - e.start_time)",
		"state":           "e.state",
		"numTrials":       "(SELECT COUNT(*) FROM trials t WHERE e.id = t.experiment_id)",
		"progress":        "COALESCE(progress, 0)",
		"user":            "e.owner_id",
		"forkedFrom":      "e.parent_id",
		"resourcePool":    "e.config->'resources'->>'resource_pool'",
		"projectId":       "project_id",
		"checkpointSize":  "e.checkpoint_size",
		"checkpointCount": "e.checkpoint_count",
		"searcherMetricsVal": `(
			SELECT
				searcher_metric_value
			FROM trials t
			WHERE t.experiment_id = e.id
			ORDER BY (CASE
				WHEN coalesce((config->'searcher'->>'smaller_is_better')::boolean, true)
					THEN searcher_metric_value
					ELSE -1.0 * searcher_metric_value
			END) ASC
			LIMIT 1
		) `,
	}
	var exists bool
	col, exists := filterExperimentColMap[columnName]
	if !exists {
		return "", fmt.Errorf("invalid experiment column %s", columnName)
	}
	return col, nil
}

// nolint: lll
func hpToSQL(c string, filterColumnType *string, filterValue *interface{},
	op *operator, q *bun.SelectQuery,
	fc *filterConjunction,
) (*bun.SelectQuery, error) {
	queryColumnType := projectv1.ColumnType_COLUMN_TYPE_UNSPECIFIED.String()
	var o operator
	var queryValue interface{}
	if filterValue == nil && op != nil && *op != empty && *op != notEmpty {
		return nil, fmt.Errorf("hyperparameter field defined without value and without a valid operator")
	}
	o = *op
	if o != empty && o != notEmpty {
		queryValue = *filterValue
	}
	if filterColumnType != nil {
		queryColumnType = *filterColumnType
	}
	var hps []string
	hp := strings.Split(strings.TrimPrefix(c, "hp."), ".")
	for len(hps) < len(hp) {
		hps = append(hps, `?`)
	}
	hpQuery := strings.Join(hps, "->")
	oSQL, err := o.toSQL()
	if err != nil {
		return nil, err
	}
	var queryArgs []interface{}
	var queryString string
	switch queryColumnType {
	case projectv1.ColumnType_COLUMN_TYPE_TEXT.String(), projectv1.ColumnType_COLUMN_TYPE_DATE.String():
		switch o {
		case empty, notEmpty:
			for i := 0; i < 2; i++ {
				for j := 0; j < 2; j++ {
					for _, hp := range hp {
						queryArgs = append(queryArgs, hp)
					}
				}
				queryArgs = append(queryArgs, bun.Safe(oSQL))
			}
			queryString = fmt.Sprintf(`(CASE
				WHEN config->'hyperparameters'->%s->>'type' = 'const' THEN config->'hyperparameters'->%s->>'val' %s
				WHEN config->'hyperparameters'->%s->>'type' = 'categorical' THEN config->'hyperparameters'->%s->>'vals' %s
				ELSE false
			 END)`, hpQuery, hpQuery, "?", hpQuery, hpQuery, "?")
		case contains:
			queryLikeValue := `%` + queryValue.(string) + `%`
			for i := 0; i < 2; i++ {
				for _, hp := range hp {
					queryArgs = append(queryArgs, hp)
				}
			}
			queryArgs = append(queryArgs, queryLikeValue)
			for i := 0; i < 2; i++ {
				for _, hp := range hp {
					queryArgs = append(queryArgs, hp)
				}
			}
			queryArgs = append(queryArgs, bun.Safe("?"), queryValue)
			queryString = fmt.Sprintf(`(CASE
				WHEN config->'hyperparameters'->%s->>'type' = 'const' THEN config->'hyperparameters'->%s->>'val' LIKE %s
				WHEN config->'hyperparameters'->%s->>'type' = 'categorical' THEN (config->'hyperparameters'->%s->>'vals')::jsonb %s %s
				ELSE false
			 END)`, hpQuery, hpQuery, "?", hpQuery, hpQuery, "?", "?")
		case doesNotContain:
			queryLikeValue := `%` + queryValue.(string) + `%`
			for i := 0; i < 2; i++ {
				for _, hp := range hp {
					queryArgs = append(queryArgs, hp)
				}
			}
			queryArgs = append(queryArgs, queryLikeValue)
			for i := 0; i < 2; i++ {
				for _, hp := range hp {
					queryArgs = append(queryArgs, hp)
				}
			}
			queryArgs = append(queryArgs, bun.Safe("?"), queryValue)
			queryString = fmt.Sprintf(`(CASE
				WHEN config->'hyperparameters'->%s->>'type' = 'const' THEN config->'hyperparameters'->%s->>'val' NOT LIKE %s
				WHEN config->'hyperparameters'->%s->>'type' = 'categorical' THEN (config->'hyperparameters'->%s->>'vals')::jsonb %s %s) IS NOT TRUE
				ELSE false
			 END)`, hpQuery, hpQuery, "?", hpQuery, hpQuery, "?", "?")
		default:
			for i := 0; i < 2; i++ {
				for _, hp := range hp {
					queryArgs = append(queryArgs, hp)
				}
			}
			queryArgs = append(queryArgs, bun.Safe(oSQL), queryValue)
			queryString = fmt.Sprintf(`(CASE WHEN config->'hyperparameters'->%s->>'type' = 'const' THEN config->'hyperparameters'->%s->>'val' %s %s ELSE false END)`,
				hpQuery, hpQuery, "?", "?")
		}
	default:
		switch o {
		case empty, notEmpty:
			for i := 0; i < 3; i++ {
				for j := 0; j < 2; j++ {
					for _, hp := range hp {
						queryArgs = append(queryArgs, hp)
					}
				}
				queryArgs = append(queryArgs, bun.Safe(oSQL))
			}
			queryString = fmt.Sprintf(`(CASE
				WHEN config->'hyperparameters'->%s->>'type' = 'const' THEN (config->'hyperparameters'->%s->>'val')::float8 %s
				WHEN config->'hyperparameters'->%s->>'type' = 'categorical' THEN config->'hyperparameters'->%s->>'vals' %s
				WHEN config->'hyperparameters'->%s->>'type' IN ('int', 'double', 'log') THEN (config->'hyperparameters'->%s) %s
				ELSE false
			 END)`, hpQuery, hpQuery, "?", hpQuery, hpQuery, "?", hpQuery, hpQuery, "?")
		case contains:
			for i := 0; i < 2; i++ {
				for _, hp := range hp {
					queryArgs = append(queryArgs, hp)
				}
			}
			queryArgs = append(queryArgs, bun.Safe(`?`), queryValue)
			for i := 0; i < 2; i++ {
				for _, hp := range hp {
					queryArgs = append(queryArgs, hp)
				}
			}
			queryArgs = append(queryArgs, queryValue)
			for _, hp := range hp {
				queryArgs = append(queryArgs, hp)
			}
			queryArgs = append(queryArgs, queryValue)
			queryString = fmt.Sprintf(`(CASE
					WHEN config->'hyperparameters'->%s->>'type' = 'categorical' THEN (config->'hyperparameters'->%s->>'vals')::jsonb %s '%s'
					WHEN config->'hyperparameters'->%s->>'type' IN ('int', 'double', 'log') THEN (config->'hyperparameters'->%s->>'minval')::float8 <= %s OR (config->'hyperparameters'->%s->>'maxval')::float8 >= %s
					ELSE false
				 END)`, hpQuery, hpQuery, "?", "?", hpQuery, hpQuery, "?", hpQuery, "?")
		case doesNotContain:
			for i := 0; i < 2; i++ {
				for _, hp := range hp {
					queryArgs = append(queryArgs, hp)
				}
			}
			queryArgs = append(queryArgs, bun.Safe(`?`), queryValue)
			for i := 0; i < 2; i++ {
				for _, hp := range hp {
					queryArgs = append(queryArgs, hp)
				}
			}
			queryArgs = append(queryArgs, queryValue)
			for _, hp := range hp {
				queryArgs = append(queryArgs, hp)
			}
			queryArgs = append(queryArgs, queryValue)
			queryString = fmt.Sprintf(`(CASE
					WHEN config->'hyperparameters'->%s->>'type' = 'categorical' THEN ((config->'hyperparameters'->%s->>'vals')::jsonb %s '%s') IS NOT TRUE
					WHEN config->'hyperparameters'->%s->>'type' IN ('int', 'double', 'log') THEN (config->'hyperparameters'->%s->>'minval')::float8 >= %s OR (config->'hyperparameters'->%s->>'maxval')::float8 <= %s
					ELSE false
				 END)`, hpQuery, hpQuery, "?", "?", hpQuery, hpQuery, "?", hpQuery, "?")
		default:
			for i := 0; i < 2; i++ {
				for j := 0; j < 2; j++ {
					for _, hp := range hp {
						queryArgs = append(queryArgs, hp)
					}
				}
				queryArgs = append(queryArgs, bun.Safe(oSQL), queryValue)
			}
			for _, hp := range hp {
				queryArgs = append(queryArgs, hp)
			}
			queryArgs = append(queryArgs, bun.Safe(oSQL),
				queryValue)
			for _, hp := range hp {
				queryArgs = append(queryArgs, hp)
			}
			queryArgs = append(queryArgs, bun.Safe(oSQL), queryValue)
			queryString = fmt.Sprintf(`(CASE
				WHEN config->'hyperparameters'->%s->>'type' = 'const' THEN (config->'hyperparameters'->%s->>'val')::float8 %s %s
				WHEN config->'hyperparameters'->%s->>'type' IN ('int', 'double', 'log') THEN ((config->'hyperparameters'->%s->>'minval')::float8 %s %s OR (config->'hyperparameters'->%s->>'maxval')::float8 %s %s)
				ELSE false
			 END)`, hpQuery, hpQuery, "?", "?", hpQuery, hpQuery, "?", "?", hpQuery, "?", "?")
		}
	}
	if fc != nil && *fc == or {
		return q.WhereOr(queryString, queryArgs...), nil
	}
	return q.Where(queryString, queryArgs...), nil
}

func (e experimentFilterRoot) toSQL(q *bun.SelectQuery) (*bun.SelectQuery, error) {
	q, err := e.FilterGroup.toSQL(q, nil)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (e experimentFilter) toSQL(q *bun.SelectQuery,
	c *filterConjunction,
) (*bun.SelectQuery, error) {
	switch e.Kind {
	case field:
		if e.Operator == nil {
			return nil, fmt.Errorf("field specified with value but no operator")
		}
		if e.Value == nil && *e.Operator != notEmpty && *e.Operator != empty {
			return q.Where("true"), nil //nolint:goconst
		}
		oSQL, err := e.Operator.toSQL()
		if err != nil {
			return nil, err
		}
		location := projectv1.LocationType_LOCATION_TYPE_EXPERIMENT.String()
		if e.Location != nil {
			location = *e.Location
		}
		switch location {
		case projectv1.LocationType_LOCATION_TYPE_EXPERIMENT.String():
			col, err := expColumnNameToSQL(e.ColumnName)
			if err != nil {
				return nil, err
			}
			var queryArgs []interface{}
			var queryString string
			switch *e.Operator {
			case contains:
				queryString = "? ILIKE ?"
				queryArgs = append(queryArgs, bun.Safe(col), fmt.Sprintf("%%%s%%", *e.Value))
			case doesNotContain:
				queryString = "? NOT ILIKE ?"
				queryArgs = append(queryArgs, bun.Safe(col), fmt.Sprintf("%%%s%%", *e.Value))
			case empty, notEmpty:
				queryString = "? ?"
				queryArgs = append(queryArgs, bun.Safe(col), bun.Safe(oSQL))
			default:
				queryArgs = append(queryArgs, bun.Safe(col),
					bun.Safe(oSQL), *e.Value)
				queryString = "? ? ?"
			}
			if c != nil && *c == or {
				q.WhereOr(queryString, queryArgs...)
			} else {
				q.Where(queryString, queryArgs...)
			}
			if err != nil {
				return nil, err
			}
		case projectv1.LocationType_LOCATION_TYPE_VALIDATIONS.String():
			metricName := strings.TrimPrefix(e.ColumnName, "validation.")
			queryColumnType := projectv1.ColumnType_COLUMN_TYPE_UNSPECIFIED.String()
			if e.Type != nil {
				queryColumnType = *e.Type
			}
			col := `e.validation_metrics->>?`
			var queryArgs []interface{}
			var queryString string
			switch queryColumnType {
			case projectv1.ColumnType_COLUMN_TYPE_NUMBER.String():
				col = fmt.Sprintf(`(%v)::float8`, col)
			}
			switch *e.Operator {
			case contains:
				queryArgs = append(queryArgs, metricName, fmt.Sprintf("%%%s%%", *e.Value))
				queryString = fmt.Sprintf("%s LIKE ?", col)
			case doesNotContain:
				queryArgs = append(queryArgs, metricName, fmt.Sprintf("%%%s%%", *e.Value))
				queryString = fmt.Sprintf("%s NOT LIKE ?", col)
			case empty, notEmpty:
				queryArgs = append(queryArgs, metricName, bun.Safe(oSQL))
				queryString = fmt.Sprintf("%s ?", col)
			default:
				queryArgs = append(queryArgs, metricName,
					bun.Safe(oSQL), *e.Value)
				queryString = fmt.Sprintf("%s ? ?", col)
			}
			if c != nil && *c == or {
				q.WhereOr(queryString, queryArgs...)
			} else {
				q.Where(queryString, queryArgs...)
			}
		case projectv1.LocationType_LOCATION_TYPE_TRAINING.String():
			queryColumnType := projectv1.ColumnType_COLUMN_TYPE_UNSPECIFIED.String()
			if e.Type != nil {
				queryColumnType = *e.Type
			}
			metricDetails := strings.Split(e.ColumnName, ".")
			metricQualifier := metricDetails[len(metricDetails)-1]
			metricName := strings.TrimSuffix(
				strings.TrimPrefix(e.ColumnName, "training."),
				"."+metricQualifier)
			if !slices.Contains(SummaryMetricStatistics, metricQualifier) {
				return nil, status.Errorf(codes.InvalidArgument,
					"sort training metrics by statistic: last, max, min, or mean")
			}
			var col string
			var queryArgs []interface{}
			var queryString string
			if metricQualifier == "mean" {
				locator := bun.Safe("trials.summary_metrics->'avg_metrics'")
				col = fmt.Sprintf(`(?->?->>'sum')::float8 / (?->?->>'count')::int`)
				queryArgs = append(queryArgs, locator, metricName, locator, metricName)
			} else {
				col = `trials.summary_metrics->'avg_metrics'->?->>?`
				queryArgs = append(queryArgs, metricName, metricQualifier)
			}
			if queryColumnType == projectv1.ColumnType_COLUMN_TYPE_NUMBER.String() {
				col = fmt.Sprintf(`(%v)::float8`, col)
			}
			switch *e.Operator {
			case contains:
				queryArgs = append(queryArgs, fmt.Sprintf("%%%s%%", *e.Value))
				queryString = fmt.Sprintf("%s LIKE ?", col)
			case doesNotContain:
				queryArgs = append(queryArgs, fmt.Sprintf("%%%s%%", *e.Value))
				queryString = fmt.Sprintf("%s NOT LIKE ?", col)
			case empty, notEmpty:
				queryArgs = append(queryArgs, bun.Safe(oSQL))
				queryString = fmt.Sprintf("%s ?", col)
			default:
				queryArgs = append(queryArgs, bun.Safe(oSQL), *e.Value)
				queryString = fmt.Sprintf("%s ? ?", col)
			}
			if c != nil && *c == or {
				q.WhereOr(queryString, queryArgs...)
			} else {
				q.Where(queryString, queryArgs...)
			}
		case projectv1.LocationType_LOCATION_TYPE_HYPERPARAMETERS.String():
			return hpToSQL(e.ColumnName, e.Type, e.Value, e.Operator, q, c)
		}
	case group:
		var co string
		var err error
		if e.Conjunction == nil {
			return nil, fmt.Errorf("group specified with no conjunction")
		}
		if len(e.Children) == 0 {
			return q.Where("true"), nil //nolint:goconst
		}
		switch *e.Conjunction {
		case and:
			co = " AND "
		case or:
			co = " OR "
		default:
			return nil, fmt.Errorf("invalid conjunction value %v", *e.Conjunction)
		}
		for _, c := range e.Children {
			q = q.WhereGroup(co, func(q *bun.SelectQuery) *bun.SelectQuery {
				_, err = c.toSQL(q, e.Conjunction)
				if err != nil {
					return q
				}
				return q
			})
			if err != nil {
				return q, err
			}
		}
	}
	return q, nil
}
