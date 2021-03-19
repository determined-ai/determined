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

    def report_progress(self, ...):
        # optional API; it's too hard to try to infer progress in the master
        ...

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


"""
Preemption API

Different than adaptive's early stopping; more like the cancel button or a spot instance.

Internally, the chief worker is calling context.api._should_preempt() and between workers we
are doing periodic asynchronous allgathering to decide when to preempt, so that all workers
preempt together
"""
context.should_preempt(period=10)  # called every batch or something
