"""
PUSH-ARCHITECTURE RELATED APIS:

The core features of our platform can be delivered if we only have metrics and checkpoints for
training.  Advanced features like preemption and hpsearch should also be possible, but MUST NOT
be necessary to deliver basic features.
"""

# Training metrics
context.training.begin_training()  # optional call, improves webui experience
context.training.report_training_metrics(
    # rest api details
    metrics=...         # optional: reduced metrics, shown in webui
    batch_metrics=...   # optional: accessible via python sdk
    batches_trained=... # optional: epochs/batches/shows in webui
    records_trained=... # optional: shows in webui
    start_time=...      # optional: shows in webui
    end_time=...        # optional: shows in webui

    # python-api details
    reducer=...         # optional: for assisted metric reduction across workers
)

# Validation metrics
context.training.begin_validation()  # optional call, improves webui experience
context.training.report_validation_metrics(
    metrics=...       # required: reduced metrics, shown in webui
    start_time=...    # optional: shows in webui
    end_time=...      # optional: shows in webui

    # python-api details
    reducer=...         # optional: for assisted metric reduction across workers
)

# For implementing save_experiment_best:
context.training.get_experiment_best_validation()

# Checkpoints
context.api._begin_checkpoint()                             # non-user-facing call
context.api._report_checkpoint(uuid, start_time, end_time)  # non-user-facing call
# user-facing API, just wraps the StorageManagers:
with context.checkpoint.save_path() as path:
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

Note that not all searchers will support searchers configured by records or batches.  Training loops
which only support epoch-based searchers include:
   - Keras
   - Estimator
   - Pytorch Lightning
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

    def report_progress(self, length)
        # optional API; it's too hard to try to infer progress in the master
        context.training._report_progress(length)

    def complete(self, searcher_metric):
        # tell the master about the searcher metric;
        # the next call to next_searcher_op() will now return something new
        context.training._complete_searcher_op(searcher_metric)


# The "basic" searcher API supporst all searchers with epoch-based training.
basic_searcher = context.training.get_basic_searcher()
for epoch in basic_searcher.epochs(initial_epoch=0):
    metrics = train_one_epoch_and_validate()
    # We automatically report progress or complete the op as necessary.
    basic_searcher.report(metrics["val_accuracy"])


# The "advanced" searcher API has you use SearcherOps directly.
epochs_complete = 0
advanced_searcher = context.training.get_advanced_searcher()
for op in advanced_searcher.ops():

    # It's on you to report metrics.
    def epoch_end_cb():
        nonlocal epochs_complete
        epochs_complete += 1
        advanced_searcher.report_searcher_progress(epochs_complete)

    do_training(length=op.epochs, cb=epoch_end_cb)

    val_metrics = do_validation()
    op.complete(val_metrics["accuracy"])



"""
Preemption API

Different than adaptive's early stopping; more like the cancel button or a spot instance.

(Normally, a worker thread is running on the chief to get the preemption message from the master,
either via long-polling or via websockets.  When the preemption signal is received, the thread will
set a flag that the chief worker can read.)

By itself, should_preempt() is blocking.  This is meant to be called every epoch, so it will exit
after the first epoch where a pause or deschedule was requested.  The chief checks for the
preemption flag (from the worker thread) and broadcasts the decision to all workers.

When block=False, the chief will check the preemption and broadcast the decision to all workers.
However, the result will not be gathered until the beginning of the *next* call to should_preempt()
(the beginning of the next batch) which effectively hides the network latency behind the forwards
and backwards passes for the batch.  This is meant to be called every batch, so you can have very
responsive preemption for a snappy cluster experience or for spot instance support.
"""
context.should_preempt()             # blocking, to call every epoch or something
                                     # (emits warnings about performance if you call it too much)

context.should_preempt(block=False)  # performant enough to call every batch
