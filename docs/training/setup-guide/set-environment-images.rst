########################
 Set Environment Images
########################

Determined launches workloads using Docker containers. By default, workloads execute inside a
Determined-provided container that includes common deep learning libraries and frameworks. The
default containers can be found on the Determined Docker Hub with tags for each Determined version:

-  `Default containers for CPU training
   <https://hub.docker.com/r/determinedai/environments/tags?page=1&name=cpu>`__
-  `Default containers for GPU training
   <https://hub.docker.com/r/determinedai/environments/tags?page=1&name=gpu>`__

By default, Determined will use the tag corresponding to your cluster's version. To specify a
different image from this default, update your job configuration to include:

.. code:: bash

   environment:
     image:
       cpu: # full cpu image path e.g. determined/environments/<tag>
       gpu: # full gpu image path e.g. determined/environments/<tag>

If one of the images above contain your required libraries, there is no additional environment
preparation needed.

If you need to add additional customization to the training environment, review the
:doc:`/training/setup-guide/custom-env` page.
