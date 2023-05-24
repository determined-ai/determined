package elastic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"time"

	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

const (
	// The maximum size of elasticsearch queries on a cluster with default configurations.
	// Can be increased but is limited by Lucene's 2m cap also.
	elasticMaxQuerySize = 10000
	// ElasticTimeWindowDelay is the time buffer to allow logs to come in before we try to serve
	// them up. We do this to not miss later logs when using search_after the way we do.
	ElasticTimeWindowDelay = 10 * time.Second
)

type jsonObj = map[string]interface{}

// AddTaskLogs indexes a batch of tasks logs into the index like tasklogs-yyyy-MM-dd based
// on the UTC value of their timestamp.
func (e *Elastic) AddTaskLogs(logs []*model.TaskLog) error {
	indexToLogs := map[string][]*model.TaskLog{}
	for _, l := range logs {
		index := logstashIndexFromTimestamp(l.Timestamp)
		indexToLogs[index] = append(indexToLogs[index], l)
	}
	// NOTE: This could potentially use the bulk APIs, but the official
	// client's support for them is very heavy - it is built to spawn
	// multiple go routines and use persistent bulk indexer objects - way
	// overkill (to the point of being slower) for the small number of logs that go here now.
	// See: https://github.com/elastic/go-elasticsearch/blob/master/_examples/bulk/indexer.go
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for index, logs := range indexToLogs {
		for _, l := range logs {
			if err := enc.Encode(l); err != nil {
				return errors.Wrap(err, "failed to make index request body")
			}
			res, err := e.client.Index(index, &buf)
			if err != nil {
				return errors.Wrapf(err, "failed to index document")
			}
			err = checkResponse(res)
			closeWithErrCheck(res.Body)
			if err != nil {
				return errors.Wrap(err, "failed to index document")
			}
		}
	}
	return nil
}

// TaskLogsCount returns the number of logs for the given task.
func (e *Elastic) TaskLogsCount(taskID model.TaskID, fs []api.Filter) (int, error) {
	count, err := e.count(jsonObj{
		"query": jsonObj{
			"bool": jsonObj{
				"filter": append(filtersToElastic(fs),
					jsonObj{
						"term": jsonObj{
							"task_id.keyword": taskID,
						},
					}),
			},
		},
	})
	if err != nil {
		return 0, errors.Wrap(err, "failed to get task log count")
	}
	return count, nil
}

