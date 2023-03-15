.. _commands-and-shells:

#####################
 Commands and Shells
#####################

In addition to structured model training workloads handled using :ref:`experiments <experiments>`,
Determined also supports free-form tasks using *commands* and *shells*. Commands and shells enable
you to use a Determined cluster and cluster GPUs without writing code that conforms to the trial
APIs.

Commands execute a user-specified program on the cluster. Commands are useful for running existing
code in batch mode.

Shells start SSH servers that let you use cluster resources interactively. Shells provide access to
the cluster in the form of interactive `SSH <https://en.wikipedia.org/wiki/SSH_(Secure_Shell)>`_
sessions.

This document describes the most common CLI and shell commands.

**********
 Commands
**********

CLI commands start with ``det command``, abbreviated as ``det cmd``. The main subcommand is ``det
cmd run``, which runs a command in the cluster and streams its output. For example, the following
CLI command uses ``nvidia-smi`` to display information about the GPUs available to tasks in the
container:

.. code::

   det cmd run nvidia-smi

You can also run more complex commands including shell constructs provided they are quoted to
prevent interpretation by the local shell:

.. code::

   det cmd run 'for x in a b c; do echo $x; done'

``det cmd run`` streams output from the command until it finishes, but the command continues
executing and occupying cluster resources even if the CLI is interrupted or killed, such as due to
entering ``Ctrl-C``. To stop the command or view additional output, you need the command UUID, which
you can get from the output of the original ``det cmd run`` or ``det cmd list``. After you have the
UUID, run

-  ``det cmd logs <UUID>`` to view a snapshot of logs.
-  ``det cmd logs -f <UUID>`` to view the current logs and continue streaming future output.
-  ``det cmd kill <UUID>`` to stop the command.

.. _install-cli:

Installation
============

The CLI is distributed as a Python wheel package. Each user should install a copy of the CLI on
their local development machine.

The CLI requires Python >= 3.7. For best results, install the CLI into a `virtualenv
<https://virtualenv.pypa.io/en/latest/>`__. To install the CLI into a virtualenv, activate the
virtualenv before installing the CLI using the pip utility.

Install the CLI using the ``pip`` utility:

.. code::

   pip install determined

After installing the CLI, configure it to connect to the Determined master at the appropriate IP
address. To do this, set the ``DET_MASTER`` environment variable:

.. code::

   export DET_MASTER=<master IP>

You might want to place this into the appropriate configuration file for your login shell, such as
``.bashrc``.

Usage
=====

After the wheel is installed, the CLI is invoked with the ``det`` command. Use ``det --help`` for
more information about the individual CLI commands.

CLI subcommands usually follow a ``<noun> <verb>`` form, similar to the paradigm of `ip
<http://www.policyrouting.org/iproute2.doc.html>`__. Certain abbreviations are supported, and a
missing verb is the same as ``list``, when possible.

For example, the different commands within each block below all do the same thing:

.. code:: bash

   # List all experiments.
   $ det experiment list
   $ det e list
   $ det e

.. code:: bash

   # List all agents.
   $ det agent list
   $ det a list
   $ det a

.. code:: bash

   # List all slots.
   $ det slot list
   $ det slot
   $ det s

For a complete description of the available nouns and abbreviations, see the output of ``det help``.
Each noun also provides a ``help`` verb that describes the possible verbs for that noun. Or, you can
provide the ``-h`` or ``--help`` argument anywhere, which causes the CLI to exit after printing a
help message for the object or action specified to that point.

Environment Variables
=====================

-  ``DET_MASTER``: The network address of the master of the Determined installation. The value can
   be overridden using the ``-m`` flag.

-  ``DET_USER`` and ``DET_PASS``: Specifies the current Determined user and password for use when
   non-interactive behaviour is required such as scripts. ``det user login`` is preferred for normal
   usage. Both ``DET_USER`` and ``DET_PASS`` must be set together to take effect. These variables
   can be overridden by using the ``-u`` flag.

Examples
========

