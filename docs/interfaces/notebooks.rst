.. _notebooks:

###################
 Jupyter Notebooks
###################

You can use `Jupyter Notebooks <https://jupyter.org/>`__ to conveniently develop and debug machine
learning models, visualize the behavior of trained models, and manage the training lifecycle of a
model manually. Determined makes launching and managing notebooks easy.

Determined Notebooks provide the following benefits:

-  Jupyter Notebooks run in containerized environments on the cluster. This makes it easy to manage
   dependencies using images and virtual environments. The HTTP requests are passed through the
   master proxy from and to the container.

-  Jupyter Notebooks can be automatically terminated if they are idle for a configurable duration to
   release resources. A notebook instance is considered to be idle if it is not receiving any HTTP
   traffic and it is not otherwise active (as defined by the ``notebook_idle_type`` option in the
   :ref:`task configuration <command-notebook-configuration>`). To enable this behavior by default,
   set ``notebook_timeout`` :ref:`option in your master config <master-config-notebook-timeout>`. To
   enable it for a particular notebook, set ``idle_timeout`` option in the notebook config.

After a Notebook is terminated, it is not possible to restore the files that are not stored in the
persistent directories. **It is important to configure the cluster to mount persistent directories
into the container and save files in the persistent directories in the container.** See
:ref:`notebook-state` for more information.

If you open a Notebook tab in JupyterLab, a kernel is automatically opened. This kernel will not be
shut down automatically, so you'll need to manually terminate it.

There are two ways to access notebooks in Determined: the :ref:`CLI <cli-ug>` and the :ref:`WebUI
<web-ui-if>`. To install the CLI, see :ref:`install-cli`.

**************
 Command Line
**************

The following command will automatically start a notebook with a single GPU and open it in your
browser.

