package elastic

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"
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
	for index, logs := range indexToLogs {
		for _, l := range logs {
			b, err := json.Marshal(l)
			if err != nil {
				return errors.Wrap(err, "failed to make index request body")
			}
			res, err := e.client.Index(index, bytes.NewReader(b))
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

func logstashIndexFromTimestamp(time *time.Time) string {
	return time.UTC().Format("triallogs-2006.01.02")
}
