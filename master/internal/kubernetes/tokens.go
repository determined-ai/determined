package kubernetes

import (
	"time"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
)

// The tokens actor is responsible for providing tokens to the pod actors which indicate
// to the pod actors that they are allowed to make calls to the Kubernetes API server
// to create Kubernetes resources (configMaps and pods).
//
// There are two reasons the token system is required as opposed to allowing the pod actors
// to create Kubernetes resources asynchronously:
//
//    1) Each pod creation first requires the creation of a configMap, however creating the two
//       is not an atomic operation. If there is a large number of concurrent creation requests
//       (e.g., a large HP search experiment) the kubernetes API server ends up processing the
//       creation of all the configMaps before starting to create pods, which adds significant
//       latency to the creation of pods.
//
//    2) If all creation requests are submitted asynchronously, it is possible the Kubernetes API
//       server will temporarily become saturated, and be slower to respond to other requests.
//
//  The reason that we implemented a token system rather than creating a worker pool which
//  processes the creation requests themselves (the way deletion is done), is to make it possible
//  to cancel creation requests after they are created, but before they are executed. Since the
//  actor system processes messages in a FIFO order, any cancellation request would only be
//  processed after the creation request case already been processed, requiring an unnecessary
//  resource creation and deletion. An example of this is when a large HP search is created and
//  then killed moments later. By having the pod actors request tokens, if the termination request
//  arrives while the pod actor is awaiting to receive a token, the pod actor can just cancel its
//  request for a token and avoid an unnecessary pod creation and deletion.

const (
	numTokens     = 5
	tokenCoolDown = time.Millisecond * 250
)

// message types received by the tokens actor.
type (
	requestToken struct {
		handler *actor.Ref
	}

	releaseToken struct {
		handler *actor.Ref
	}
)

// message types sent by the tokens actor.
type (
	grantToken struct{}
)

type tokens struct {
	provisionedTokens map[*actor.Ref]bool
	cancelledTokens   map[*actor.Ref]bool
}

func newTokens() *tokens {
	return &tokens{
		provisionedTokens: make(map[*actor.Ref]bool),
		cancelledTokens:   make(map[*actor.Ref]bool),
	}
}

// Receive implements the actor interface.
func (t *tokens) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:

	case requestToken:
		if err := t.receiveAcquireToken(ctx, msg); err != nil {
			return err
		}

	case releaseToken:
		t.receiveReleaseToken(ctx, msg)

	default:
		ctx.Log().Errorf("unexpected message %T", msg)
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

func (t *tokens) receiveAcquireToken(ctx *actor.Context, msg requestToken) error {
	if _, handlerPresent := t.cancelledTokens[msg.handler]; handlerPresent {
		delete(t.cancelledTokens, msg.handler)
		return nil
	}

	if len(t.provisionedTokens) == numTokens {
		actors.NotifyAfter(ctx, tokenCoolDown, msg)
		return nil
	}

	if _, handlerPresent := t.provisionedTokens[msg.handler]; handlerPresent {
		return errors.Errorf(
			"handler that already owns a token is requesting a token again %s",
			msg.handler.Address())
	}

	t.provisionedTokens[msg.handler] = true
	ctx.Tell(msg.handler, grantToken{})

	return nil
}

func (t *tokens) receiveReleaseToken(ctx *actor.Context, msg releaseToken) {
	if _, handlerPresent := t.provisionedTokens[msg.handler]; handlerPresent {
		delete(t.provisionedTokens, msg.handler)
		return
	}

	// It's possible that an actor requests a token and then releases the token
	// prior to receiving it. In this case if the token has not yet been granted
	// we need to avoid granting it.
	t.cancelledTokens[msg.handler] = true
}