.. code::

   $ det notebook start
   Scheduling notebook unique-oyster (id: 5b2a9ea4-a6bb-4d2b-b42b-25e4064a3220)...
   [DOCKER BUILD 🔨] Step 1/11 : FROM nvidia/cuda:9.0-cudnn7-runtime-ubuntu16.04
   [DOCKER BUILD 🔨]
   [DOCKER BUILD 🔨]  ---> 9918ba890dca
   [DOCKER BUILD 🔨] Step 2/11 : RUN rm /etc/apt/sources.list.d/*
   ...
   [DOCKER BUILD 🔨] Successfully tagged nvidia/cuda:9.0-cudnn7-runtime-ubuntu16.04-73bf63cc864088137a477ce62f39ffe8
   [Determined] 2019-04-04T17:53:22.076591700Z [I 17:53:22.075 NotebookApp] Writing notebook server cookie secret to /root/.local/share/jupyter/runtime/notebook_cookie_secret
   [Determined] 2019-04-04T17:53:23.067911400Z [W 17:53:23.067 NotebookApp] All authentication is disabled.  Anyone who can connect to this server will be able to run code.
   [Determined] 2019-04-04T17:53:23.073644300Z [I 17:53:23.073 NotebookApp] Serving notebooks from local directory: /
   disconnecting websocket
   Jupyter Notebook is running at: http://localhost:8080/proxy/5b2a9ea4-a6bb-4d2b-b42b-25e4064a3220-notebook-0/lab/tree/Notebook.ipynb?reset

After the notebook has been scheduled onto the cluster, the Determined CLI will open a web browser
window pointed to that notebook's URL. Back in the terminal, you can use the ``det notebook list``
command to see that this notebook is one of those currently ``RUNNING`` on the Determined cluster:

.. code::

   $ det notebook list
    Id                                   | Entry Point                                            | Registered Time              | State
   --------------------------------------+--------------------------------------------------------+------------------------------+---------
    0f519413-2411-4b3c-adbc-9b1b60c96156 | ['jupyter', 'notebook', '--config', '/etc/jupyter.py'] | 2019-04-04T17:52:48.1961129Z | RUNNING
    5b2a9ea4-a6bb-4d2b-b42b-25e4064a3220 | ['jupyter', 'notebook', '--config', '/etc/jupyter.py'] | 2019-04-04T17:53:20.387903Z  | RUNNING
    66da599e-62d2-4c2d-91c4-01a04045e4ab | ['jupyter', 'notebook', '--config', '/etc/jupyter.py'] | 2019-04-04T17:52:58.4573214Z | RUNNING

The ``--context`` option adds a folder or file to the notebook environment, allowing its contents to
be accessed from within the notebook.

.. code::

   det notebook start --context folder/file

The ``--config-file`` option can be used to create a notebook with an environment specified by a
configuration file.

.. code::

   det notebook start --config-file config.yaml

For more information on how to write the notebook configuration file, see
:ref:`notebook-configuration`.

*********************
 Useful CLI Commands
*********************

A full list of notebook-related commands can be found by running:

.. code::

   det notebook --help

To view all running notebooks:

.. code::

   det notebook list

To kill a notebook, you need its ID, which can be found using the ``list`` command.

.. code::

   det notebook kill <id>

*******
 WebUI
*******

You can also start a Notebook from the WebUI. To do this, go to the **Tasks** pane and then select
**Launch JupyterLab**.

.. note::

   Depending on your particular setup, you can select the appropriate resource pool when creating a
   new notebook.

.. image:: /assets/images/launch-cpu-notebook@2x.jpg
   :width: 100%
   :alt: Determined AI model training interactive WebUI where you can launch a new Jupyter Notebook.

|

The WebUI displays a list of tasks running on the cluster including running notebooks. You can
reopen, kill, or view logs for each notebook.

You can customize the keyboard shortcut you use to launch a JupyterLab Notebook. To do this, visit
the Shorcuts settings by selecting your profile name in the upper left corner and choosing
**Settings**.

.. _notebook-configuration:

************************
 Notebook Configuration
************************

Notebooks can be passed a notebook configuration option to control the notebook environment. For
example, to launch a notebook that uses two GPUs:

.. code::

   $ det notebook start --config resources.slots=2

Alternatively, a YAML file can also be used to configure the notebook, using the ``--config-file``
option:

.. code::

   $ cat > config.yaml <<EOL
   description: test-notebook
   resources:
     slots: 2
   bind_mounts:
     - host_path: /data/notebook_scratch
       container_path: /scratch
   idle_timeout: 30m
   EOL
   $ det notebook start --config-file config.yaml

See :ref:`command-notebook-configuration` for details on the supported configuration options.

Finally, to configure notebooks to run a predefined set of commands at startup, you can include a
:ref:`startup hook <startup-hooks>` in a directory specified with the ``--context`` option:

.. code::

   $ mkdir my_context_dir
   $ echo "pip3 install pandas" > my_context_dir/startup-hook.sh
   $ det notebook start --context my_context_dir

.. _cpu-only-notebooks:

Example: CPU-Only Notebooks

By default, each notebook is assigned a single GPU. This is appropriate for some uses of notebooks
(e.g., training a deep learning model) but unnecessary for other tasks (e.g., analyzing the training
metrics of a previously trained model). To launch a notebook that does not use any GPUs, set
``resources.slots`` to ``0``:

.. code::

   $ det notebook start --config resources.slots=0

.. _notebook-state:

*********************************
 Save and Restore Notebook State
*********************************

.. warning::

   It is only possible to save and restore notebook state on Determined clusters that are configured
   with a shared filesystem available to all agents.

To ensure that your work is saved even if your notebook gets terminated, it is recommended to launch
all notebooks with a shared filesystem directory *bind-mounted* into the notebook container and work
on files inside of the bind mounted directory.

For example, a user ``jimmy`` with a shared filesystem home directory at ``/shared/home/jimmy``
could use the following configuration to launch a notebook:

.. code::

   $ cat > config.yaml << EOL
   bind_mounts:
     - host_path: /shared/home/jimmy
       container_path: /shared/home/jimmy
   EOL
   $ det notebook start --config-file config.yaml

By default, launching a cluster by ``det deploy gcp up``, ``det deploy aws --deployment-type efs``,
or ``det deploy aws --deployment-type fsx`` creates a Network file system that is shared by all the
agents and is automatically mounted into Notebook containers at
``/run/determined/workdir/shared_fs/``.

To launch a notebook with ``det deploy local cluster-up``, a user can add the ``--auto-bind-mount``
flag, which mounts the user's home directory into the task containers by default:

.. code::

   $ det deploy local cluster-up --auto-bind-mount="/shared/home/jimmy"
   $ det notebook start

Working on a notebook file within the shared bind mounted directory will ensure that your code and
Jupyter checkpoints are saved on the shared filesystem rather than an ephemeral container
filesystem. If your notebook gets terminated, launching another notebook and loading the previous
notebook file will effectively restore the session of your previous notebook. To restore the *full*
notebook state (in addition to code), you can use Jupyter's ``File`` > ``Revert to Checkpoint``
functionality.

.. note::

   By default, JupyterLab will take a checkpoint every 120 seconds in an ``.ipynb_checkpoints``
   folder in the same directory as the notebook file. To modify this setting, click on ``Settings``
   > ``Advanced Settings Editor`` and change the value of ``"autosaveInternal"`` under ``Document
   Manager``.

*************************************
 Use the Determined CLI in Notebooks
*************************************

The :ref:`Determined CLI <cli-ug>` is installed into notebook containers by default. This allows you
to interact with Determined from inside a notebook---e.g., launch new deep learning workloads or
examine the metrics from an active or historical Determined experiment. For example, to list
Determined experiments from inside a notebook, run the notebook command ``!det experiment list``.
