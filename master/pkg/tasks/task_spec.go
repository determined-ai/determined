package tasks

// MakeTaskSpecFn is a workaround for the delayed initialization that we have around how tasks are
// run.  The master knows which task spec and task container defaults belong to which pool, but the
// actual parsing of configs might be delegated to e.g. a CommandManager which does not have access
// to the same information.  This lets us avoid extra Asks or passing the Master object around.
type MakeTaskSpecFn func(poolName string, numSlots int) TaskSpec
