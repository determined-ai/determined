:orphan:

**New Features**

- SDK: Support for tracking and viewing batch inference metrics
  - Adds ability to keep track of what experiments use particular
    checkpoint or model version for inference. 
  - New ``.get_metrics()`` functionality added to ``checkpoint.Checkpoint``
    and ``model.ModelVersion`` SDK objects to fetch related metrics.

**Improvements**

- SDK: ``client.stream_training_metrics()`` and ``client.stream_validation_metrics()``
  deprecated. Please use ``client.stream_trials_metrics()`` instead.
