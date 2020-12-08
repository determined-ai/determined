package elastic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/proto/pkg/apiv1"

	"github.com/determined-ai/determined/master/internal/api"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"
)

const (
	// The maximum size of elasticsearch queries on a cluster with default configurations.
	// Can be increased but is limited by Lucene's 2m cap also.
	elasticMaxQuerySize = 10000
	// A time buffer to allow logs to come in before we try to serve them up. We do this to
	// not miss later logs when using search_after.
	elasticTimeWindowDelay = -10 * time.Second
)

type jsonObj = map[string]interface{}

// AddTrialLogs indexes a batch of trial logs into the index like triallogs-yyyy-MM-dd based
// on the UTC value of their timestamp.
func (e *Elastic) AddTrialLogs(logs []*model.TrialLog) error {
	indexToLogs := map[string][]*model.TrialLog{}
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
			err := enc.Encode(l)
			if err != nil {
				return errors.Wrap(err, "failed to make index request body")
			}
			res, err := e.client.Index(index, &buf)
			if err != nil {
				return errors.Wrapf(err, "failed to index document")
			}
			err = res.Body.Close()
			if err != nil {
				return errors.Wrap(err, "failed to close index response body")
			}
		}
	}
	return nil
}

// TrialLogCount returns the number of trial logs for the given trial.
func (e *Elastic) TrialLogCount(trialID int, fs []api.Filter) (int, error) {
	query := jsonObj{
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
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return 0, fmt.Errorf("failed to encode query: %w", err)
	}

	res, err := e.client.Count(
		e.client.Count.WithBody(&buf))
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve log count: %w", err)
	}
	defer closeWithErrCheck(res.Body)

	resp := struct {
		Count int `json:"count"`
	}{}
	err = json.NewDecoder(res.Body).Decode(&resp)
	if err != nil {
		return 0, errors.New("failed to decode count api response")
	}
	return resp.Count, nil
}

