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
		src := (*resp).Source()
		msg := "actor not found"
		if src != nil {
			msg = fmt.Sprintf("/api/v1%s not found", src.Address().String())
		}
		return status.Error(codes.NotFound, msg)
	}
	if err := (*resp).Error(); err != nil {
		fmt.Println(err)
		switch typedErr := err.(type) {
		case *echo.HTTPError:
			return status.Error(codeFromHTTPStatus(typedErr.Code), typedErr.Error())
		default:
			return err
		}
	}
	return nil
}
