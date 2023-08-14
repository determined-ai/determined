.. _custom-env:

##############################
 Customizing Your Environment
##############################

Determined launches workloads using Docker containers. By default, workloads execute inside a
Determined-provided container that includes common deep learning libraries and frameworks.

If your model code has additional dependencies, the easiest way to install them is to specify a
:ref:`startup hook <startup-hooks>`. For more complex dependencies, use a :ref:`custom Docker image
<custom-docker-images>`.

If you are using Determined on Kubernetes, review the :ref:`Custom Pod Specs <custom-pod-specs>`
guide.

.. _environment-variables:

***********************
 Environment Variables
***********************

For both trial runners and commands, you can configure the environment variables inside the
container using the :ref:`experiment <experiment-configuration>` or :ref:`task
<command-notebook-configuration>` ``environment.environment_variables`` configuration field. The
format is a list of ``NAME=VALUE`` strings. For example:

.. code:: yaml

   environment:
     environment_variables:
       - A=hello world
       - B=$A
       - C=${B}

Variables are set sequentially, which affect variables that depend on the expansion of other
variables. In the example, names ``A``, ``B``, and ``C`` each have the value ``hello_world`` in the
container.

Proxy variables set in this way take precedent over variables set in the :ref:`agent configuration
<agent-config-reference>`.

You can also set variables for each accelerator type, separately:

.. code:: yaml

   environment:
     environment_variables:
       cpu:
         - A=hello x86
       gpu:
         - A=hello nvidia
       rocm:
         - A=hello amd

.. _startup-hooks:

***************
 Startup Hooks
***************

If a ``startup-hook.sh`` file exists in the top level of your model definition directory, this file
is automatically run with every Docker container startup. This occurs before any Python interpreters
are launched or deep learning operations are performed. The startup hook can be used to customize
the container environment, install additional dependencies, and download data sets among other shell
script commands.

Startup hooks are not cached and run before the start of every workload, so expensive or
long-running operations in a startup hook can result in poor performance.

This example startup hook installs the ``wget`` utility and the ``pandas`` Python package:

.. code:: bash

   apt-get update && apt-get install -y wget
   python3 -m pip install pandas

This :download:`Iris example </examples/iris_tf_keras.tgz>` contains a TensorFlow Keras model that
uses a startup hook to install an additional Python dependency.

.. _container-images:

******************
 Container Images
******************

Officially supported, default Docker images are provided to launch containers for experiments,
commands, and other workflows.

All trial runner containers are launched with additional Determined-specific harness code, which
orchestrates model training and evaluation in the container. Trial runner containers are also loaded
with the experiment's model definition and hyperparameter values for the current trial.

GPU-specific versions of each library are automatically selected when running on agents with GPUs.

.. _default-environment:

Default Images
==============

+-------------+-----------------------------------------------------------------------------------+
| Environment | File Name                                                                         |
+=============+===================================================================================+
| CPUs        | ``determinedai/environments:py-3.8-pytorch-1.12-tf-2.11-cpu-0.24.0``              |
+-------------+-----------------------------------------------------------------------------------+
| NVIDIA GPUs | ``determinedai/environments:cuda-11.3-pytorch-1.12-tf-2.11-gpu-0.24.0``           |
+-------------+-----------------------------------------------------------------------------------+
| AMD GPUs    | ``determinedai/environments:rocm-5.0-pytorch-1.10-tf-2.7-rocm-0.24.0``            |
+-------------+-----------------------------------------------------------------------------------+

.. _custom-docker-images:

Custom Images
=============

While the official images contain all the dependencies needed for basic deep learning workloads,
many workloads have additional dependencies. If the extra dependencies are quick to install, you
might consider using a :ref:`startup hook <startup-hooks>`. Where installing dependencies using
``startup-hook.sh`` takes too long, it is recommended that you build your own Docker image and
publish to a Docker registry, such as `Docker Hub <https://hub.docker.com/>`__.

.. warning::

   Do NOT install TensorFlow, PyTorch, Horovod, or Apex packages, which conflict with
   Determined-installed packages.

It is recommended that custom images use one of the official Determined images as a base image,
using the ``FROM`` instruction.

Example Dockerfile that installs custom ``conda``-, ``pip``-, and ``apt``-based dependencies:

.. code:: bash

   # Determined Image
   FROM determinedai/environments:cuda-11.3-pytorch-1.12-tf-2.11-gpu-0.24.0

   # Custom Configuration
   RUN apt-get update && \
      DEBIAN_FRONTEND="noninteractive" apt-get -y install tzdata && \
      apt-get install -y unzip python-opencv graphviz
   COPY environment.yml /tmp/environment.yml
   COPY pip_requirements.txt /tmp/pip_requirements.txt
   RUN conda env update --name base --file /tmp/environment.yml
   RUN conda clean --all --force-pkgs-dirs --yes
   RUN eval "$(conda shell.bash hook)" && \
      conda activate base && \
      pip install --requirement /tmp/pip_requirements.txt

Assuming that this image is published to a public repository on Docker Hub, use the following
declaration format to configure an experiment, command, or notebook:

.. code:: yaml

   environment:
     image: "my-user-name/my-repo-name:my-tag"

where ``my-user-name`` is your Docker Hub user, ``my-repo-name`` is the name of the Docker Hub
repository, and ``my-tag`` is the image tag to use, such as ``latest``.

If you publish your image to a private Docker Hub repository, you can specify the credentials needed
to access the repository:

.. code:: yaml

   environment:
     image: "my-user-name/my-repo-name:my-tag"
     registry_auth:
       username: my-user-name
       password: my-password

If you publish the image to a private `Docker Registry <https://docs.docker.com/registry/>`__,
specify the registry path as part of the ``image`` field:

.. code:: yaml

   environment:
     image: "myregistry.local:5000/my-user-name/my-repo-name:my-tag"

Images are fetched using HTTPS by default. An HTTPS proxy can be configured using the
``https_proxy`` field in the :ref:`agent configuration <agent-config-reference>`.

The custom image and credentials can be set as the defaults for all tasks launched in Determined,
using the ``image`` and ``registry_auth`` fields in the :ref:`master configuration
<master-config-reference>`. Make sure to restart the master for this to take effect.

.. _virtual-env:

**********************
 Virtual Environments
**********************

Model developers commonly use virtual environments. The following example configures virtual
environments using :ref:`custom images <custom-docker-images>`:

.. code:: bash

   # Determined Image
   FROM determinedai/environments:py-3.8-pytorch-1.12-tf-2.11-cpu-0.24.0

   # Create a virtual environment
   RUN conda create -n myenv python=3.8
   RUN eval "$(conda shell.bash hook)" && \
      conda activate myenv && \
      pip install scikit-learn

   # Set the default virtual environment
   RUN echo 'eval "$(conda shell.bash hook)" && conda activate myenv' >> ~/.bashrc

To ensure that a virtual environment is activated every time a new interactive terminal session is
created, in JupyterLab or using Determined Shell, update ``~/.bashrc`` with the scripts to activate
the virtual environment you want.

This example switches to a virtual environment using a :ref:`startup hook <startup-hooks>`:

.. code:: bash

   # Switch to the desired virtual environment
   eval "$(conda shell.bash hook)"
   conda activate myenv

   # Do that for every new interactive terminal session
   echo 'eval "$(conda shell.bash hook)" && conda activate myenv' >> ~/.bashrc
