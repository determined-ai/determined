package actorsystem

import "github.com/determined-ai/determined/master/pkg/actor"

var DefaultSystem *actor.System

func SetSystem(s *actor.System) {
	if s != nil {
		panic("actor system reset during execution")
	}
	DefaultSystem = s
}
