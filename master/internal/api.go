package internal

import (
	"fmt"
	"reflect"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/rbac"
	"github.com/determined-ai/determined/master/internal/templates"
	"github.com/determined-ai/determined/master/internal/trials"
	"github.com/determined-ai/determined/master/internal/usergroup"
	"github.com/determined-ai/determined/master/internal/webhooks"
	"github.com/determined-ai/determined/master/pkg/actor"
)

type apiServer struct {
	m *Master

	usergroup.UserGroupAPIServer
	rbac.RBACAPIServerWrapper
	webhooks.WebhooksAPIServer
	trials.TrialSourceInfoAPIServer
	trials.TrialsAPIServer
	templates.TemplateAPIServer
}

// ask asks at addr the req and puts the response into what v points at. When appropriate,
// errors are converted appropriate for an API response. Error cases are enumerated below:
//   - If v points to an unsettable value, a 500 is returned.
//   - If the actor cannot be found, a 404 is returned.
//   - If v is settable and the actor didn't respond or responded with nil, a 404 is returned.
//   - If the actor returned an error and it is a well-known error type, it is coalesced to gRPC.
//   - If the actor returned plain error, a 500 is returned.
//   - Finally, if the response's type is OK, it is put into v.
//   - Else, a 500 is returned.
func (a *apiServer) ask(addr actor.Address, req interface{}, v interface{}) error {
	if reflect.ValueOf(v).IsValid() && !reflect.ValueOf(v).Elem().CanSet() {
		return status.Errorf(
			codes.Internal,
			"ask to actor %s contains valid but unsettable response holder %T", addr, v,
		)
	}
	expectingResponse := reflect.ValueOf(v).IsValid() && reflect.ValueOf(v).Elem().CanSet()
	switch resp := a.m.system.AskAt(addr, req); {
	case resp.Source() == nil:
		return api.NotFoundErrs("actor", fmt.Sprint(addr), true)
	case expectingResponse && resp.Empty(), expectingResponse && resp.Get() == nil:
		return status.Errorf(
			codes.NotFound,
			"actor %s %s", addr, actorDidNotRespond,
		)
	case resp.Error() != nil:
		if ok, err := api.EchoErrToGRPC(resp.Error()); ok {
			return err
		}
		return api.APIErrToGRPC(resp.Error())
	default:
		if expectingResponse {
			if reflect.ValueOf(v).Elem().Type() != reflect.ValueOf(resp.Get()).Type() {
				return status.Errorf(
					codes.Internal,
					"actor %s returned unexpected message (%T): %v", addr, resp, resp,
				)
			}
			reflect.ValueOf(v).Elem().Set(reflect.ValueOf(resp.Get()))
		}
		return nil
	}
}

const (
	actorDidNotRespond = "did not respond"
)
