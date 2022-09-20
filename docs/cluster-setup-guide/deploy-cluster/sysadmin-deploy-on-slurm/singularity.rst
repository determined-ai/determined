.. _slurm-image-config:

####################################
 Provide a Singularity Images Cache
####################################

When the cluster does not have Internet access or if you want to provide a local cache of
Singularity images to be used on the cluster, you can put the Singularity images you want in the
applicable directory under the ``singularity_image_root`` directory. If found, the image reference
is translated to a local file path and substituted in the ``singularity run`` command to avoid the
need for Singularity to download and convert the image for each user.

Each version of Determined utilizes specifically-tagged Docker containers. The image tags referenced
by default in this version of Determined are described below.

***********************
 Default Docker Images
***********************

+-------------+-------------------------------------------------------------------------+
| Environment | File Name                                                               |
+=============+=========================================================================+
| CPUs        | ``determinedai/environments:py-3.8-pytorch-1.10-tf-2.8-cpu-69f397f``    |
+-------------+-------------------------------------------------------------------------+
| Nvidia GPUs | ``determinedai/environments:cuda-11.3-pytorch-1.10-tf-2.8-gpu-69f397f`` |
+-------------+-------------------------------------------------------------------------+
| AMD GPUs    | ``determinedai/environments:rocm-4.2-pytorch-1.9-tf-2.5-rocm-9119094``  |
+-------------+-------------------------------------------------------------------------+

See :doc:`/training/setup-guide/set-environment-images` for the images Docker Hub location and add
each tagged image required by your environment and experiments to the image cache.

************
 Add Images
************

Add each tagged image requried by your environment and the needs of your experiments to the image
cache:

#. Create a directory path using the same prefix as the image name referenced in the
   ``singularity_image_root`` directory. For example, the image
   ``determinedai/environments:cuda-11.3-pytorch-1.10-tf-2.8-gpu-9119094`` is added in the directory
   ``determinedai``.

   .. code:: bash

      cd $singularity_image_root
      mkdir determinedai

#. If your system has internet access, you can download images directly into the cache.

   .. code:: bash

      cd $singularity_image_root
      image="determinedai/environments:cuda-11.3-pytorch-1.10-tf-2.8-gpu-9119094"
      singularity pull $image docker://$image

#. Otherwise, from an internet-connected system, download the desired image using the Singulartity
   pull command then copy it to the ``determinedai`` folder under ``singularity_image_root``.

   .. code:: bash

      singularity pull \
            temporary-image \
            docker://determinedai/environments:cuda-11.3-pytorch-1.10-tf-2.8-gpu-9119094
      scp temporary-image mycluster:$singularity_image_root/determinedai/environments:cuda-11.3-pytorch-1.10-tf-2.8-gpu-9119094
