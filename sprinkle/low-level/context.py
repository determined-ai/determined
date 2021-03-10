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
context = det.GenericContext()

# distributed data: applies to any workload; training or inference, local or cluster.
# (same as the DistributedContext right now)
context.distributed.get_rank()
context.distributed.get_size()
context.distributed.get_local_rank()
context.distributed.get_num_agents()

# rendezvous data: info about machines and IPs that are participating in distributed jobs.
# This is configured automatically by the rendezvous info layer in Determined, but can also be
# configured automatically through an environment variable if you want to use Determined's dtrain
# features outside of Determined.
context.rendezvous.get_num_nodes()
context.rendezvous.get_node_rank()
context.rendezvous.get_slots_per_node()

# master data: applies to any job running on-cluster, of any job that you have
# chosen to connect to the master with context.connect_to_master()
context.connect_to_master(...)  # args similar to Determined() object
context.master.get_addr()
context.master.get_port()
context.master.get_tls()

# cluster data: applies to any job running on-cluster
context.cluster.get_container_id()
context.cluster.get_container_gpus()
context.cluster.get_slot_ids()

# training-related data; only applies to `det e create` workloads
context.training.get_experiment_config()
context.training.get_data_config()
context.training.get_experiment_id()
context.training.get_trial_id()
context.training.get_global_batch_size()
context.training.get_per_slot_batch_size()
context.training.get_hparams()
context.training.get_trial_seed()
context.training.get_latest_checkpoint()

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
# (fault tolerant via context.training.get_latest_checkpoint)
op = context.training.get_searcher_op()
    # op will be one of TrainingOp() or ValidationOp()
training_op.complete(
    checkpoint=...    # required: an already-completed checkpoint uuid
)
validation_op.complete(
    searcher_metric=...  # required
)


# Preemption API:
# (different than adaptive's  early stopping; more like the cancel button or a spot instance)
context.distributed.should_preempt(period=10)
    # internally, the cheif worker is calling context.api._should_preempt() and between workers we
    # are doing periodic asynchronous allreduces to decide when to preempt, so that all workers
    # preempt together
