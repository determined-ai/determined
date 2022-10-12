//go:build integration
// +build integration

package elastic

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"
)

const (
	refreshWaitFor       = "wait_for"
	taskLogsTemplateName = "determined-tasklogs-template"
	taskLogsIndexPattern = "determined-tasklogs-*"
)

// WaitForIngest waits for index to be ingested.
func (e *Elastic) WaitForIngest(index string) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(model.TaskLog{}); err != nil {
		return errors.Wrap(err, "failed to make index request body")
	}
	res, err := e.client.Index(index, &buf,
		e.client.Index.WithRefresh(refreshWaitFor), e.client.Index.WithTimeout(time.Minute))
	if err != nil {
		return errors.Wrapf(err, "failed to index document")
	}
	err = checkResponse(res)
	closeWithErrCheck(res.Body)
	if err != nil {
		return errors.Wrap(err, "failed to index document")
	}
	return nil
}

// AddDateNanosTemplate adds an index template that maps timestamps to date_nanos.
func (e *Elastic) AddDateNanosTemplate() error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(jsonObj{
		"index_patterns": []string{taskLogsIndexPattern},
		"mappings": jsonObj{
			"properties": jsonObj{
				"timestamp": jsonObj{
					"type": "date_nanos",
				},
			},
		},
	}); err != nil {
		return errors.Wrap(err, "failed to make put index template request body")
	}
	res, err := e.client.Indices.PutTemplate(taskLogsTemplateName, &buf)
	if err != nil {
		return errors.Wrapf(err, "failed to put index template")
	}
	defer closeWithErrCheck(res.Body)
	if err = checkResponse(res); err != nil {
		return errors.Wrap(err, "failed to put index template")
	}
	return nil
}

// CurrentLogstashIndex returns the current logstash index.
func CurrentLogstashIndex() string {
	t := time.Now().UTC()
	return logstashIndexFromTimestamp(&t)
}
