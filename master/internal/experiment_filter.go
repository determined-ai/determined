package internal

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/uptrace/bun"

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

	metricGroupValidation string = "validation_metrics"
	metricGroupTraining   string = "avg_metrics"
	metricIDTraining      string = "training"
	metricIDValidation    string = "validation"
)

var metricIDTemplate = regexp.MustCompile(
	`(?P<group>[[:print:]]+?)\.(?P<name>[[:print:]]+)\.(?P<qualifier>min|max|mean|last)`)

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
		"progress":        "ROUND(COALESCE(progress, 0) * 100)::INTEGER", // multiply by 100 for percent
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
		"externalExperimentId": "e.external_experiment_id",
		"externalTrialId":      "r.external_run_id",
	}
	var exists bool
	col, exists := filterExperimentColMap[columnName]
	if !exists {
		return "", fmt.Errorf("invalid experiment column %s", columnName)
	}
	return col, nil
}

func runColumnNameToSQL(columnName string) (string, error) {
	// To prevent SQL injection this function should never
	// return a user generated field name

	filterExperimentColMap := map[string]string{
		"id":                    "r.id",
		"experimentDescription": "e.config->>'description'",
		"experimentName":        "e.config->>'name'",
		"tags":                  "e.config->>'labels'",
		"searcherType":          "e.config->'searcher'->>'name'",
		"searcherMetric":        "e.config->'searcher'->>'metric'",
		"startTime":             "r.start_time",
		"endTime":               "r.end_time",
		"duration":              "extract(epoch FROM coalesce(r.end_time, now()) - r.start_time)",
		"state":                 "r.state",
		"experimentProgress":    "ROUND(COALESCE(progress, 0) * 100)::INTEGER", // multiply by 100 for percent
		"user":                  "e.owner_id",
		"forkedFrom":            "e.parent_id",
		"resourcePool":          "e.config->'resources'->>'resource_pool'",
		"projectId":             "r.project_id",
		"checkpointSize":        "e.checkpoint_size",
		"checkpointCount":       "e.checkpoint_count",
		"searcherMetricsVal":    "r.searcher_metric_value",
		"externalExperimentId":  "e.external_experiment_id",
		"externalRunId":         "r.external_run_id",
		"experimentId":          "e.id",
		"localId":               "CONCAT(p.key, '-' , r.local_id::text)",
		"isExpMultitrial":       "e.config->'searcher'->>'name' != 'single'",
		"parentArchived":        "(w.archived OR p.archived)",
	}
	var exists bool
	col, exists := filterExperimentColMap[columnName]
	if !exists {
		return "", fmt.Errorf("invalid run column %s", columnName)
	}
	return col, nil
}

