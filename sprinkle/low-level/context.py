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
context = det.generic.init()  # args similar to Determined() object, for connecting to a master if
                              # you are running from off-master

# master data: applies to any job running on-cluster or if you logged in at init()
context.master.addr
context.master.port
context.master.tls

# cluster data: applies to any job running on-cluster (is this actually needed anywhere?)
context._cluster.container_id
context._cluster.container_gpus
context._cluster.slot_ids

# distributed data: applies to any workload; training or inference, local or cluster.
# (same as the DistributedContext right now)
context.distributed.rank
context.distributed.size
context.distributed.local_rank
context.distributed.num_agents
# zmq_allgather is used for barriers or metric reducers of various sorts
# zmq_allgather is more "consistently available" than "performant"
context.distributed.zmq_allgather(Any) -> List[Any]
# async allgather needed for distributed preemption
context.distributed._start_zmq_allgather(Any) -> GatherID
context.distributed._finish_zmq_allgather(GatherID) -> List[Any]

# rendezvous data: info about machines and IPs that are participating in distributed jobs.
# This is configured automatically by the rendezvous info layer in Determined, but can also be
# configured automatically through an environment variable if you want to use Determined's dtrain
# features outside of Determined.
context.rendezvous.num_nodes
context.rendezvous.node_rank
context.rendezvous.slots_per_node

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

"""
PUSH-ARCHITECTURE RELATED APIS:

The core features of our platform can be delivered if we only have metrics and checkpoints for
training.  Advanced features like preemption and hpsearch should also be possible, but MUST NOT
be necessary to deliver basic features.
"""

# Training metrics

context.training.begin_training(  # optional call, improves webui experience
    start_time=...      # optional: defaults to time.time()
)
context.training.report_training_metrics(
    batches_trained=... # required: how many batches were trained for this report

    metrics=...         # optional: reduced metrics, shown in webui
    batch_metrics=...   # optional: accessible via python sdk
    records_trained=... # optional: how many records trained for this report
    epochs_trained=...  # optional: how many epochs trained for this report
    end_time=...        # optional: shows in webui
)

# Validation metrics
context.training.begin_validation(  # optional call, improves webui experience
    start_time=...      # optional: defaults to time.time()
)
context.training.report_validation_metrics(
    metrics=...         # required: reduced metrics, shown in webui
    end_time=...        # optional: shows in webui
)

# Checkpoints
context.api._begin_checkpoint()                             # non-user-facing call
context.api._report_checkpoint(uuid, start_time, end_time)  # non-user-facing call
# user-facing API, just wraps the StorageManagers:
with context.checkpoint.save_path as path:
    ... # user saves checkpoint into path
# TODO: for downloading, checkpoints, do we just stick to some form of Checkpoint Export API?


"""
Searcher Push API

Right now the searcher passes Training and Validation operations.  I vote we simplify it to emit
only one kind of operation, which combines them, since it's never possible to emit any sequence
other than pairs of Training/Validation.

If you make the length an absolute length instead of an incremental length, then you can
completely separate the checkpointing logic from the searcher logic.  This moves us closer
to a world where any job can checkpoint and restore via a common API, rather than just training
jobs.

Fault tolerance is handled by just restoring every trial from its latest checkpoint
whenever we restore it.

This means: if you want adaptive search but don't want to implement resuming, you can
still achive katib-style adaptive in Determined.
"""

class SearcherOp:
    """
    You get a SearcherOp from context.training.next_searcher_op().

    A SearcherOp is like the master saying:

        "tell me the searcher metric when you have finished X amount of training"

    and *nothing* else.
    """
    def __init__(self, context, unit, length):
        self._context = context
        self._unit = unit  # one of EPOCHS, BATCHES, or RECORDS
        self._length = length  # int

    @property
    def unit(self):
        return self._unit

    @property
    def length(self):
        return self._length

    @property
    def records(self):
        assert self._unit == RECORDS
        return self._length

    @property
    def batches(self):
        assert self._unit == BATCHES
        return self._length

    @property
    def epochs(self):
        assert self._unit == EPOCHS
        return self._length

    def complete(self, searcher_metric):
        # tell the master about the searcher metric;
        # the next call to next_searcher_op() will now return something new
        context.training._complete_searcher_op(searcher_metric)

# maybe an iterator to wrap next_searcher_op()?
for op in contex.training.iter_searcher_ops():
    # obviously you'd use our keras first-class support instead,
    # but for academic purposes, you could just feed this value to your trainer
    metrics = model.fit(epochs=op.epochs)
    op.complete(seacher_metric=metrics["val_accuracy"])


# Preemption API:
# (different than adaptive's early stopping; more like the cancel button or a spot instance)
context.distributed.should_preempt(period=10)
    # internally, the cheif worker is calling context.api._should_preempt() and between workers we
    # are doing periodic asynchronous allreduces to decide when to preempt, so that all workers
    # preempt together
