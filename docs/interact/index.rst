#######################
 Interact with Cluster
#######################

This section describes the mechanisms available for interacting with a cluster.

*****************
 WebUI Interface
*****************

Typically, you need to use the WebUI and the CLI.

The WebUI allows users to create and monitor the progress of experiments. It is accessible by
visiting ``http://master-addr:8080``, where ``master-addr`` is the hostname or IP address where the
Determined master is running.

*****
 CLI
*****

The :ref:`command-line interface (CLI) <cli>` is distributed as a Python wheel package. After the
wheel is installed, use the CLI ``det`` command to interact with the cluster.

************
 Automation
************

Python and REST APIs are provided for programmatic interfaces. The :ref:`Python API <client>`
defines a Pythonic way to access the cluster. The :ref:`REST API <rest-api>` is another way to
programmatically interact with a cluster.

.. toctree::
   :maxdepth: 1
   :hidden:

   cli
   api-experimental-client
   rest-apis
