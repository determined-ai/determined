package actor

import (
	"fmt"
	"reflect"

	"github.com/pkg/errors"
)

// errNoResponse is returned to the requesting actor when the requested actor
// has no response.
var errNoResponse = errors.New("no response from actor")

type errUnexpectedMessage struct {
	ctx *Context
}

func (e errUnexpectedMessage) Error() string {
	var msg interface{} = e.ctx.Message()
	if v := reflect.ValueOf(e.ctx.Message()); v.Kind() == reflect.Ptr {
		msg = v.Elem().Interface()
	}
	sender := "<external>"
	if e.ctx.sender != nil {
		sender = e.ctx.sender.Address().String()
	}
	self := "<unknown>"
	if e.ctx.Self() != nil {
		self = e.ctx.Self().Address().String()
	}
	response := "no response expected"
	if e.ctx.ExpectingResponse() {
		response = "response expected"
	}
	return fmt.Sprintf("unexpected message from %s to %s (%T): %+v (%s)",
		sender, self, e.ctx.Message(), msg, response)
}

// ErrUnexpectedMessage is returned by an actor in response to a message that it was not expecting
// to receive.
func ErrUnexpectedMessage(ctx *Context) error {
	return errUnexpectedMessage{ctx: ctx}
}
