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
	var lo string
	var ty string
	var col string
	if l == nil {
		lo = projectv1.LocationType_LOCATION_TYPE_EXPERIMENT.String()
	} else {
		lo = *l
	}
	if t == nil {
		ty = projectv1.ColumnType_COLUMN_TYPE_UNSPECIFIED.String()
	} else {
		ty = *t
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

	switch lo {
	case projectv1.LocationType_LOCATION_TYPE_EXPERIMENT.String():
		col, exists := filterExperimentColMap[c]
		if !exists {
			return col, fmt.Errorf("invalid experiment column %s", col)
		}
		return col, nil
	case projectv1.LocationType_LOCATION_TYPE_VALIDATIONS.String():
		col = fmt.Sprintf(`e.validation_metrics->>'%s'`, strings.TrimPrefix(c, "validation."))
		switch ty {
		case projectv1.ColumnType_COLUMN_TYPE_NUMBER.String():
			col = fmt.Sprintf(`(%v)::float8`, col)
		}
	default:
		return col, fmt.Errorf("unhandled project location %v", lo)
	}
	return col, nil
}

func hpToSQL(c string, filterColumnType *string, filterValue *interface{},
	op *operator, q *bun.SelectQuery,
	fc *filterConjunction) (*bun.SelectQuery, error) {
	var queryColumnType string
	var o operator
	var queryValue interface{}
	if filterValue == nil && op != nil && *op != empty && *op != notEmpty {
		return q, fmt.Errorf("hyperparameter field defined without value and without a valid operator")
	}
	o = *op
	if o != empty && o != notEmpty {
		queryValue = *filterValue
	}
	if filterColumnType == nil {
		queryColumnType = projectv1.ColumnType_COLUMN_TYPE_UNSPECIFIED.String()
	} else {
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
		if o != empty && o != notEmpty && //nolint: gocritic
			o != contains && o != doesNotContain { //nolint: gocritic
			queryArgs = append(queryArgs, bun.Safe(hpQuery), bun.Safe(hpQuery), bun.Safe(oSQL), queryValue)
			// nolint: lll
			queryString = `(CASE WHEN config->'hyperparameters'->?->>'type' = 'const' THEN config->'hyperparameters'->?->>'val' ? ? ELSE false END)`
			if fc != nil && *fc == or {
				return q.WhereOr(queryString, queryArgs...), nil
			}
			return q.Where(queryString, queryArgs...), nil
		} else if o == empty || o == notEmpty {
			queryArgs = append(queryArgs, bun.Safe(hpQuery),
				bun.Safe(hpQuery), bun.Safe(oSQL),
				bun.Safe(hpQuery), bun.Safe(hpQuery),
				bun.Safe(oSQL))
			// nolint: lll
			queryString = `(CASE
				WHEN config->'hyperparameters'->?->>'type' = 'const' THEN config->'hyperparameters'->?->>'val' ?
				WHEN config->'hyperparameters'->?->>'type' = 'categorical' THEN config->'hyperparameters'->?->>'vals' ?
				ELSE false
			 END)`
			if fc != nil && *fc == or {
				return q.WhereOr(queryString, queryArgs...), nil
			}
			return q.Where(queryString, queryArgs...), nil
		} else {
			if o == contains {
				// nolint: lll
				queryString = `(CASE 
					WHEN config->'hyperparameters'->?->>'type' = 'const' THEN config->'hyperparameters'->?->>'val' LIKE
					WHEN config->'hyperparameters'->?->>'type' = 'categorical' THEN (config->'hyperparameters'->?->>'vals')::jsonb ?? ?
					ELSE false
				 END)`
				queryArgs = append(queryArgs, bun.Safe(hpQuery),
					bun.Safe(hpQuery), fmt.Sprintf(`%%%s%%`, queryValue),
					bun.Safe(hpQuery), bun.Safe(hpQuery), queryValue)
				if fc != nil && *fc == or {
					return q.WhereOr(queryString, queryArgs...), nil
				}
				return q.Where(queryString, queryArgs...), nil
			}
			// nolint: lll
			queryString = `(CASE 
				WHEN config->'hyperparameters'->?->>'type' = 'const' THEN config->'hyperparameters'->?->>'val' ?
				WHEN config->'hyperparameters'->?->>'type' = 'categorical' THEN (config->'hyperparameters'->?->>'vals')::jsonb ?? ? IS NOT TRUE
				ELSE false
			 END)`
			queryArgs = append(queryArgs, bun.Safe(hpQuery),
				bun.Safe(hpQuery), `NOT LIKE %`+fmt.Sprintf(`%v`, queryValue)+`%`,
				bun.Safe(hpQuery), bun.Safe(hpQuery), queryValue)
			if fc != nil && *fc == or {
				return q.WhereOr(queryString, queryArgs...), nil
			}
			return q.Where(queryString, queryArgs...), nil
		}
	case projectv1.ColumnType_COLUMN_TYPE_DATE.String():
		if o != empty && o != notEmpty && //nolint: gocritic
			o != contains && o != doesNotContain { //nolint: gocritic
			queryArgs = append(queryArgs, bun.Safe(hpQuery),
				bun.Safe(hpQuery), bun.Safe(oSQL),
				queryValue)
			// nolint: lll
			queryString = `(CASE
				WHEN config->'hyperparameters'->?->>'type' = 'const' THEN config->'hyperparameters'->?->>'val' ? ?
				ELSE false
			 END)`
			if fc != nil && *fc == or {
				return q.WhereOr(queryString, queryArgs...), nil
			}
			return q.Where(queryString, queryArgs...), nil
		} else if o == empty || o == notEmpty {
			queryArgs = append(queryArgs, bun.Safe(hpQuery), bun.Safe(hpQuery),
				bun.Safe(oSQL), bun.Safe(hpQuery), bun.Safe(hpQuery),
				bun.Safe(oSQL))
			// nolint: lll
			queryString = `(CASE
				WHEN config->'hyperparameters'->?->>'type' = 'const' THEN config->'hyperparameters'->?->>'val' ?
				WHEN config->'hyperparameters'->?->>'type' = 'categorical' THEN config->'hyperparameters'->?->>'vals' ?
				ELSE false
			END)`
			if fc != nil && *fc == or {
				return q.WhereOr(queryString, queryArgs...), nil
			}
			return q.Where(queryString, queryArgs...), nil
		} else {
			if o == contains {
				queryArgs = append(queryArgs, bun.Safe(hpQuery), bun.Safe(hpQuery), queryValue)
				// nolint: lll
				queryString = `(CASE
					WHEN config->'hyperparameters'->?->>'type' = 'categorical' THEN (config->'hyperparameters'->?->>'vals')::jsonb ?? ?
					ELSE false
				 END)`
				if fc != nil && *fc == or {
					return q.WhereOr(queryString, queryArgs...), nil
				}
				return q.Where(queryString, queryArgs...), nil
			}
			queryArgs = append(queryArgs, bun.Safe(hpQuery), bun.Safe(hpQuery), queryValue)
			// nolint: lll
			queryString = `
				(CASE
					WHEN config->'hyperparameters'->?->>'type' = 'categorical' THEN (config->'hyperparameters'->?->>'val')::jsonb ?? ?) IS NOT TRUE
					ELSE false
				 END)`
			if fc != nil && *fc == or {
				return q.WhereOr(queryString, queryArgs...), nil
			}
			return q.Where(queryString, queryArgs...), nil
		}
	default:
		if o != empty && o != notEmpty && //nolint: gocritic
			o != contains && o != doesNotContain { //nolint: gocritic
			queryArgs = append(queryArgs, bun.Safe(hpQuery), bun.Safe(hpQuery),
				bun.Safe(oSQL), queryValue, bun.Safe(hpQuery), bun.Safe(hpQuery),
				bun.Safe(oSQL), queryValue, bun.Safe(hpQuery), bun.Safe(oSQL),
				queryValue, bun.Safe(hpQuery), bun.Safe(oSQL), queryValue)
			// nolint: lll
			queryString = `(CASE
				WHEN config->'hyperparameters'->?->>'type' = 'const' THEN (config->'hyperparameters'->?->>'val')::float8 ? ?
				WHEN config->'hyperparameters'->?->>'type' IN ('int', 'double', 'log') THEN ((config->'hyperparameters'->?->>'minval')::float8 ? ? OR (config->'hyperparameters'->?->>'maxval')::float8 ? ?)
				ELSE false
			 END)`
			if fc != nil && *fc == or {
				return q.WhereOr(queryString, queryArgs...), nil
			}
			return q.Where(queryString, queryArgs...), nil
		} else if o == empty || o == notEmpty {
			queryArgs = append(queryArgs, bun.Safe(hpQuery), bun.Safe(hpQuery),
				bun.Safe(oSQL), bun.Safe(hpQuery), bun.Safe(hpQuery),
				bun.Safe(oSQL), bun.Safe(hpQuery), bun.Safe(hpQuery),
				bun.Safe(oSQL))
			// nolint: lll
			queryString = `(CASE
				WHEN config->'hyperparameters'->?->>'type' = 'const' THEN (config->'hyperparameters'->?->>'val')::float8 ?
				WHEN config->'hyperparameters'->?->>'type' = 'categorical' THEN config->'hyperparameters'->?->>'vals' ?
				WHEN config->'hyperparameters'->?->>'type' IN ('int', 'double', 'log') THEN (config->'hyperparameters'->?) ?
				ELSE false
			 END)`
			if fc != nil && *fc == or {
				return q.WhereOr(queryString, queryArgs...), nil
			}
			return q.Where(queryString, queryArgs...), nil
		} else {
			if o == contains {
				queryArgs = append(queryArgs, bun.Safe(hpQuery), bun.Safe(hpQuery),
					bun.Safe(`?`), queryValue, bun.Safe(hpQuery), bun.Safe(hpQuery),
					queryValue, bun.Safe(hpQuery), queryValue)
				// nolint: lll
				queryString = `(CASE
					WHEN config->'hyperparameters'->?->>'type' = 'categorical' THEN (config->'hyperparameters'->?->>'vals')::jsonb ? '?'
					WHEN config->'hyperparameters'->?->>'type' IN ('int', 'double', 'log') THEN (config->'hyperparameters'->?->>'minval')::float8 <= ? OR (config->'hyperparameters'->?->>'maxval')::float8 >= ?
					ELSE false
				 END)`
				if fc != nil && *fc == or {
					return q.WhereOr(queryString, queryArgs...), nil
				}
				return q.Where(queryString, queryArgs...), nil
			}
			queryArgs = append(queryArgs, bun.Safe(hpQuery), bun.Safe(hpQuery),
				bun.Safe(`?`), queryValue, bun.Safe(hpQuery), bun.Safe(hpQuery),
				queryValue, bun.Safe(hpQuery), queryValue)
			// nolint: lll
			queryString = `(CASE
					WHEN config->'hyperparameters'->?->>'type' = 'categorical' THEN ((config->'hyperparameters'->?->>'vals')::jsonb ? '?') IS NOT TRUE
					WHEN config->'hyperparameters'->?->>'type' IN ('int', 'double', 'log') THEN (config->'hyperparameters'->?->>'minval')::float8 >= ? OR (config->'hyperparameters'->?->>'maxval')::float8 <= ?
					ELSE false
				 END)`
			if fc != nil && *fc == or {
				return q.WhereOr(queryString, queryArgs...), nil
			}
			return q.Where(queryString, queryArgs...), nil
		}
	}
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
			if *e.Operator == contains || *e.Operator == doesNotContain { //nolint: gocritic
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
				default:
					return q, fmt.Errorf("invalid contains operator %v", *e.Operator)
				}
				if err != nil {
					return q, err
				}
			} else if *e.Operator == empty || *e.Operator == notEmpty {
				q = q.Where("? ?", bun.Safe(col), bun.Safe(oSQL))
			} else {
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
