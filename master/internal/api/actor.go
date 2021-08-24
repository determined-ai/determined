package api

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/labstack/echo/v4"

	"github.com/determined-ai/determined/master/pkg/actor"
)

func codeFromHTTPStatus(code int) codes.Code {
	switch {
	case code == 400:
		return codes.InvalidArgument
	case 200 <= code && code < 300:
		return codes.OK
	}
	return codes.Internal
}

// ProcessActorResponseError checks actor resposne for errors.
func ProcessActorResponseError(resp *actor.Response) error {
	if (*resp).Empty() {
		msg := "actor not found"
		if src := (*resp).Source(); src != nil {
			msg = fmt.Sprintf("actor not found: /api/v1%s", src.Address().String())
		}
		return status.Error(codes.NotFound, msg)
	}
	if err := (*resp).Error(); err != nil {
		switch typedErr := err.(type) {
		case *echo.HTTPError:
			return status.Error(codeFromHTTPStatus(typedErr.Code), typedErr.Error())
		default:
			return err
		}
	}
	return nil
}
