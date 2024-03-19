package stream

import (
	"encoding/json"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/stream"
)

// StartupMsg is the first message a streaming client sends.
//
// It declares initially known keys and also configures the initial subscriptions for the stream.
type StartupMsg struct {
	SyncID    string              `json:"sync_id"`
	Known     KnownKeySet         `json:"known"`
	Subscribe SubscriptionSpecSet `json:"subscribe"`
}

// KnownKeySet allows a client to describe which primary keys it knows of as existing,
// so the server can respond with which client-known keys have been deleted or disappeared,
// and also which server-known keys are not yet known to the client (appearances).
//
// Each field of a KnownKeySet is a comma-separated list of int64s and ranges like "a,b-c,d".
type KnownKeySet struct {
	Projects string `json:"projects"`
	Models   string `json:"models"`
}

// prepareWebsocketMessage converts the MarshallableMsg into a websocket.PreparedMessage.
func prepareWebsocketMessage(obj stream.MarshallableMsg) interface{} {
	jbytes, err := json.Marshal(obj)
	if err != nil {
		log.Errorf("error marshaling message for streaming: %s", err.Error())
		return nil
	}
	msg, err := websocket.NewPreparedMessage(websocket.TextMessage, jbytes)
	if err != nil {
		log.Errorf("error preparing message for streaming: %s", err.Error())
		return nil
	}
	return msg
}