+-------------------------------------------+----------------------------------------------------+
| Commands(s)                               | Description                                        |
+===========================================+====================================================+
| ``det e`` |br| ``det experiment`` |br|    | Show information about experiments in the cluster. |
| ``det experiment list``                   |                                                    |
+-------------------------------------------+----------------------------------------------------+
| ``det -m 1.2.3.4 e`` |br|                 | Show information about experiments in the cluster  |
| ``DET_MASTER=1.2.3.4 det e``              | at network address ``1.2.3.4``.                    |
+-------------------------------------------+----------------------------------------------------+
| ``det t logs -f 289``                     | Show the logs for trial 289 and continue showing   |
|                                           | new logs as they arrive.                           |
+-------------------------------------------+----------------------------------------------------+
| ``det e label add 17 foobar``             | Add the label ``foobar`` to experiment 17.         |
+-------------------------------------------+----------------------------------------------------+
| ``det e describe 493 --metrics --csv``    | Display information about experiment 493,          |
|                                           | including full metrics, in CSV format.             |
+-------------------------------------------+----------------------------------------------------+
| ``det e create -f --paused const.yaml .`` | Create an experiment with the configuration file   |
|                                           | ``const.yaml`` and the code contained in the       |
|                                           | current directory. The experiment is created in a  |
|                                           | paused state, which means that it is not scheduled |
|                                           | on the cluster until it is activated.              |
+-------------------------------------------+----------------------------------------------------+
| ``det e set max-slots 85 4``              | Ensure that experiment 85 does not use more than 4 |
|                                           | slots in the cluster.                              |
+-------------------------------------------+----------------------------------------------------+
| ``det u create --admin hoid``             | Create a new user named ``hoid`` who has admin     |
|                                           | privileges.                                        |
+-------------------------------------------+----------------------------------------------------+
| ``det version``                           | Show detailed information about the CLI and        |
|                                           | master. This command does not take both an object  |
|                                           | and an action.                                     |
+-------------------------------------------+----------------------------------------------------+

.. |br| raw:: html

   <br />

********
 Shells
********

Shell-related CLI commands start with ``det shell``. To start a persistent SSH server container in
the Determined cluster and connect an interactive session to it, use ``det shell start``:

.. code::

   det shell start

After starting a server with ``det shell start``, you can make another independent connection to the
same server by running ``det shell open <UUID>``. You can get the UUID from the output of the
original ``det shell start`` or ``det shell list`` command:

.. code::

   $ det shell list
    Id                                   | Owner      | Description                  | State   | Exit Status
   --------------------------------------+------------+------------------------------+---------+---------------
    d75c3908-fb11-4fa5-852c-4c32ed30703b | determined | Shell (annually-alert-crane) | RUNNING | N/A
   $ det shell open d75c3908-fb11-4fa5-852c-4c32ed30703b

Optionally, you can provide extra options to pass to the SSH client when using ``det shell start``
or ``det shell open`` by including them after ``--``. For example, this command starts a new shell
and forwards a port from the local machine to the container:

.. code::

   det shell start -- -L8080:localhost:8080

To stop the SSH server container and free cluster resources, run ``det shell kill <UUID>``.

.. _cli:

****************************************
 Command-line Interface (CLI) Reference
****************************************

.. code:: bash

   usage: det [-h] [-u username] [-m address] [-v] command ...

   Determined command-line client

   positional arguments:
     command
       help                show help for this command
       auth                manage auth
       agent (a)           manage agents
       command (cmd)       manage commands
       checkpoint (c)      manage checkpoints
       deploy (d)          manage deployments
       experiment (e)      manage experiments
       job (j)             manage job
       master (m)          manage master
       model (m)           manage models
       notebook            manage notebooks
       oauth               manage OAuth
       preview-search      preview search
       resources (res)     query historical resource allocation
       shell               manage shells
       slot (s)            manage slots
       task                manage tasks (commands, experiments, notebooks,
                           shells, tensorboards)
       template (tpl)      manage config templates
       tensorboard         manage TensorBoard instances
       trial (t)           manage trials
       user (u)            manage users
       version             show version information

   optional arguments:
     -h, --help            show this help message and exit
     -u username, --user username
                           run as the given user (default: None)
     -m address, --master address
                           master address (default: localhost:8080)
     -v, --version         print CLI version and exit