// TaskLogs return a set of logs matching the provided criteria from the task.
// This uses the search after API, since from+size is prohibitively
// expensive for deep pagination and the scroll api specifically recommends
// search after over itself.
// https://www.elastic.co/guide/en/elasticsearch/reference/6.8/search-request-search-after.html
func (e *Elastic) TaskLogs(
	taskID model.TaskID, limit int, fs []api.Filter, order apiv1.OrderBy,
	searchAfter interface{},
) ([]*model.TaskLog, interface{}, error) {
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
							"task_id.keyword": taskID,
						},
					},
					// Only look at logs posted more than 10 seconds ago. In the event
					// a fluentbit shipper is backed up, it may post logs with a timestamp
					// that falls before the current time. If we do a search_after based on
					// the latest logs, we may miss these backed up unless we do this.
					// This probably should be using PIT instead.
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
				ID     string         `json:"_id"`
				Source *model.TaskLog `json:"_source"`
				Sort   []interface{}  `json:"sort"`
			} `json:"hits"`
		} `json:"hits"`
	}{}

	if err := e.search(query, &resp); err != nil {
		return nil, nil, errors.Wrap(err, "failed to query task logs")
	}

	var logs []*model.TaskLog
	for i := range resp.Hits.Hits {
		// The short form `for _, h := range resp.Hits.Hits` will result in &h.ID being
		// the same address and all logs having identical IDs.
		h := resp.Hits.Hits[i]
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

// DeleteTaskLogs deletes the logs for the given tasks.
func (e *Elastic) DeleteTaskLogs(ids []model.TaskID) error {
	taskIDterms := make([]jsonObj, len(ids))
	for i, id := range ids {
		taskIDterms[i] = jsonObj{
			"term": jsonObj{
				"task_id.keyword": id,
			},
		}
	}

	query := jsonObj{
		"query": jsonObj{
			"bool": jsonObj{
				"filter": []jsonObj{
					{
						"bool": jsonObj{
							"should": taskIDterms,
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

// TaskLogsFields returns the unique fields that can be filtered on for the given task.
func (e *Elastic) TaskLogsFields(taskID model.TaskID) (*apiv1.TaskLogsFieldsResponse, error) {
	query := jsonObj{
		"size": 0,
		"query": jsonObj{
			"bool": jsonObj{
				"filter": []jsonObj{
					{
						"term": jsonObj{
							"task_id.keyword": taskID,
						},
					},
				},
			},
		},
		"aggs": jsonObj{
			// These keys are the aggregate names; they must match the aggregate names we expect to
			// be returned, which are defined in the type of resp below.
			"allocation_ids": jsonObj{
				"terms": jsonObj{
					"field": "allocation_id.keyword",
				},
			},
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
			AllocationIDs stringAggResult `json:"allocation_ids"`
			AgentIDs      stringAggResult `json:"agent_ids"`
			ContainerIDs  stringAggResult `json:"container_ids"`
			RankIDs       intAggResult    `json:"rank_ids"`
			Sources       stringAggResult `json:"sources"`
			StdTypes      stringAggResult `json:"stdtypes"`
		} `json:"aggregations"`
	}{}
	if err := e.search(query, &resp); err != nil {
		return nil, errors.Wrap(err, "failed to aggregate trial log fields")
	}

	return &apiv1.TaskLogsFieldsResponse{
		AllocationIds: resp.Aggregations.AllocationIDs.toKeys(),
		AgentIds:      resp.Aggregations.AgentIDs.toKeys(),
		ContainerIds:  resp.Aggregations.ContainerIDs.toKeys(),
		RankIds:       resp.Aggregations.RankIDs.toKeysInt32(),
		Stdtypes:      resp.Aggregations.StdTypes.toKeys(),
		Sources:       resp.Aggregations.Sources.toKeys(),
	}, nil
}

// MaxTerminationDelay is the max delay before a consumer can be sure all logs have been recevied.
// For Elasticsearch, this _must_ be greater than internal/elastic.elasticTimeWindowDelay or else
// following terminates before all logs are delivered.
func (e *Elastic) MaxTerminationDelay() time.Duration {
	return ElasticTimeWindowDelay + time.Second
}

// search runs the search request with query as its body and populates the result into resp.
func (e *Elastic) search(query jsonObj, resp interface{}) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return errors.Wrap(err, "failed to encoding query")
	}

	res, err := e.client.Search(e.client.Search.WithBody(&buf))
	if err != nil {
		return errors.Wrap(err, "failed to perform search")
	}
	defer closeWithErrCheck(res.Body)
	if err = checkResponse(res); err != nil {
		return errors.Wrap(err, "failed to perform search")
	}

	err = json.NewDecoder(res.Body).Decode(resp)
	if err != nil {
		return fmt.Errorf("failed to decode search api response")
	}

	return nil
}

// count runs the count request with query as its body returns the result.
func (e *Elastic) count(query jsonObj) (int, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return 0, errors.Wrap(err, "failed to encode query")
	}

	res, err := e.client.Count(e.client.Count.WithBody(&buf))
	if err != nil {
		return 0, errors.Wrap(err, "failed to retrieve log count")
	}
	defer closeWithErrCheck(res.Body)
	if err = checkResponse(res); err != nil {
		return 0, errors.Wrap(err, "failed to retrieve log count")
	}

	resp := struct {
		Count int `json:"count"`
	}{}
	err = json.NewDecoder(res.Body).Decode(&resp)
	if err != nil {
		return 0, errors.New("failed to decode count api response")
	}

	return resp.Count, nil
}

func filtersToElastic(fs []api.Filter) []jsonObj {
	var terms []jsonObj
	for _, f := range fs {
		switch f.Operation {
		case api.FilterOperationIn:
			values, err := interfaceToSlice(f.Values)
			if err != nil {
				panic(fmt.Errorf("invalid IN filter values: %w", err))
			}
			var inTerms []jsonObj
			for _, v := range values {
				switch v.(type) {
				case string:
					// For strings, we filter against the keyword not the analyzed text.
					// If you have any text field, for example `agent_id`, by default,
					// elasticsearch will analyze this field and operations against it
					// use this analyzed field, leading to unexpected results.
					// The text fields we use should be stored as multi-fields, with an
					// additional field `keyword` under the original field that stores
					// the input as type `keyword` for literal comparisons. When elastic
					// encounters JSON strings, the default dynamic mappings for
					// the cluster will create this field.
					// Relates to https://github.com/elastic/elasticsearch/issues/53020
					// and https://github.com/elastic/elasticsearch/issues/53181.
					inTerms = append(inTerms,
						jsonObj{
							"term": jsonObj{
								f.Field + ".keyword": v,
							},
						})
				default:
					inTerms = append(inTerms,
						jsonObj{
							"term": jsonObj{
								f.Field: v,
							},
						})
				}
			}
			terms = append(terms,
				jsonObj{
					"bool": jsonObj{
						"should": inTerms,
					},
				})
		case api.FilterOperationInOrNull:
			// used to specify value matches int[] or is nil
			// see https://stackoverflow.com/questions/48563275/
			values, err := interfaceToSlice(f.Values)
			if err != nil {
				panic(fmt.Errorf("invalid IN OR NULL filter values: %w", err))
			}
			var inTerms []jsonObj
			for _, v := range values {
				switch v.(type) {
				case string:
					// See notes on FilterOperationIn
					inTerms = append(inTerms,
						jsonObj{
							"term": jsonObj{
								f.Field + ".keyword": v,
							},
						})
				default:
					inTerms = append(inTerms,
						jsonObj{
							"term": jsonObj{
								f.Field: v,
							},
						})
				}
			}
			inTerms = append(inTerms,
				jsonObj{
					"bool": jsonObj{
						"must_not": jsonObj{
							"exists": jsonObj{
								"field": f.Field,
							},
						},
					},
				})
			terms = append(terms,
				jsonObj{
					"bool": jsonObj{
						"should": inTerms,
					},
				})
		case api.FilterOperationLessThanEqual:
			terms = append(terms,
				jsonObj{
					"range": jsonObj{
						f.Field: jsonObj{
							"lte": f.Values,
						},
					},
				})
		case api.FilterOperationGreaterThan:
			terms = append(terms,
				jsonObj{
					"range": jsonObj{
						f.Field: jsonObj{
							"gt": f.Values,
						},
					},
				})
		case api.FilterOperationStringContainment:
			terms = append(terms,
				jsonObj{
					"wildcard": jsonObj{
						f.Field: jsonObj{
							"value": fmt.Sprintf("*%s*", f.Values),
						},
					},
				})

		default:
			panic(fmt.Sprintf("unsupported filter operation: %d", f.Operation))
		}
	}
	return terms
}

func orderByToElastic(order apiv1.OrderBy) string {
	const (
		ascKeyword  = "asc"
		descKeyword = "desc"
	)
	switch order {
	case apiv1.OrderBy_ORDER_BY_UNSPECIFIED:
		return ascKeyword
	case apiv1.OrderBy_ORDER_BY_ASC:
		return ascKeyword
	case apiv1.OrderBy_ORDER_BY_DESC:
		return descKeyword
	default:
		panic(fmt.Sprintf("unexpected order by: %s", order))
	}
}

// interfaceToSlice accepts an interface{} whose underlying type is []T for any T
// and returns it as type []interface{}.
func interfaceToSlice(x interface{}) ([]interface{}, error) {
	var iSlice []interface{}
	switch reflect.TypeOf(x).Kind() {
	case reflect.Slice:
		s := reflect.ValueOf(x)
		for i := 0; i < s.Len(); i++ {
			iSlice = append(iSlice, s.Index(i).Interface())
		}
	default:
		return nil, fmt.Errorf("interfaceToSlice only accepts slice, not %T", x)
	}
	return iSlice, nil
}

type stringAggResult struct {
	DocCountErrorUpperBound int `json:"doc_count_error_upper_bound"`
	SumOtherDocCount        int `json:"sum_other_doc_count"`
	Buckets                 []struct {
		DocCount int    `json:"doc_count"`
		Key      string `json:"key"`
	} `json:"buckets"`
}

func (r stringAggResult) toKeys() []string {
	var keys []string
	for _, b := range r.Buckets {
		keys = append(keys, b.Key)
	}
	return keys
}

type intAggResult struct {
	DocCountErrorUpperBound int `json:"doc_count_error_upper_bound"`
	SumOtherDocCount        int `json:"sum_other_doc_count"`
	Buckets                 []struct {
		DocCount int `json:"doc_count"`
		Key      int `json:"key"`
	} `json:"buckets"`
}

func (r intAggResult) toKeysInt32() []int32 {
	var keys []int32
	for _, b := range r.Buckets {
		keys = append(keys, int32(b.Key))
	}
	return keys
}

func logstashIndexFromTimestamp(time *time.Time) string {
	return time.UTC().Format("determined-tasklogs-2006.01.02")
}

func checkResponse(res *esapi.Response) error {
	if res.StatusCode > 299 || res.StatusCode < 200 {
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body with code %d", res.StatusCode)
		}
		return fmt.Errorf("request failed with code %d: %s", res.StatusCode, b)
	}
	return nil
}

func closeWithErrCheck(closer io.Closer) {
	err := closer.Close()
	if err != nil {
		log.Errorf("error closing closer: %s", err)
	}
}