func runHpToSQL(c string, filterColumnType *string, filterValue *interface{},
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
	var queryArgs []interface{}
	runHparam := strings.TrimPrefix(c, "hp.")
	oSQL, err := o.toSQL()
	if err != nil {
		return nil, err
	}
	var queryString string
	switch o {
	case empty:
		queryString = fmt.Sprintf(`r.id NOT IN (SELECT run_id FROM run_hparams WHERE hparam='%s')`, runHparam)
	case notEmpty:
		queryString = fmt.Sprintf(`r.id IN (SELECT run_id FROM run_hparams WHERE hparam='%s')`, runHparam)
	case contains:
		queryArgs = append(queryArgs, queryValue)
		switch queryColumnType {
		case projectv1.ColumnType_COLUMN_TYPE_NUMBER.String():
			queryString = fmt.Sprintf(`r.id IN (SELECT run_id FROM run_hparams WHERE hparam='%s' AND number_val=%s)`,
				runHparam, "?")
		case projectv1.ColumnType_COLUMN_TYPE_TEXT.String():
			queryString = fmt.Sprintf(`r.id IN (SELECT run_id FROM run_hparams WHERE hparam='%s' AND text_val LIKE %s)`,
				runHparam, "?")
		default:
			queryString = fmt.Sprintf(`r.id IN (SELECT run_id FROM run_hparams WHERE hparam='%s' AND bool_val=%s)`,
				runHparam, "?")
		}
	case doesNotContain:
		queryArgs = append(queryArgs, queryValue)
		switch queryColumnType {
		case projectv1.ColumnType_COLUMN_TYPE_NUMBER.String():
			queryString = fmt.Sprintf(`r.id IN (SELECT run_id FROM run_hparams WHERE hparam='%s' AND number_val!=%s)`,
				runHparam, "?")
		case projectv1.ColumnType_COLUMN_TYPE_TEXT.String():
			queryString = fmt.Sprintf(`r.id IN (SELECT run_id FROM run_hparams WHERE hparam='%s' AND text_val NOT LIKE %s)`,
				runHparam, "?")
		default:
			queryString = fmt.Sprintf(`r.id IN (SELECT run_id FROM run_hparams WHERE hparam='%s' AND bool_val!=%s)`,
				runHparam, "?")
		}
	default:
		queryArgs = append(queryArgs, bun.Safe(oSQL), queryValue)
		switch queryColumnType {
		case projectv1.ColumnType_COLUMN_TYPE_NUMBER.String():
			queryString = fmt.Sprintf(`r.id IN (SELECT run_id FROM run_hparams WHERE hparam='%s' AND number_val %s %s)`,
				runHparam, "?", "?")
		case projectv1.ColumnType_COLUMN_TYPE_TEXT.String():
			queryString = fmt.Sprintf(`r.id IN (SELECT run_id FROM run_hparams WHERE hparam='%s' AND text_val %s %s)`,
				runHparam, "?", "?")
		default:
			queryString = fmt.Sprintf(`r.id IN (SELECT run_id FROM run_hparams WHERE hparam='%s' AND bool_val %s %s)`,
				runHparam, "?", "?")
		}
	}

	if fc != nil && *fc == or {
		return q.WhereOr(queryString, queryArgs...), nil
	}
	return q.Where(queryString, queryArgs...), nil
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
				WHEN config->'hyperparameters'->%s->>'type' = 'categorical' THEN (config->'hyperparameters'->%s->>'vals')::jsonb %s %s IS NOT TRUE
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

func expRunOperatorQuery(o operator, col string, oSQL string, val *interface{}) (string, []interface{}) {
	var queryArgs []interface{}
	var queryString string
	switch o {
	case contains:
		queryString = "? ILIKE ?"
		queryArgs = append(queryArgs, bun.Safe(col), fmt.Sprintf("%%%s%%", *val))
	case doesNotContain:
		queryString = "? NOT ILIKE ?"
		queryArgs = append(queryArgs, bun.Safe(col), fmt.Sprintf("%%%s%%", *val))
	case empty:
		queryString = "? IS NULL OR ? = '' OR ? = '[]'"
		queryArgs = append(queryArgs, bun.Safe(col), bun.Safe(col), bun.Safe(col))
	case notEmpty:
		queryString = "? IS NOT NULL AND ? != '' AND ? != '[]'"
		queryArgs = append(queryArgs, bun.Safe(col), bun.Safe(col), bun.Safe(col))
	default:
		queryArgs = append(queryArgs, bun.Safe(col),
			bun.Safe(oSQL), *val)
		queryString = "? ? ?"
	}
	return queryString, queryArgs
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
			var col string
			col, err = expColumnNameToSQL(e.ColumnName)
			if err != nil {
				return nil, err
			}
			queryString, queryArgs := expRunOperatorQuery(*e.Operator, col, oSQL, e.Value)
			if c != nil && *c == or {
				q.WhereOr(queryString, queryArgs...)
			} else {
				q.Where(queryString, queryArgs...)
			}
			if err != nil {
				return nil, err
			}
		case projectv1.LocationType_LOCATION_TYPE_RUN.String():
			var col string
			col, err = runColumnNameToSQL(e.ColumnName)
			if err != nil {
				return nil, err
			}
			queryString, queryArgs := expRunOperatorQuery(*e.Operator, col, oSQL, e.Value)
			if c != nil && *c == or {
				q.WhereOr(queryString, queryArgs...)
			} else {
				q.Where(queryString, queryArgs...)
			}
			if err != nil {
				return nil, err
			}
		case projectv1.LocationType_LOCATION_TYPE_VALIDATIONS.String(),
			projectv1.LocationType_LOCATION_TYPE_TRAINING.String(),
			projectv1.LocationType_LOCATION_TYPE_CUSTOM_METRIC.String():
			queryColumnType := projectv1.ColumnType_COLUMN_TYPE_UNSPECIFIED.String()
			if e.Type != nil {
				queryColumnType = *e.Type
			}
			metricGroup, metricName, metricQualifier, err := parseMetricsName(e.ColumnName)
			if err != nil {
				return nil, err
			}
			var col string
			var queryArgs []interface{}
			var queryString string
			col = `r.summary_metrics->?->?->>?`
			queryArgs = append(queryArgs, metricGroup, metricName, metricQualifier)
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
		case projectv1.LocationType_LOCATION_TYPE_RUN_HYPERPARAMETERS.String():
			return runHpToSQL(e.ColumnName, e.Type, e.Value, e.Operator, q, c)
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

// training.loss.min -> avg_metrics, loss, min
// group_a.value.last -> group_a, value, last
// group_b.value.a.last -> group_b, value.a, last .
func parseMetricsName(str string) (metricGroup string, metricName string, metricQualifier string, err error) {
	matches := metricIDTemplate.FindStringSubmatch(str)
	if len(matches) < 4 {
		return "", "", "", fmt.Errorf("%s is not a valid metrics id", str)
	}

	metricGroup = matches[1]
	metricName = matches[2]
	metricQualifier = matches[3]

	if metricGroup == metricIDTraining {
		metricGroup = metricGroupTraining
	}
	if metricGroup == metricIDValidation {
		metricGroup = metricGroupValidation
	}

	return metricGroup, metricName, metricQualifier, nil
}
