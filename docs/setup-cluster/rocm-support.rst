.. _rocm-support:

##################
 AMD ROCm Support
##################

.. contents:: Table of Contents
   :local:
   :depth: 2

**********
 Overview
**********

.. note::
   ROCm support in Determined is experimental. Features and configurations may change in future releases. We recommend testing thoroughly in a non-production environment before deploying to production.

Determined provides experimental support for AMD ROCm GPUs in Kubernetes deployments. Determined
provides prebuilt Docker images for ROCm, including the latest ROCm 6.1 version with DeepSpeed
support for MI300x users:

-  `pytorch-infinityhub-dev
   <https://hub.docker.com/repository/docker/determinedai/pytorch-infinityhub-dev/tags>`__
-  `pytorch-infinityhub-hpc-dev
   <https://hub.docker.com/repository/docker/determinedai/pytorch-infinityhub-hpc-dev/tags>`__

You can build these images locally based on the Dockerfiles found in the `environments repository
<https://github.com/determined-ai/environments/blob/main/Dockerfile-infinityhub-pytorch>`__.

For more detailed information about configuration, visit the :ref:`helm-config-reference` or visit
:ref:`rocm-known-issues` for details on current limitations and troubleshooting.

.. _rocm-config-k8s:

**************************************
 Configuring Kubernetes for ROCm GPUs
**************************************

To use ROCm GPUs in your Kubernetes deployment:

1. Ensure your Kubernetes cluster has nodes with ROCm-capable GPUs and the necessary drivers installed.

2. In your Helm chart values or Determined configuration, set the following:

   .. code-block:: yaml

      resourceManager:
        defaultComputeResourcePool: rocm-pool

      resourcePools:
        - pool_name: rocm-pool
          gpu_type: rocm
          max_slots: <number_of_rocm_gpus>

3. When submitting experiments or launching tasks, specify ``slot_type: rocm`` in your experiment configuration.

*********************************
 Using ROCm Images in Experiments
*********************************

To use ROCm images in your experiments, specify the image in your experiment configuration:

.. code-block:: yaml

   environment:
     image: determinedai/pytorch-infinityhub-dev:rocm6.1-pytorch2.1-deepspeed0.10.0

Ensure that your experiment configuration also specifies ``slot_type: rocm`` to use ROCm GPUs.

.. _rocm-known-issues:

******************************
 Known Issues and Limitations
******************************

-  **Agent Deprecation**: Agent-based deployments are deprecated for ROCm support. Use Kubernetes with ROCm support for your deployments.

-  **HIP GPU Errors**: Launching experiments with ``slot_type: rocm`` may fail with the error
   ``RuntimeError: No HIP GPUs are available``. Ensure compute nodes have compatible ROCm drivers and
   libraries installed and available in default locations or added to the ``PATH`` and/or ``LD_LIBRARY_PATH``.

-  **Boost Filesystem Errors**: You may encounter the error ``boost::filesystem::remove: Directory
   not empty`` during ROCm operations. A workaround is to disable per-container ``/tmp`` using bind mounts 
   in your experiment configuration or globally using the ``task_container_defaults`` section in your master configuration:

      .. code:: yaml

         bind_mounts:
            - host_path: /tmp
              container_path: /tmp
