package internal

import (
	"fmt"
	"strings"

	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/uptrace/bun"
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
		return s, fmt.Errorf("invalid operator %v", *o)
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
		col, exists := filterExperimentColMap[c]
		if !exists {
			return col, fmt.Errorf("invalid experiment column %s", col)
		}
		return col, nil
	case projectv1.LocationType_LOCATION_TYPE_VALIDATIONS.String():
		col = fmt.Sprintf(`e.validation_metrics->>'%s'`, strings.TrimPrefix(c, "validation."))
		switch columnType {
		case projectv1.ColumnType_COLUMN_TYPE_NUMBER.String():
			col = fmt.Sprintf(`(%v)::float8`, col)
		}
	default:
		return col, fmt.Errorf("unhandled column location type %v", locationType)
	}
	return col, nil
}

// nolint: lll
func hpToSQL(c string, filterColumnType *string, filterValue *interface{},
	op *operator, q *bun.SelectQuery,
	fc *filterConjunction) (*bun.SelectQuery, error) {
	queryColumnType := projectv1.ColumnType_COLUMN_TYPE_UNSPECIFIED.String()
	var o operator
	var queryValue interface{}
	if filterValue == nil && op != nil && *op != empty && *op != notEmpty {
		return q, fmt.Errorf("hyperparameter field defined without value and without a valid operator")
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
	for _, h := range hp {
		hps = append(hps, fmt.Sprintf(`'%s'`, h))
	}
	hpQuery := strings.Join(hps, "->")
	oSQL, err := o.toSQL()
	if err != nil {
		return q, err
	}
	var queryArgs []interface{}
	var queryString string
	switch queryColumnType {
	case projectv1.ColumnType_COLUMN_TYPE_TEXT.String():
		switch o {
		case empty, notEmpty:
			queryArgs = append(queryArgs, bun.Safe(hpQuery),
				bun.Safe(hpQuery), bun.Safe(oSQL),
				bun.Safe(hpQuery), bun.Safe(hpQuery),
				bun.Safe(oSQL))
			queryString = `(CASE
				WHEN config->'hyperparameters'->?->>'type' = 'const' THEN config->'hyperparameters'->?->>'val' ?
				WHEN config->'hyperparameters'->?->>'type' = 'categorical' THEN config->'hyperparameters'->?->>'vals' ?
				ELSE false
			 END)`
		case contains:
			queryString = `(CASE 
				WHEN config->'hyperparameters'->?->>'type' = 'const' THEN config->'hyperparameters'->?->>'val' LIKE
				WHEN config->'hyperparameters'->?->>'type' = 'categorical' THEN (config->'hyperparameters'->?->>'vals')::jsonb ?? ?
				ELSE false
			 END)`
			queryArgs = append(queryArgs, bun.Safe(hpQuery),
				bun.Safe(hpQuery), fmt.Sprintf(`%%%s%%`, queryValue),
				bun.Safe(hpQuery), bun.Safe(hpQuery), queryValue)
		case doesNotContain:
			queryString = `(CASE 
				WHEN config->'hyperparameters'->?->>'type' = 'const' THEN config->'hyperparameters'->?->>'val' ?
				WHEN config->'hyperparameters'->?->>'type' = 'categorical' THEN (config->'hyperparameters'->?->>'vals')::jsonb ?? ? IS NOT TRUE
				ELSE false
			 END)`
			queryArgs = append(queryArgs, bun.Safe(hpQuery),
				bun.Safe(hpQuery), `NOT LIKE %`+fmt.Sprintf(`%v`, queryValue)+`%`,
				bun.Safe(hpQuery), bun.Safe(hpQuery), queryValue)
		default:
			queryArgs = append(queryArgs, bun.Safe(hpQuery), bun.Safe(hpQuery), bun.Safe(oSQL), queryValue)
			queryString = `(CASE WHEN config->'hyperparameters'->?->>'type' = 'const' THEN config->'hyperparameters'->?->>'val' ? ? ELSE false END)`
		}
	case projectv1.ColumnType_COLUMN_TYPE_DATE.String():
		switch o {
		case empty, notEmpty:
			queryArgs = append(queryArgs, bun.Safe(hpQuery), bun.Safe(hpQuery),
				bun.Safe(oSQL), bun.Safe(hpQuery), bun.Safe(hpQuery),
				bun.Safe(oSQL))
			queryString = `(CASE
				WHEN config->'hyperparameters'->?->>'type' = 'const' THEN config->'hyperparameters'->?->>'val' ?
				WHEN config->'hyperparameters'->?->>'type' = 'categorical' THEN config->'hyperparameters'->?->>'vals' ?
				ELSE false
			END)`
		case contains:
			queryArgs = append(queryArgs, bun.Safe(hpQuery), bun.Safe(hpQuery), queryValue)
			queryString = `(CASE
					WHEN config->'hyperparameters'->?->>'type' = 'categorical' THEN (config->'hyperparameters'->?->>'vals')::jsonb ?? ?
					ELSE false
				 END)`
		case doesNotContain:
			queryArgs = append(queryArgs, bun.Safe(hpQuery), bun.Safe(hpQuery), queryValue)
			queryString = `
				(CASE
					WHEN config->'hyperparameters'->?->>'type' = 'categorical' THEN (config->'hyperparameters'->?->>'val')::jsonb ?? ?) IS NOT TRUE
					ELSE false
				 END)`
		default:
			queryArgs = append(queryArgs, bun.Safe(hpQuery),
				bun.Safe(hpQuery), bun.Safe(oSQL),
				queryValue)
			queryString = `(CASE
				WHEN config->'hyperparameters'->?->>'type' = 'const' THEN config->'hyperparameters'->?->>'val' ? ?
				ELSE false
			 END)`
		}
	default:
		switch o {
		case empty, notEmpty:
			queryArgs = append(queryArgs, bun.Safe(hpQuery), bun.Safe(hpQuery),
				bun.Safe(oSQL), bun.Safe(hpQuery), bun.Safe(hpQuery),
				bun.Safe(oSQL), bun.Safe(hpQuery), bun.Safe(hpQuery),
				bun.Safe(oSQL))
			queryString = `(CASE
				WHEN config->'hyperparameters'->?->>'type' = 'const' THEN (config->'hyperparameters'->?->>'val')::float8 ?
				WHEN config->'hyperparameters'->?->>'type' = 'categorical' THEN config->'hyperparameters'->?->>'vals' ?
				WHEN config->'hyperparameters'->?->>'type' IN ('int', 'double', 'log') THEN (config->'hyperparameters'->?) ?
				ELSE false
			 END)`
		case contains:
			queryArgs = append(queryArgs, bun.Safe(hpQuery), bun.Safe(hpQuery),
				bun.Safe(`?`), queryValue, bun.Safe(hpQuery), bun.Safe(hpQuery),
				queryValue, bun.Safe(hpQuery), queryValue)
			queryString = `(CASE
					WHEN config->'hyperparameters'->?->>'type' = 'categorical' THEN (config->'hyperparameters'->?->>'vals')::jsonb ? '?'
					WHEN config->'hyperparameters'->?->>'type' IN ('int', 'double', 'log') THEN (config->'hyperparameters'->?->>'minval')::float8 <= ? OR (config->'hyperparameters'->?->>'maxval')::float8 >= ?
					ELSE false
				 END)`
		case doesNotContain:
			queryArgs = append(queryArgs, bun.Safe(hpQuery), bun.Safe(hpQuery),
				bun.Safe(`?`), queryValue, bun.Safe(hpQuery), bun.Safe(hpQuery),
				queryValue, bun.Safe(hpQuery), queryValue)
			queryString = `(CASE
					WHEN config->'hyperparameters'->?->>'type' = 'categorical' THEN ((config->'hyperparameters'->?->>'vals')::jsonb ? '?') IS NOT TRUE
					WHEN config->'hyperparameters'->?->>'type' IN ('int', 'double', 'log') THEN (config->'hyperparameters'->?->>'minval')::float8 >= ? OR (config->'hyperparameters'->?->>'maxval')::float8 <= ?
					ELSE false
				 END)`
		default:
			queryArgs = append(queryArgs, bun.Safe(hpQuery), bun.Safe(hpQuery),
				bun.Safe(oSQL), queryValue, bun.Safe(hpQuery), bun.Safe(hpQuery),
				bun.Safe(oSQL), queryValue, bun.Safe(hpQuery), bun.Safe(oSQL),
				queryValue, bun.Safe(hpQuery), bun.Safe(oSQL), queryValue)
			queryString = `(CASE
				WHEN config->'hyperparameters'->?->>'type' = 'const' THEN (config->'hyperparameters'->?->>'val')::float8 ? ?
				WHEN config->'hyperparameters'->?->>'type' IN ('int', 'double', 'log') THEN ((config->'hyperparameters'->?->>'minval')::float8 ? ? OR (config->'hyperparameters'->?->>'maxval')::float8 ? ?)
				ELSE false
			 END)`
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
		return q, err
	}
	if !e.ShowArchived {
		q.Where(`e.archived = false`)
	}
	return q, nil
}

func (e experimentFilter) toSQL(q *bun.SelectQuery,
	c *filterConjunction) (*bun.SelectQuery, error) {
	switch e.Kind {
	case field:
		if e.Operator == nil {
			return q, fmt.Errorf("field specified with value but no operator")
		}
		if e.Value == nil && *e.Operator != notEmpty && *e.Operator != empty {
			return q.Where("true"), nil //nolint:goconst
		}
		oSQL, err := e.Operator.toSQL()
		if err != nil {
			return q, err
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
				return q, err
			}
		} else {
			return hpToSQL(e.ColumnName, e.Type, e.Value, e.Operator, q, c)
		}
	case group:
		var co string
		var err error
		if e.Conjunction == nil {
			return q, fmt.Errorf("group specified with no conjunction")
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
			return q, fmt.Errorf("invalid conjunction value %v", *e.Conjunction)
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
