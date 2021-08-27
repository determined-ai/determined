"""
CONTEXT OBJECTS IN THE SPRINKLE API

The context object is core to how we reverse the plugin experience into a
library experience; the context object is sort of like an instance of a
library that has some information bound within it that is particular to the
current running task.  E.g. det.get_trial_id() would be a strange library call
but context.get_trial_id() is very natural.

Goals for context objects in the Sprinkle API:

 - There should only be one context class for all things PyTorchTrial, and
   only one for all things Keras, etc.  The context class for PyTorchTrial and
   the context class for Keras don't have to be the same though.

 - The same context object must be usable in both local training and cluster
   training environments.  Obviously if it exposes cluster-training-specific
   extensions during cluster training, those extensions do not need to work in
   local training environments, but any user code which does not depend on
   such environment-specific extensions should run equally well in either
   environment.

 - The same context object must be usable in both training and non-training
   workloads.  The sprinkle API for training should feel analagous to the
   sprnkle API for distributed batch inference and to distributed test set
   metrics, even though the calls the user makes will differ.

 - Backwards compatibility: the existing Trial APIs need to still work, so
   those TrialContext classes should remain unchanged.
"""

# GENERIC CONTEXT
# Useful for custom workloads and is the basis for the framework contexts.

with det.generic.init(
    rank_info=...,  # provide info from a custom dtrain backend.
    session=...,  # provide a session() if you are running off-cluster (future enhancement)
) as context:  # use a contextmanager so we can start/stop off-thread resources (e.g. profiler)
    ...

# cluster data: applies to any job running on-cluster (is this actually needed anywhere?)
context._cluster.container_id
context._cluster.container_gpus
context._cluster.slot_ids

# distributed data: applies to any workload; training or inference, local or cluster.
# (same as the DistributedContext right now)
context.distributed.rank
context.distributed.size
context.distributed.local_rank
context.distributed.local_size
context.distributed.cross_rank
context.distributed.cross_size
# zmq_allgather is used for barriers or metric reducers of various sorts
# zmq_allgather is more "consistently available" than "performant"
context.distributed.zmq_allgather(Any) -> List[Any]
# async allgather needed for distributed preemption
context.distributed._start_zmq_allgather(Any) -> GatherID
context.distributed._finish_zmq_allgather(GatherID) -> List[Any]

# training-related data; only applies to `det e create` workloads
context.training.experiment_config
context.training.data_config
context.training.experiment_id
context.training.trial_id
context.training.global_batch_size
context.training.per_slot_batch_size
context.training.hparams
context.training.trial_seed
context.training.initial_checkpoint
