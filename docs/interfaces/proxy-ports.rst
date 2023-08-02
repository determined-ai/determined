.. _proxy-ports:

#######################
 Exposing Custom Ports
#######################

Determined allows you to expose a custom network port in a task container, and access it using a
local tunnel.

For multi-container tasks, such as distributed training experiments, only the ports on the chief
container (``rank=0``) will be exposed.

***************
 Configuration
***************

First, specify the ports in the ``environments -> proxy_ports`` section of the experiment or task
config, for example:

.. code:: yaml

   environment:
     proxy_ports:
       - proxy_port: 8265
         proxy_tcp: true

Launch your task or experiment normally. Then, use the ``det`` CLI to start a tunnel. Running the
following command will setup a tunnel proxying ``localhost:8265`` to port ``8265`` in the task
container.

.. code:: bash

   python -m determined.cli.tunnel --listener 8265 --auth $DET_MASTER $TASK_ID:8265

where $DET_MASTER is your Determined master address, and $TASK_ID is the task id of the launched
task or experiment. You can look up the task id using CLI command ``det task list``.

Alternatively, you can use a shortcut which allows to launch the experiment, follow its logs, and
run the tunnel all at once:

.. code:: bash

   det e create config_file.yaml model_def -f -p 8265

Unauthenticated Mode
====================

Optionally, you can run a tunnel with determined authentication turned off. This mode may be useful
when the proxied app is handling security by itself, such as a web app protected by username and
password. To use it,

#. Add ``unauthenticated: true`` option in the task config.
#. Omit ``--auth`` option from the tunnel CLI.

.. code:: yaml

   environment:
     proxy_ports:
       - proxy_port: 8265
         proxy_tcp: true
         unauthenticated: true

.. code:: bash

   python -m determined.cli.tunnel --listener 8265 $DET_MASTER $TASK_ID:8265
