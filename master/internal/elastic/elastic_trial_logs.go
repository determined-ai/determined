package elastic

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/determined-ai/determined/proto/pkg/apiv1"

	"github.com/determined-ai/determined/master/internal/api"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"
)

// TrialLogsCount returns the number of trial logs for the given trial.
func (e *Elastic) TrialLogsCount(trialID int, fs []api.Filter) (int, error) {
	count, err := e.count(jsonObj{
		"query": jsonObj{
			"bool": jsonObj{
				"filter": append(filtersToElastic(fs),
					jsonObj{
						"term": jsonObj{
							"trial_id": trialID,
						},
					}),
			},
		},
	})
	if err != nil {
		return 0, errors.Wrap(err, "failed to get trial log count")
	}
	return count, nil
}

// TrialLogs return a set of trial logs within a specified window.
// This uses the search after API, since from+size is prohibitively
// expensive for deep pagination and the scroll api specifically recommends
// search after over itself.
// https://www.elastic.co/guide/en/elasticsearch/reference/6.8/search-request-search-after.html
func (e *Elastic) TrialLogs(
	trialID, limit int, fs []api.Filter, order apiv1.OrderBy, searchAfter interface{},
) ([]*model.TrialLog, interface{}, error) {
	if limit > elasticMaxQuerySize {
		limit = elasticMaxQuerySize
	}

	query := jsonObj{
		"size": limit,
		"query": jsonObj{
			"bool": jsonObj{
				"filter": append(filtersToElastic(fs),
					jsonObj{
						"term": jsonObj{
							"trial_id": trialID,
						},
					},
					// Only look at logs posted more than 10 seconds ago. In the event
					// a fluentbit shipper (deprecated) is backed up, it may post logs with a timestamp
					// that falls before the current time. If we do a search_after based on
					// the latest logs, we may miss these backed up unless we do this.
					jsonObj{
						"range": jsonObj{
							"timestamp": jsonObj{
								"lte": time.Now().UTC().Add(-ElasticTimeWindowDelay),
							},
						},
					}),
			},
		},
		"sort": []jsonObj{
			{"timestamp": orderByToElastic(order)},
			// If two containers emit logs with the same timestamp down
			// to the nanosecond, it may be lost in some cases still, but
			// this should be better than nothing.
			{
				"container_id.keyword": jsonObj{
					"order": orderByToElastic(order),
					// https://www.elastic.co/guide/en/elasticsearch/reference/7.9/
					// sort-search-results.html#_ignoring_unmapped_fields
					"unmapped_type": "keyword",
				},
			},
		},
	}

	if searchAfter != nil {
		query["search_after"] = searchAfter
	}

	resp := struct {
		Hits struct {
			Hits []struct {
				ID     string          `json:"_id"`
				Source *model.TrialLog `json:"_source"`
				Sort   []interface{}   `json:"sort"`
			} `json:"hits"`
		} `json:"hits"`
	}{}

	if err := e.search(query, &resp); err != nil {
		return nil, nil, errors.Wrap(err, "failed to query trial logs")
	}

	var logs []*model.TrialLog
	for i := range resp.Hits.Hits {
		// The short form `for _, h := range resp.Hits.Hits` will result in &h.ID being
		// the same address and all logs having identical IDs.
		h := resp.Hits.Hits[i]
		h.Source.Resolve()
		h.Source.StringID = &h.ID
		logs = append(logs, h.Source)
	}

	var sortValues interface{}
	if len(resp.Hits.Hits) > 0 {
		sortValues = resp.Hits.Hits[len(resp.Hits.Hits)-1].Sort
	} else if searchAfter != nil {
		sortValues = searchAfter
	}

	return logs, sortValues, nil
}

// DeleteTrialLogs deletes the logs for the given trial IDs.
func (e *Elastic) DeleteTrialLogs(ids []int) error {
	trialIDterms := make([]jsonObj, len(ids))
	for i, id := range ids {
		trialIDterms[i] = jsonObj{
			"term": jsonObj{
				"trial_id": id,
			},
		}
	}

	query := jsonObj{
		"query": jsonObj{
			"bool": jsonObj{
				"filter": []jsonObj{
					{
						"bool": jsonObj{
							"should": trialIDterms,
						},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return errors.Wrap(err, "failed to encoding query")
	}

	// TODO(brad): Here and elsewhere, we really should just hit indices that could possibly have
	// logs for a given trial.
	res, err := e.client.DeleteByQuery([]string{"*"}, &buf)
	if err != nil {
		return errors.Wrap(err, "failed to perform delete")
	}
	defer closeWithErrCheck(res.Body)
	if err = checkResponse(res); err != nil {
		return errors.Wrap(err, "failed to perform delete")
	}

	return nil
}

// TrialLogsFields returns the unique fields that can be filtered on for the given trial.
func (e *Elastic) TrialLogsFields(trialID int) (*apiv1.TrialLogsFieldsResponse, error) {
	query := jsonObj{
		"size": 0,
		"query": jsonObj{
			"bool": jsonObj{
				"filter": []jsonObj{
					{
						"term": jsonObj{
							"trial_id": trialID,
						},
					},
				},
			},
		},
		"aggs": jsonObj{
			// These keys are the aggregate names; they must match the aggregate names we expect to
			// be returned, which are defined in the type of resp below.
			"agent_ids": jsonObj{
				"terms": jsonObj{
					"field": "agent_id.keyword",
				},
			},
			"container_ids": jsonObj{
				"terms": jsonObj{
					"field": "container_id.keyword",
				},
			},
			"rank_ids": jsonObj{
				"terms": jsonObj{
					"field": "rank_id",
				},
			},
			"sources": jsonObj{
				"terms": jsonObj{
					"field": "source.keyword",
				},
			},
			"stdtypes": jsonObj{
				"terms": jsonObj{
					"field": "stdtype.keyword",
				},
			},
		},
	}
	resp := struct {
		Aggregations struct {
			AgentIDs     stringAggResult `json:"agent_ids"`
			ContainerIDs stringAggResult `json:"container_ids"`
			RankIDs      intAggResult    `json:"rank_ids"`
			Sources      stringAggResult `json:"sources"`
			StdTypes     stringAggResult `json:"stdtypes"`
		} `json:"aggregations"`
	}{}
	if err := e.search(query, &resp); err != nil {
		return nil, errors.Wrap(err, "failed to aggregate trial log fields")
	}

	return &apiv1.TrialLogsFieldsResponse{
		AgentIds:     resp.Aggregations.AgentIDs.toKeys(),
		ContainerIds: resp.Aggregations.ContainerIDs.toKeys(),
		RankIds:      resp.Aggregations.RankIDs.toKeysInt32(),
		Stdtypes:     resp.Aggregations.StdTypes.toKeys(),
		Sources:      resp.Aggregations.Sources.toKeys(),
	}, nil
}
