.. _benefits:

##########
 Benefits
##########

Determined is a deep learning training platform that simplifies infrastructure management for domain
experts while enabling configuration-based deep learning functionality that engineering-oriented
practitioners might find inconvenient to implement. The Determined cohesive, end-to-end training
platform provides best-in-class functionality for deep learning model training, including the
following benefits:

+------------------------------------------------+-----------------------------------------------------------+
| Implementation                                 | Benefit                                                   |
+================================================+===========================================================+
| Automated model tuning                         | Optimize models by searching through conventional         |
|                                                | hyperparameters or macro- architectures, using a variety  |
|                                                | of search algorithms. Hyperparameter searches are         |
|                                                | automatically parallelized across the accelerators in the |
|                                                | cluster. See :ref:`hyperparameter-tuning`.                |
+------------------------------------------------+-----------------------------------------------------------+
| Cluster-backed notebooks, commands, and shells | Leverage your shared cluster computing devices in a more  |
|                                                | versatile environment. See :ref:`notebooks` and           |
|                                                | :ref:`commands-and-shells`.                               |
+------------------------------------------------+-----------------------------------------------------------+
| Cluster management                             | Automatically manage ML accelerators, such as GPUs,       |
|                                                | on-premise or in cloud VMs using your own environment,    |
|                                                | automatically scaling for your on-demand workloads.       |
|                                                | Determined runs in either AWS or GCP, so you can switch   |
|                                                | easily according to your requirements. See :ref:`Resource |
|                                                | Pools <resource-pools>`, :ref:`Scheduling <scheduling>`,  |
|                                                | and :ref:`Elastic Infrastructure                          |
|                                                | <elastic-infrastructure>`.                                |
+------------------------------------------------+-----------------------------------------------------------+
| Containerization                               | Develop and train models in customizable containers that  |
|                                                | enable simple, consistent dependency management           |
|                                                | throughout the model development lifecycle. See           |
|                                                | :ref:`custom-env`.                                        |
+------------------------------------------------+-----------------------------------------------------------+
| Distributed training                           | Easily distribute a single training job across multiple   |
|                                                | accelerators to speed up model training and reduce model  |
|                                                | development iteration time. Determined uses synchronous,  |
|                                                | data-parallel distributed training, with key performance  |
|                                                | optimizations over other available options. See           |
|                                                | :ref:`multi-gpu-training-concept`.                        |
+------------------------------------------------+-----------------------------------------------------------+
| Experiment collaboration                       | Automatically track your experiment configuration and     |
|                                                | environment to facilitate reproducibility and             |
|                                                | collaboration among teams. See :ref:`experiments`.        |
+------------------------------------------------+-----------------------------------------------------------+
| Fault tolerance                                | Models are checkpointed throughout the training process   |
|                                                | and can be restarted from the latest checkpoint,          |
|                                                | automatically. This enables training jobs to              |
|                                                | automatically tolerate transient hardware or system       |
|                                                | issues in the cluster.                                    |
+------------------------------------------------+-----------------------------------------------------------+
| Framework support                              | Broad framework support leverages these capabilities      |
|                                                | using any of the leading machine learning frameworks      |
|                                                | without needing to manage a different cluster for each.   |
|                                                | Different frameworks for different models can be used     |
|                                                | without risking future lock-in. See                       |
|                                                | :ref:`apis-howto-overview`.                               |
+------------------------------------------------+-----------------------------------------------------------+
| Profiling                                      | Out-of-the-box system metrics (measurements of hardware   |
|                                                | usage) and timings (durations of actions taken during     |
|                                                | training, such as data loading).                          |
+------------------------------------------------+-----------------------------------------------------------+
| Visualization                                  | Visualize your model and training procedure by using The  |
|                                                | built-in WebUI and by launching managed                   |
|                                                | :ref:`tensorboards` instances.                            |
+------------------------------------------------+-----------------------------------------------------------+
