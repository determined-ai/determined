//+build integration

package elastic

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"
)

const refreshWaitFor = "wait_for"

func (e *Elastic) WaitForIngest(index string) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(model.TrialLog{}); err != nil {
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

func CurrentLogstashIndex() string {
	t := time.Now()
	return logstashIndexFromTimestamp(&t)
}
