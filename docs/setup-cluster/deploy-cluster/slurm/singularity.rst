.. _slurm-image-config:

#################################
 Provide a Container Image Cache
#################################

When the cluster does not have Internet access or if you want to provide a local cache of container
images to improve performance, you can download the desired container images to a shared directory
and then reference them using file system paths instead of Docker registry references.

There are two mechanisms you can use to reference cached container images depending upon the
container runtime in use.

-  :ref:`referencing-local-image-paths`
-  :ref:`singularity-image-cache`
-  :ref:`manage-singularity-cache`
-  :ref:`manage-enroot-cache`

***********************
 Default Docker Images
***********************

Each version of Determined utilizes specifically-tagged Docker containers. The image tags referenced
by default in this version of Determined are described below.

+-------------+--------------------------------------------------------------------------+
| Environment | File Name                                                                |
+=============+==========================================================================+
| CPUs        | ``determinedai/environments:py-3.8-pytorch-1.12-tf-2.11-cpu-2b7e2a1``    |
+-------------+--------------------------------------------------------------------------+
| NVIDIA GPUs | ``determinedai/environments:cuda-11.3-pytorch-1.12-tf-2.11-gpu-2b7e2a1`` |
+-------------+--------------------------------------------------------------------------+
| AMD GPUs    | ``determinedai/environments:rocm-5.0-pytorch-1.10-tf-2.7-rocm-2b7e2a1``  |
+-------------+--------------------------------------------------------------------------+

See :doc:`/model-dev-guide/prepare-container/set-environment-images` for the images Docker Hub
location, and add each tagged image needed by your experiments to the image cache.

.. _referencing-local-image-paths:

*******************************
 Referencing Local Image Paths
*******************************

Singularity and Podman each support various local container file formats and reference them using a
slightly different syntax. Utilize a cached image by referencing a local path using the experiment
configuration :ref:`environment.image <exp-environment-image>`. When using this strategy, the local
directory needs to be accessible on all compute nodes.

When using Podman, you could save images in OCI archive format to files in a local directory
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

************************************************************
 Configuring an Apptainer/Singularity Image Cache Directory
************************************************************

When using Apptainer/Singularity, you may use :ref:`referencing-local-image-paths` as described
above, or you may instead configure a directory tree of images to be searched. To utilize this
capability, configure a shared directory in :ref:`resource_manager.singularity_image_root
<cluster-configuration-slurm>`. The shared directory needs to be accessible to the launcher and on
all compute nodes. Whenever an image is referenced, it is translated to a local file path as
described in :ref:`environment.image <exp-environment-image>`. If found, the local path is
substituted in the ``singularity run`` command to avoid the need for Singularity to download and
convert the image for each user.

You can manually manage the content of this directory tree, or you may use the
:ref:`manage-singularity-cache <manage-singularity-cache>` script which automates those same steps.
To manually populate the cache, add each tagged image required by your environment and the needs of
your experiments to the image cache using the following steps:

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

.. _manage-singularity-cache:

********************************************************************************
 Managing the Singularity Image Cache using the manage-singularity-cache script
********************************************************************************

A convenience script, ``/usr/bin/manage-singularity-cache``, is provided by the HPC launcher
installation to simplify the management of the Singularity image cache. The script simplifies the
management of the Singularity image cache directory content and helps ensure proper name, placement,
and permissions of content added to the cache. Adding container images to the Singularity image
cache avoids the overhead of downloading the images and allows for sharing of images between
multiple users. It provides the following features:

-  Download the Determined default cuda, cpu, or rocm environment images
-  Download an arbitrary Docker image reference
-  Copy a local Singularity image file into the cache
-  List the currently available images in the cache

If your system has internet access, you can download images directly into the cache. Use the
``--cuda``, ``--cpu``, or ``--rocm`` options to download the current default CUDA, CPU, or ROCM
environment container image into the cache. For example, to download the default CUDA container
image, use the following command:

.. code:: bash

   manage-singularity-cache --cuda

If your system has internet access, you can download any desired Docker container image (e.g.
``determinedai/environments:py-3.8-pytorch-1.10-tf-2.8-cpu-096d730``) into the cache using the
command:

.. code:: bash

   manage-singularity-cache determinedai/environments:py-3.8-pytorch-1.10-tf-2.8-cpu-096d730

Otherwise, from an internet-connected system, download the desired image using the Singularity
``pull`` command, then copy it to a system with access to the ``singularity_image_root`` folder. You
can then add the image to the cache by specifying the local file name using ``-i`` and the Docker
image reference which determines the name to be added to the cache.

.. code:: bash

   manage-singularity-cache -i localfile.sif determinedai/environments:py-3.8-pytorch-1.10-tf-2.8-cpu-096d730

You can view the current set of Docker image names in the cache with the ``-l`` option.

.. code:: bash

   manage-singularity-cache -l
   determinedai/environments:py-3.8-pytorch-1.10-tf-2.8-cpu-096d730
   determinedai/environments:cuda-11.3-pytorch-1.10-tf-2.8-gpu-096d730

.. _manage-enroot-cache:

**********************************************************************
 Managing the Enroot Image Cache using the manage-enroot-cache script
**********************************************************************

This script, ``/usr/bin/manage-enroot-cache``, simplifies the management of a set of shared Enroot
.sqsh file downloads and then creates an Enroot container for use by the current user. It provides
the following features:

-  Download the Determined default cuda, cpu, or rocm environment images
-  Download an arbitrary Docker image reference
-  Share a directory of re-usable imported .sqsh files
-  Optionally, create a per-user container from a shared .sqsh file
-  List the currently available images in the shared .sqsh file cache

When using ``manage-enroot-cache`` you must provide a temporary directory via the ``-s`` option
which is used to download (enroot import) the associated enroot .sqsh file. The .sqsh file is read
by the ``enroot create`` command to generate the container. The directory need only be accessible on
the local host. If the directory you specify is shared with other users, the script will re-use any
downloaded .sqsh files and directly ``enroot create`` an enroot container without needing a separate
download.

Download the shared cache .sqsh file for the current default Determined CUDA and CPU images (enroot
import), and then create the associated containers from them for the current user (``enroot
create``) use the following command:

.. code:: bash

   manage-enroot-cache -s /shared/enroot --cuda --cpu

Download the shared cache .sqsh file for an arbitrary docker image (enroot import), and then create
a container from it for the current user (``enroot create``) use the following command:

.. code:: bash

   manage-enroot-cache -s /shared/enroot determinedai/environments:cuda-10.2-base-gpu-mpi-0.19.4

If you only want the sharable .sqsh file without the overhead of container creation, use the
``--nocreate`` option:

.. code:: bash

   manage-enroot-cache -s /shared/enroot --nocreate determinedai/environments:cuda-10.2-base-gpu-mpi-0.19.4

To optionally configure credentials for image downloads, follow the `enroot documentation
<https://github.com/NVIDIA/enroot/blob/master/doc/cmd/import.md>`__. Specify the user name with the
``--username`` option:

.. code:: bash

   manage-enroot-cache -s /shared/enroot --username <username-here> --cuda --cpu

``--username`` is positional -- if used it should appear before any image reference.

You can view the current set of Docker image names in the cache with the ``-l`` option.

.. code:: bash

   manage-enroot-cache -s /shared/enroot -l
