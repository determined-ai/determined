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

Then launch your task or experiment normally. Once it's up, use the ``det`` CLI to start a tunnel.
Running this command will setup a tunnel proxying ``localhost:8265`` to port ``8265`` in the task
container.

.. code:: bash

   det -m determined.cli.tunnel --listener 8265 $DET_MASTER $TASK_ID:8265

where $DET_MASTER is your Determined master address, and $TASK_ID is the task id of the launched
task or experiment. You can look up the task id using CLI command ``det task list``.

Alternatively, we provide a shortcut which allows to launch the experiment, follow its logs, and run
the tunnel all at once:

.. code:: bash

   det e create config_file.yaml model_def -f -p 8265
