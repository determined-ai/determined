.. _features:

##########
 Concepts
##########

Deep learning practitioners come from a variety of disciplines. Depending on their background, some
practitioners have strong foundations in engineering, while others focus on statistics and domain
expertise. Determined AI is a deep learning training platform that simplifies infrastructure
management for domain experts while enabling configuration-based deep learning functionality that is
generally inconvenient to implement for engineering-oriented practitioners.

Many current systems are point solutions for specific problems in deep learning, so combining the
systems is tough and inefficient. Determined's cohesive end-to-end training platform provides
best-in-class functionality for deep learning model training, with a suite of benefits, including:

-  **Cluster management**: Automatically manage ML accelerators (e.g., GPUs) on-premise or in cloud
   VMs, using your own environment that automatically scales for your on-demand workloads.
   Determined runs in either AWS or GCP, so you can switch easily as your needs require.

-  **Containerization**: Develop and train models in customizable containers, which enable simple
   and consistent dependency management throughout the model development lifecycle.

-  **Cluster-backed notebooks, commands, and shells**: Leverage your shared cluster computing
   devices in a more versatile environment.

-  **Experiment collaboration**: Automatically track the configuration and environment for each of
   your experiments, facilitating reproducibility and collaboration among teams.

-  **Visualization**: Visualize your model and training procedure by using Determined's built-in
   WebUI, and also by launching managed tensorboard instances.

-  **Fault tolerance**: Models are checkpointed throughout the training process and can be restarted
   from the latest checkpoint automatically. This enables training jobs to automatically tolerate
   transient hardware or system issues in the cluster.

-  **Automated model tuning**: Optimize models by searching through conventional hyperparameters or
   macro-architectures, using a variety of search algorithms. Hyperparameter searches are
   automatically parallelized across the accelerators in the cluster.

-  **Distributed training**: Easily distribute a single training job across multiple accelerators to
   speed up model training and reduce model development iteration time. Determined uses synchronous,
   data-parallel distributed training, with key performance optimizations over other available
   options.

-  **Broad framework support**: Leverage these capabilities using any of the leading machine
   learning frameworks without having to manage a different cluster for each. Different frameworks
   for different models can be used without worrying about future lock-in.

.. toctree::
   :maxdepth: 1
   :hidden:

   elastic-infrastructure
   experiment
   resource-pool
   scheduling
   yaml
