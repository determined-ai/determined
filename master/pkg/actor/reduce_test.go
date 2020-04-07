package actor

import (
	"fmt"
	"testing"

	"gotest.tools/assert"
)

type coordinator struct {
	sources [][]int
}

func (c *coordinator) Receive(context *Context) error {
	switch context.Message().(type) {
	case string:
		for i, source := range c.sources {
			context.ActorOf(fmt.Sprintf("reducer-%d", i), &reducer{source: source})
		}
		total := 0
		for result := range context.AskAll(context.Message(), context.Children()...) {
			total += (result).Get().(int)
		}
		context.Respond(total)
	}
	return nil
}

type reducer struct {
	source []int
}

func (r *reducer) Receive(context *Context) error {
	switch context.Message().(type) {
	case string:
		total := 0
		for _, value := range r.source {
			total += value
		}
		context.Respond(total)
	}
	return nil
}

func TestSystem(t *testing.T) {
	system := NewSystem("reducer")

	coord, _ := system.ActorOf(Addr("coordinator"), &coordinator{
		sources: [][]int{
			{1, 1, 1, 1, 1},
			{5, 5, 5, 5, 5},
			{5, 5, 5, 5, 5},
			{5, 5, 5, 5, 5},
			{5, 5, 5, 5, 5},
			{5, 5, 5, 5, 5},
			{10, 10},
			{10, 10},
			{100000},
		},
	})
	assert.Equal(t, system.Ask(coord, "result").Get(), 100170)
	assert.NilError(t, coord.StopAndAwaitTermination())
}
