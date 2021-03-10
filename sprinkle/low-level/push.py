"""
PUSH-ARCHITECTURE RELATED APIS:

The core features of our platform can be delivered if we only have metrics and checkpoints for
training.  Advanced features like preemption and hpsearch should also be possible, but MUST NOT
be necessary to deliver basic features.
"""

# Training metrics
context.training.begin_training()  # optional call, improves webui experience
context.training.report_training_metrics(
    metrics=...         # optional: reduced metrics, shown in webui
    batch_metrics=...   # optional: accessible via python sdk
    batches_trained=... # optional: epochs/batches/shows in webui
    records_trained=... # optional: shows in webui
    start_time=...      # optional: shows in webui
    end_time=...        # optional: shows in webui
)

# Validation metrics
context.training.begin_validation()  # optional call, improves webui experience
context.training.report_validation_metrics(
    metrics=...       # required: reduced metrics, shown in webui
    start_time=...    # optional: shows in webui
    end_time=...      # optional: shows in webui
)

# Checkpoints
context.api._begin_checkpoint()                             # non-user-facing call
context.api._report_checkpoint(uuid, start_time, end_time)  # non-user-facing call
# user-facing API, just wraps the StorageManagers:
with context.checkpoint.save_path as path:
    ... # user saves checkpoint into path
# TODO: for downloading, checkpoints, do we just stick to some form of Checkpoint Export API?


# Searcher API:
"""
SEARCHER API

goals:
    - makes standalone sense
    - searcher API is not required to use metrics/checkpoint APIs
    - easy for users to interact with directly, if they want to use a
      non-supported ml framework

Plan:
    To respond to a training op, include a completed checkpoint id.
    To respond to a validation op, include just the searcher metric.

    This means you might have to e.g. redo a validation metric if you report
    validation to one api but crash before contacting the searcher.  That's an
    acceptable cost for making the searcher API accessible to real users.

        # returns Union[None, TrainingOp, ValidationOp]
        op = context.api.next_searcher_op()

        # training op: checkpoint uuid be already-reported via checkpoint API
        op.complete(metrics=..., checkpoint=...)

        # validation op: report the searcher metric
        op.complete(searcher_metric=...)
"""

context.training.get_latest_checkpoint() # fault tolerance

op = context.training.get_searcher_op()
    # op will be one of TrainingOp() or ValidationOp()

training_op.complete(
    checkpoint=...    # required: an already-completed checkpoint uuid
)

validation_op.complete(
    searcher_metric=...  # required
)


"""
Preemption API

Different than adaptive's early stopping; more like the cancel button or a spot instance.

Internally, the cheif worker is calling context.api._should_preempt() and between workers we
are doing periodic asynchronous allgathering to decide when to preempt, so that all workers
preempt together
"""
context.distributed.should_preempt(period=10)  # called every batch or something
