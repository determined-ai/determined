package user

import (
	"github.com/determined-ai/determined/master/pkg/model"
)

func IsSSOUser(_ model.User) bool {
	return false
}
