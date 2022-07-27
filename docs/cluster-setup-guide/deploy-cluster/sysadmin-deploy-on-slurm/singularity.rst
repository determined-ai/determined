.. _slurm-image-config:

####################################
 Provide a Singularity Images Cache
####################################

When the cluster does not have Internet access or if you want to provide a local cache of
Singularity images to be used on the cluster, you can put the Singularity images you want in the
applicable directory under the ``singularity_image_root`` directory. If found, the image reference
is translated to a local file path and substituted in the ``singularity run`` command to avoid the
need for Singularity to download and convert the image.

Each version of Determined utilizes specifically-tagged Docker containers. The image tags referenced
by default in this version of Determined are described below.

*********************
Default Docker Images
*********************

+-------------+---------------------------------------------------------------------------------------+
| Environment | File Name                                                                             |
+=============+=======================================================================================+
| CPUs        | ``determinedai/environments:py-3.8-pytorch-1.10-lightning-1.5-tf-2.8-cpu-3e933ea``    |
+-------------+---------------------------------------------------------------------------------------+
| Nvidia GPUs | ``determinedai/environments:cuda-11.3-pytorch-1.10-lightning-1.5-tf-2.8-gpu-3e933ea`` |
+-------------+---------------------------------------------------------------------------------------+
| AMD GPUs    | ``determinedai/environments:rocm-4.2-pytorch-1.9-tf-2.5-rocm-3e933ea``                |
+-------------+---------------------------------------------------------------------------------------+

See :doc:`/training/setup-guide/overview` for the Docker Hub location of these images.

**********
Add Images
**********

Add each tagged image requried by your environment and the needs of your experiments to the image cache:

#. Create a directory path using the same prefix as the image name referenced in the
   ``singularity_image_root`` directory. For example, the image
   ``determinedai/environments:cuda-11.3-pytorch-1.10-lightning-1.5-tf-2.8-gpu-6e45071`` is added in
   the directory ``determinedai``.

   .. code:: bash

      cd $singularity_image_root
      mkdir determinedai

#. From an internet-connected system, download the image you want, such as
   ``determinedai/environments:cuda-11.3-pytorch-1.10-lightning-1.5-tf-2.8-gpu-6e45071` using the
   ``singularity pull`` command.

   .. code:: bash

      singularity pull \
            environments:cuda-11.3-pytorch-1.10-lightning-1.5-tf-2.8-gpu-6e45071 \
            determinedai/environments:cuda-11.3-pytorch-1.10-lightning-1.5-tf-2.8-gpu-6e45071

   This puts the ``environments:cuda-11.3-pytorch-1.10-lightning-1.5-tf-2.8-gpu-6e45071``
   Singularity image in the current directory.

#. Put the ``environments:cuda-11.3-pytorch-1.10-lightning-1.5-tf-2.8-gpu-6e45071`` downloaded
   Singularity image in the ``determinedai`` folder under ``singularity_image_root``.
