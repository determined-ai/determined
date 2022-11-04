.. _slurm-image-config:

#################################
 Provide a Container Image Cache
#################################

When the cluster does not have Internet access or if you want to provide a local cache of container
images to improve performance, you can download the desired container images to a shared directory
and then reference them using file system paths instead of docker registry references.

There are two mechanisms you can use to reference cached container images depending upon the
container runtime in use.

   -  :ref:`referencing-local-image-paths`
   -  :ref:`singularity-image-cache`

***********************
 Default Docker Images
***********************

Each version of Determined utilizes specifically-tagged Docker containers. The image tags referenced
by default in this version of Determined are described below.

+-------------+-------------------------------------------------------------------------+
| Environment | File Name                                                               |
+=============+=========================================================================+
| CPUs        | ``determinedai/environments:py-3.8-pytorch-1.10-tf-2.8-cpu-096d730``    |
+-------------+-------------------------------------------------------------------------+
| Nvidia GPUs | ``determinedai/environments:cuda-11.3-pytorch-1.10-tf-2.8-gpu-096d730`` |
+-------------+-------------------------------------------------------------------------+
| AMD GPUs    | ``determinedai/environments:rocm-5.0-pytorch-1.10-tf-2.7-rocm-096d730`` |
+-------------+-------------------------------------------------------------------------+

See :doc:`/training/setup-guide/set-environment-images` for the images Docker Hub location, and add
each tagged image needed by your experiments to the image cache.

.. _referencing-local-image-paths:

*******************************
 Referencing Local Image Paths
*******************************

Each container runtime supports various local container file formats and references them using a
slightly different syntax. Utilize a cached image by referencing a local path using the experiment
configuration :ref:`environment.image <exp-environment-image>`.

When using PodMan, you could save images in OCI archive format to files in a local directory
``/shared/containers``

      .. code:: bash

         podman save determinedai/environments:cuda-11.3-pytorch-1.10-tf-2.8-gpu-096d730 \
           --format=oci-archive \
           -o /shared/containers/cuda-11.3-pytorch-1.10-tf-2.8-gpu

   and then reference the image in your experiment configuration using the syntax below.

      .. code:: yaml

         environment:
            image: oci-archive:/shared/containers/cuda-11.3-pytorch-1.10-tf-2.8-gpu

When using Singularity, you could save SIF files in a local directory ``/shared/containers``

   .. code:: bash

      singularity pull /shared/containers/cuda-11.3-pytorch-1.10-tf-2.8-gpu \
         determinedai/environments:cuda-11.3-pytorch-1.10-tf-2.8-gpu-096d730

and then reference in your experiment configuration using a full path using the syntax below.

   .. code:: yaml

      environment:
         image: /shared/containers/cuda-11.3-pytorch-1.10-tf-2.8-gpu.sif

Set these ``image`` file references above as the default for all jobs by specifying them in the
:ref:`task_container_defaults <master-task-container-defaults>` section of the
``/etc/determined/master.yaml`` file.

Note: If you specify an image using :ref:`task_container_defaults <master-task-container-defaults>`,
you prevent new environment container image versions from being adopted on each update of
Determined.

.. _singularity-image-cache:

*************************************************
 Configuring a Singularity Image Cache Directory
*************************************************

When using Singularity, you may use :ref:`referencing-local-image-paths` as described above, or you
may instead configure a directory tree of images to be searched. To utilize this capability, provide
a shared directory in :ref:`resource_manager.singularity_image_root <cluster-configuration-slurm>`.
Whenever an image is referenced, it is translated to a local file path as described in
:ref:`environment.image <exp-environment-image>`. If found, the local path is substituted in the
``singularity run`` command to avoid the need for Singularity to download and convert the image for
each user.

Add each tagged image required by your environment and the needs of your experiments to the image
cache:

#. Create a directory path using the same prefix as the image name referenced in the
   ``singularity_image_root`` directory. For example, the image
   ``determinedai/environments:cuda-11.3-pytorch-1.10-tf-2.8-gpu-096d730`` is added in the directory
   ``determinedai``.

   .. code:: bash

      cd $singularity_image_root
      mkdir determinedai

#. If your system has internet access, you can download images directly into the cache.

   .. code:: bash

      cd $singularity_image_root
      image="determinedai/environments:cuda-11.3-pytorch-1.10-tf-2.8-gpu-096d730"
      singularity pull $image docker://$image

#. Otherwise, from an internet-connected system, download the desired image using the Singularity
   pull command then copy it to the ``determinedai`` folder under ``singularity_image_root``.

   .. code:: bash

      singularity pull \
            temporary-image \
            docker://$image
      scp temporary-image mycluster:$singularity_image_root/$image
