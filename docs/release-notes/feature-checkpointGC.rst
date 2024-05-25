:orphan:

**New Features**

-  Kubernetes: Add a feature where Determined offers the users to provide custom Checkpoint GC pod spec for 
    individual experiment. You can specify a custom ``checkpointGcPodSpec`` directly in the experiment 
    configuration file. This configuration is done using the ``task_container_defaults.checkpoint_gc_pod_spec`` 
    field within your experiment ``value.yaml`` file. When a custom CheckpointGC pod spec is defined for an 
    experiment, it takes precedence over the default CheckpointGC pod specifications. This provides flexibility 
    to tailor the garbage collection settings according to the specific needs of the experiment.