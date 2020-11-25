package elastic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/determined-ai/determined/master/internal/api"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"
)

// The maximum size of elasticsearch queries on a cluster with default configurations.
// Can be increased but is limited by Lucene's 2m cap also.
const (
	ElasticMaxQuerySize    = 10000
	ElasticTimeWindowDelay = -10 * time.Second
)

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
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": append(filtersToElastic(fs),
					map[string]interface{}{
						"term": map[string]interface{}{
							"trial_id": trialID,
						},
					}),
			},
		},
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return 0, fmt.Errorf("failed to encoding query: %w", err)
	}

	res, err := e.client.Count(
		e.client.Count.WithBody(&buf))
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve log count: %w", err)
	}
	defer res.Body.Close()

	resp := struct {
		Count int `json:"count"`
	}{}
	err = json.NewDecoder(res.Body).Decode(&resp)
	if err != nil {
		return 0, fmt.Errorf("failed to decode count api response")
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
	if limit > ElasticMaxQuerySize {
		limit = ElasticMaxQuerySize
	}

	query := map[string]interface{}{
		// Use from+size to begin at the requested offset, but move to search after
		// API after first query to paginate the requests.
		"size": limit,
		"from": offset,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": append(filtersToElastic(fs),
					map[string]interface{}{
						"term": map[string]interface{}{
							"trial_id": trialID,
						},
					},
					// Only look at logs posted more than 10 seconds ago. In the event
					// a fluentbit shipper is backed up, it may post logs with a timestamp
					// that falls before the current time. If we do a search_after based on
					// the latest logs, we may miss these backed up unless we do this.
					map[string]interface{}{
						"range": map[string]interface{}{
							"timestamp": map[string]interface{}{
								"lt": time.Now().UTC().Add(ElasticTimeWindowDelay),
							},
						},
					}),
			},
		},
		"sort": []map[string]interface{}{
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
		return nil, nil, fmt.Errorf("failed to encoding query: %w", err)
	}

	res, err := e.client.Search(e.client.Search.WithBody(&buf))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to perform search: %w", err)
	}
	defer res.Body.Close()

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

		var containerID string
		if h.Source.ContainerID != nil {
			containerID = *h.Source.ContainerID
		} else {
			containerID = "UNKNOWN CONTAINER"
		}

		var rankID string
		if h.Source.RankID != nil {
			rankID = fmt.Sprintf("[rank=%d]", *h.Source.RankID)
		}

		var level string
		if h.Source.Level != nil {
			level = *h.Source.Level
		}

		h.Source.Message = fmt.Sprintf("[%s] [%s] %s || %s %s",
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

func filtersToElastic(fs []api.Filter) []map[string]interface{} {
	var terms []map[string]interface{}
	for _, f := range fs {
		switch f.Operation {
		case api.FilterOperationIn:
			switch reflect.TypeOf(f.Values).Kind() {
			case reflect.Slice:
				s := reflect.ValueOf(f.Values)
				var inTerms []map[string]interface{}
				for i := 0; i < s.Len(); i++ {
					inTerms = append(inTerms,
						map[string]interface{}{
							"term": map[string]interface{}{
								// filter against the keyword not the analyzed text.
								f.Field + ".keyword": s.Index(i).Interface(),
							},
						})
				}
				terms = append(terms,
					map[string]interface{}{
						"bool": map[string]interface{}{
							"should": inTerms,
						},
					})
			default:
				panic(fmt.Sprintf("unsupported IN filter values %T", f.Values))
			}
		case api.FilterOperationLessThan:
			terms = append(terms,
				map[string]interface{}{
					"range": map[string]interface{}{
						f.Field: map[string]interface{}{
							"lt": f.Values,
						},
					},
				})
		case api.FilterOperationGreaterThan:
			terms = append(terms,
				map[string]interface{}{
					"range": map[string]interface{}{
						f.Field: map[string]interface{}{
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

func logstashIndexFromTimestamp(time *time.Time) string {
	return time.UTC().Format("triallogs-2006.01.02")
}
