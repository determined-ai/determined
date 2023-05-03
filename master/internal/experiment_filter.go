package internal

import (
	"fmt"
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
	doesNotContain     operator          = "does not contain"
	empty              operator          = "is empty"
	notEmpty           operator          = "not empty"
	is                 operator          = "is"
	isNot              operator          = "is not"
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
		s = "!=" //nolint: goconst
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
	case is:
		s = "="
	case isNot:
		s = "!="
	case contains:
		return s, nil
	case doesNotContain:
		return s, nil
	default:
		return "", fmt.Errorf("invalid operator %v", *o)
	}
	return s, nil
}

func columnNameToSQL(c string, l *string, t *string) (string, error) {
	locationType := projectv1.LocationType_LOCATION_TYPE_EXPERIMENT.String()
	columnType := projectv1.ColumnType_COLUMN_TYPE_UNSPECIFIED.String()
	var col string
	if l != nil {
		locationType = *l
	}
	if t != nil {
		columnType = *t
	}
	filterExperimentColMap := map[string]string{
		"id":              "e.id",
		"description":     "e.config->>'description'",
		"name":            "e.config->>'name'",
		"tags":            "e.config->>'labels'",
		"startTime":       "e.start_time",
		"endTime":         "e.end_time",
		"duration":        "extract(seconds FROM coalesce(e.end_time, now()) - e.start_time)",
		"state":           "e.state",
		"numTrials":       "(SELECT COUNT(*) FROM trials t WHERE e.id = t.experiment_id)",
		"progress":        "COALESCE(progress, 0)",
		"user":            "COALESCE(u.display_name, u.username)",
		"forkedFrom":      "e.parent_id",
		"resourcePool":    "e.config->'resources'->>'resource_pool'",
		"projectId":       "project_id",
		"checkpointSize":  "checkpoint_size",
		"checkpointCount": "checkpoint_count",
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

	switch locationType {
	case projectv1.LocationType_LOCATION_TYPE_EXPERIMENT.String():
		var exists bool
		col, exists = filterExperimentColMap[c]
		if !exists {
			return "", fmt.Errorf("invalid experiment column %s", col)
		}
		return col, nil
	case projectv1.LocationType_LOCATION_TYPE_VALIDATIONS.String():
		col = fmt.Sprintf(`e.validation_metrics->>'%s'`, strings.TrimPrefix(c, "validation."))
		switch columnType {
		case projectv1.ColumnType_COLUMN_TYPE_NUMBER.String():
			col = fmt.Sprintf(`(%v)::float8`, col)
		}
	default:
		return "", fmt.Errorf("unhandled column location type %v", locationType)
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
			i := 0
			for i < 2 {
				for _, hp := range hp {
					queryArgs = append(queryArgs, hp, hp)
				}
				queryArgs = append(queryArgs, bun.Safe(oSQL))
				i++
			}
			queryString = fmt.Sprintf(`(CASE
				WHEN config->'hyperparameters'->%s->>'type' = 'const' THEN config->'hyperparameters'->%s->>'val' %s
				WHEN config->'hyperparameters'->%s->>'type' = 'categorical' THEN config->'hyperparameters'->%s->>'vals' %s
				ELSE false
			 END)`, hpQuery, hpQuery, "?", hpQuery, hpQuery, "?")
		case contains:
			i := 0
			queryLikeValue := `%` + queryValue.(string) + `%`
			for i < 2 {
				for _, hp := range hp {
					queryArgs = append(queryArgs, hp)
				}
				i++
			}
			queryArgs = append(queryArgs, queryLikeValue)
			i = 0
			for i < 2 {
				for _, hp := range hp {
					queryArgs = append(queryArgs, hp)
				}
				i++
			}
			queryArgs = append(queryArgs, bun.Safe("?"), queryValue)
			queryString = fmt.Sprintf(`(CASE
				WHEN config->'hyperparameters'->%s->>'type' = 'const' THEN config->'hyperparameters'->%s->>'val' LIKE %s
				WHEN config->'hyperparameters'->%s->>'type' = 'categorical' THEN (config->'hyperparameters'->%s->>'vals')::jsonb %s %s
				ELSE false
			 END)`, hpQuery, hpQuery, "?", hpQuery, hpQuery, "?", "?")
		case doesNotContain:
			queryString = `(CASE 
				WHEN config->'hyperparameters'->?->>'type' = 'const' THEN config->'hyperparameters'->?->>'val' ?
				WHEN config->'hyperparameters'->?->>'type' = 'categorical' THEN (config->'hyperparameters'->?->>'vals')::jsonb ?? ? IS NOT TRUE
				ELSE false
			 END)`
			queryArgs = append(queryArgs, hpQuery,
				hpQuery, `NOT LIKE %`+fmt.Sprintf(`%v`, queryValue)+`%`,
				hpQuery, hpQuery, queryValue)
		default:
			i := 0
			for i < 2 {
				for _, hp := range hp {
					queryArgs = append(queryArgs, hp)
				}
				i++
			}
			queryArgs = append(queryArgs, bun.Safe(oSQL), queryValue)
			queryString = fmt.Sprintf(`(CASE WHEN config->'hyperparameters'->%s->>'type' = 'const' THEN config->'hyperparameters'->%s->>'val' %s %s ELSE false END)`,
				hpQuery, hpQuery, "?", "?")
		}
	default:
		switch o {
		case empty, notEmpty:
			i, j := 0, 0
			for i < 3 {
				for j < 2 {
					for _, hp := range hp {
						queryArgs = append(queryArgs, hp)
					}
					j++
				}
				queryArgs = append(queryArgs, bun.Safe(oSQL))
				i++
				j = 0
			}
			queryString = fmt.Sprintf(`(CASE
				WHEN config->'hyperparameters'->%s->>'type' = 'const' THEN (config->'hyperparameters'->%s->>'val')::float8 %s
				WHEN config->'hyperparameters'->%s->>'type' = 'categorical' THEN config->'hyperparameters'->%s->>'vals' %s
				WHEN config->'hyperparameters'->%s->>'type' IN ('int', 'double', 'log') THEN (config->'hyperparameters'->%s) %s
				ELSE false
			 END)`, hpQuery, hpQuery, "?", hpQuery, hpQuery, "?", hpQuery, hpQuery, "?")
		case contains:
			i := 0
			for i < 2 {
				for _, hp := range hp {
					queryArgs = append(queryArgs, hp)
				}
				i++
			}
			queryArgs = append(queryArgs, bun.Safe(`?`), queryValue)
			i = 0
			for i < 2 {
				for _, hp := range hp {
					queryArgs = append(queryArgs, hp)
				}
				i++
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
			i := 0
			for i < 2 {
				for _, hp := range hp {
					queryArgs = append(queryArgs, hp)
				}
				i++
			}
			queryArgs = append(queryArgs, bun.Safe(`?`), queryValue)
			i = 0
			for i < 2 {
				for _, hp := range hp {
					queryArgs = append(queryArgs, hp)
				}
				i++
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
			i, j := 0, 0
			for i < 2 {
				for j < 2 {
					for _, hp := range hp {
						queryArgs = append(queryArgs, hp)
					}
					j++
				}
				queryArgs = append(queryArgs, bun.Safe(oSQL), queryValue)
				i++
				j = 0
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
	if !e.ShowArchived {
		q.Where(`e.archived = false`)
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
		if e.Location == nil ||
			*e.Location != projectv1.LocationType_LOCATION_TYPE_HYPERPARAMETERS.String() {
			col, err := columnNameToSQL(e.ColumnName, e.Location, e.Type)
			switch *e.Operator {
			case contains:
				if c != nil && *c == or {
					q.WhereOr("? LIKE ?", bun.Safe(col), fmt.Sprintf("%%%s%%", *e.Value))
				} else {
					q.Where("? LIKE ?", bun.Safe(col), fmt.Sprintf("%%%s%%", *e.Value))
				}
			case doesNotContain:
				if c != nil && *c == or {
					q.WhereOr("? NOT LIKE ?", bun.Safe(col), fmt.Sprintf("%%%s%%", *e.Value))
				} else {
					q.Where("? NOT LIKE ?", bun.Safe(col), fmt.Sprintf("%%%s%%", *e.Value))
				}
			case empty, notEmpty:
				q = q.Where("? ?", bun.Safe(col), bun.Safe(oSQL))
			default:
				if c != nil && *c == or {
					q.WhereOr("? ? ?", bun.Safe(col),
						bun.Safe(oSQL), *e.Value)
				} else {
					q.Where("? ? ?", bun.Safe(col),
						bun.Safe(oSQL), *e.Value)
				}
			}
			if err != nil {
				return nil, err
			}
		} else {
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
