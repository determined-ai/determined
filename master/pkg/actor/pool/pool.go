package pool

import (
	"github.com/determined-ai/determined/master/pkg/actor"
)

// QueueFullError is returned by SubmitTask when the pool is so backed up that the queue is full.
type QueueFullError struct{}

func (e QueueFullError) Error() string {
	return "actor pool queue is full"
}

// ActorPool implements thread-pool like behavior in the actor system. It is given a function to
// handle tasks, and will call this function with each task submitted to the actor pool.
type ActorPool struct {
	name string

	taskHandler func(task interface{}) interface{}
	callback    func(result interface{})

	queueLimit   uint
	workersLimit uint
	workers      uint
	counter      uint64

	manager *actor.Ref
	system  *actor.System

	queue chan interface{}
}

// NewActorPool initializes a new actor pool and starts its manager actor.
func NewActorPool(
	system *actor.System,
	queueLimit uint,
	workersLimit uint,
	name string,
	taskHandler func(task interface{}) interface{},
	callback func(result interface{})) ActorPool {
	pool := ActorPool{
		name: name,

		taskHandler: taskHandler, // May be called many times in parallel
		callback:    callback,    // Will only be called by the manager actor

		queueLimit:   queueLimit,
		workersLimit: workersLimit,

		system: system,

		queue: make(chan interface{}, queueLimit),
	}
	ref, _ := system.ActorOf(actor.Addr(name), &pool)
	pool.manager = ref
	return pool
}

// SubmitTask is a convenience function for sending a task to the actor pool's manager actor.
func (p *ActorPool) SubmitTask(task interface{}) error {
	result := p.system.Ask(p.manager, sendTask{task}).Get()
	if result != nil {
		return result.(error)
	}
	return nil
}

// Internal message types
type (
	// For giving a new task to the manager.
	sendTask struct {
		task interface{}
	}

	// For workers to request their next task.
	receiveTask struct{}

	// To return the result of a task to the manager.
	returnTask struct {
		result interface{}
	}

	// Instructs a worker to start it's main loop.
	workerUp struct{}

	// Instructs a worker to stop. Not used as a "message" in Receive, but a response to a worker's
	// Ask().
	workerDown struct{}
)

// Manager

// Receive handles all the messages for the actor pool manager.
func (p *ActorPool) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart, actor.PostStop, actor.ChildStopped:
		// Expected life-cycle messages; do nothing
	case actor.ChildFailed:
		ctx.Log().Warnf("worker failed in actor pool %s: %+v", p.name, msg)
	case sendTask:
		if uint(len(p.queue)) == p.queueLimit {
			ctx.Respond(QueueFullError{})
			return nil
		}
		p.queue <- msg.task
		if p.workers < p.workersLimit {
			newWorker := worker{pool: p}
			ref, _ := ctx.ActorOf(p.counter, newWorker)
			p.counter++
			p.workers++
			ctx.Tell(ref, workerUp{})
		}
	case receiveTask:
		var response interface{}
		select {
		case realTask := <-p.queue:
			response = realTask
		default:
			response = workerDown{}
			p.workers--
		}
		ctx.Respond(response)
	case returnTask:
		if p.callback != nil {
			p.callback(msg.result)
		}
	default:
		ctx.Log().Errorf("unknown message received by actor pool %s: %+v!",
			p.name, ctx.Message())
	}
	return nil
}

// Worker

type worker struct {
	pool *ActorPool
}

// Receive contains the main loop for the worker. Beyond life-cycle messages, it receives a single
// workerUp message from the manager, then asks for tasks from the master until receiving a
// workerDown message.
func (w worker) Receive(ctx *actor.Context) error {
	// Other than minimal life-cycle messages, a worker receives a workerUp message and is then
	// stuck in an infinite loop, polling for tasks from the master until it receives a workerDown
	// response and stops itself.
	switch ctx.Message().(type) {
	case actor.PreStart, actor.PostStop:
		// Expected life-cycle messages; do nothing
	case workerUp:
		for {
			task := ctx.Ask(ctx.Sender(), receiveTask{}).Get()

			switch realTask := task.(type) {
			case workerDown:
				ctx.Self().Stop()
				break
			default:
				result := w.pool.taskHandler(realTask)
				ctx.Tell(ctx.Sender(), returnTask{result})
			}
		}
	default:
		ctx.Log().Errorf("unknown message received by actor pool worker: %v!", ctx.Message())
	}
	return nil
}