// TrialLogs return a set of trial logs within a specified window.
// This uses the search after API, since from+size is prohibitively
// expensive for deep pagination and the scroll api specifically recommends
// search after over itself.
// https://www.elastic.co/guide/en/elasticsearch/reference/6.8/search-request-search-after.html
func (e *Elastic) TrialLogs(
	trialID, offset, limit int, fs []api.Filter, searchAfter interface{},
) ([]*model.TrialLog, interface{}, error) {
	if limit > elasticMaxQuerySize {
		limit = elasticMaxQuerySize
	}

	query := jsonObj{
		// Use from+size to begin at the requested offset, but move to search after
		// API after first query to paginate the requests.
		"size": limit,
		"from": offset,
		"query": jsonObj{
			"bool": jsonObj{
				"filter": append(filtersToElastic(fs),
					jsonObj{
						"term": jsonObj{
							"trial_id": trialID,
						},
					},
					// Only look at logs posted more than 10 seconds ago. In the event
					// a fluentbit shipper is backed up, it may post logs with a timestamp
					// that falls before the current time. If we do a search_after based on
					// the latest logs, we may miss these backed up unless we do this.
					jsonObj{
						"range": jsonObj{
							"timestamp": jsonObj{
								"lt": time.Now().UTC().Add(elasticTimeWindowDelay),
							},
						},
					}),
			},
		},
		"sort": []jsonObj{
			{"timestamp": "asc"},
			// If two containers emit logs with the same timestamp down
			// to the nanosecond, it may be lost in some cases still, but
			// this should be better than nothing.
			{"container_id.keyword": "asc"},
		},
	}

	if searchAfter != nil {
		query["search_after"] = searchAfter
		// If a request comes with searchAfter values, offset is meaningless
		// so we just remove it.
		delete(query, "from")
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, nil, fmt.Errorf("failed to encode query: %w", err)
	}

	res, err := e.client.Search(e.client.Search.WithBody(&buf))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to perform search: %w", err)
	}
	defer closeWithErrCheck(res.Body)

	resp := struct {
		Hits struct {
			Hits []struct {
				Source *model.TrialLog `json:"_source"`
				Sort   []interface{}   `json:"sort"`
			} `json:"hits"`
		} `json:"hits"`
	}{}
	err = json.NewDecoder(res.Body).Decode(&resp)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode search api response")
	}

	var logs []*model.TrialLog
	for _, h := range resp.Hits.Hits {
		var timestamp string
		if h.Source.Timestamp != nil {
			timestamp = h.Source.Timestamp.Format(time.RFC3339Nano)
		} else {
			timestamp = "UNKNOWN TIME"
		}

		// This is just to match postgres.
		const containerIDMaxLength = 8
		var containerID string
		if h.Source.ContainerID != nil {
			containerID = *h.Source.ContainerID
			if len(containerID) > containerIDMaxLength {
				containerID = containerID[:containerIDMaxLength]
			}
		} else {
			containerID = "UNKNOWN CONTAINER"
		}

		var rankID string
		if h.Source.RankID != nil {
			rankID = fmt.Sprintf("[rank=%d] ", *h.Source.RankID)
		}

		var level string
		if h.Source.Level != nil {
			level = fmt.Sprintf("%s: ", *h.Source.Level)
		}

		h.Source.Message = fmt.Sprintf("[%s] [%s] %s|| %s %s",
			timestamp, containerID, rankID, level, *h.Source.Log)
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
				inTerms = append(inTerms,
					jsonObj{
						"term": jsonObj{
							// filter against the keyword not the analyzed text. If you
							// have any text field, for example `agent_id`, by default,
							// elasticsearch will analyze this field and operations against it
							// use this analyzed field, leading to unexpected results.
							// The text fields we use should be stored as multi-fields, with an
							// additional field `keyword` under the original field that stores
							// the input as type `keyword` for literal comparisons. When elastic
							// encounters JSON strings, the default dynamic mappings for
							// the cluster will create this field.
							// Relates to https://github.com/elastic/elasticsearch/issues/53020
							// and https://github.com/elastic/elasticsearch/issues/53181.
							f.Field + ".keyword": v,
						},
					})
			}
			terms = append(terms,
				jsonObj{
					"bool": jsonObj{
						"should": inTerms,
					},
				})
		case api.FilterOperationLessThan:
			terms = append(terms,
				jsonObj{
					"range": jsonObj{
						f.Field: jsonObj{
							"lt": f.Values,
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
		default:
			panic(fmt.Sprintf("unsupported filter operation: %d", f.Operation))
		}
	}
	return terms
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

// TrialLogFields returns the unique fields that can be filtered on for the given trial.
func (e *Elastic) TrialLogFields(trialID int) (*apiv1.TrialLogsFieldsResponse, error) {
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

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, fmt.Errorf("failed to encode query: %w", err)
	}

	res, err := e.client.Search(e.client.Search.WithBody(&buf))
	if err != nil {
		return nil, fmt.Errorf("failed to perform search: %w", err)
	}
	defer closeWithErrCheck(res.Body)

	resp := struct {
		Aggregations struct {
			AgentIDs     stringAggResult `json:"agent_ids"`
			ContainerIDs stringAggResult `json:"container_ids"`
			RankIDs      intAggResult    `json:"rank_ids"`
			Sources      stringAggResult `json:"sources"`
			StdTypes     stringAggResult `json:"stdtypes"`
		} `json:"aggregations"`
	}{}
	err = json.NewDecoder(res.Body).Decode(&resp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode search api response")
	}

	return &apiv1.TrialLogsFieldsResponse{
		AgentIds:     resp.Aggregations.AgentIDs.toKeys(),
		ContainerIds: resp.Aggregations.ContainerIDs.toKeys(),
		RankIds:      resp.Aggregations.RankIDs.toKeysInt32(),
		Stdtypes:     resp.Aggregations.StdTypes.toKeys(),
		Sources:      resp.Aggregations.Sources.toKeys(),
	}, err
}

func logstashIndexFromTimestamp(time *time.Time) string {
	return time.UTC().Format("triallogs-2006.01.02")
}

func closeWithErrCheck(closer io.Closer) {
	err := closer.Close()
	if err != nil {
		log.Errorf("error closing closer: %s", err)
	}
}
