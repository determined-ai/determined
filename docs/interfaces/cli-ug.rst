.. _cli-ug:

################
 Determined CLI
################

To use Determined, you'll need, at minimum, the Determined Command-Line Interface (CLI) and a
Determined cluster. The Determined CLI includes the ``det`` command line tools for interacting with
a Determined cluster. This page contains instructions for using the CLI, including installion and
upgrade.

.. warning::

   Although Determined supports password-based authentication, communication between the Determined
   CLI, Determined WebUI, and the Determined master does not take place over an encrypted channel by
   default.

.. note::

   All users should install the Determined CLI on their local development machine.

.. note::

   You can also interact with Determined using the :ref:`web-ui-if`.

.. _install-cli:

**************
 Installation
**************

The CLI is distributed as a Python wheel package. The CLI requires Python >= 3.7. For best results,
install the CLI into a `virtualenv <https://virtualenv.pypa.io/en/latest/>`__. To install the CLI
into a virtualenv, activate the virtualenv before installing the CLI using the pip utility.

Install the CLI using the ``pip`` utility:

.. code::

   pip install determined

After installing the CLI, configure it to connect to the Determined master at the appropriate IP
address. To do this, set the ``DET_MASTER`` environment variable:

.. code::

   export DET_MASTER=<master IP>

Place this into the appropriate configuration file for your login shell, such as ``.bashrc``.

Verifying Installation
======================

To verify that the Determined CLI has been installed correctly, use the following command:

.. code:: bash

   det --version

This command displays the installed version of the Determined CLI. If the installation was
successful, you should see the version number in the output.

Uninstalling
============

If you need to uninstall the Determined CLI, use the following command:

.. code:: bash

   pip uninstall determined-cli

This command uninstalls the Determined CLI from your system.

Upgrading
=========

To upgrade the Determined CLI to the latest version, use the following command:

.. code:: bash

   pip install --upgrade determined-cli

This command upgrades the Determined CLI to the latest available version.

*****************
 Getting Started
*****************

After installing the Determined CLI, you can start using it to interact with your Determined
cluster. The CLI is invoked with the ``det`` command.

CLI subcommands usually follow a ``<noun> <verb>`` form, similar to the paradigm of `ip
<http://www.policyrouting.org/iproute2.doc.html>`__. Certain abbreviations are supported, and a
missing verb is the same as ``list``, when possible. The following examples show different ways to
achieve the same outcome using the ``<noun><verb>`` form, followed by the abbreviation, followed by
a missing ``<verb>``:

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

***********
 CLI Usage
***********

For a comprehensive list of nouns and abbreviations, use ``det help`` or ``det -h``. Each noun has a
``help`` verb detailing its associated verbs.

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

Syntax
======

To use the CLI tool, follow the proper syntax.

``det [-h] [-u username] [-m address] [-v] command ...``

-  det: This is the main command you'll use for interacting with the Determined AI CLI.

-  [-h]: The square brackets indicate that this is an optional argument. ``-h``or ``--help`` can be
   used to display a help message and exit. If you need information about a specific command, add
   the ``-h`` flag after the ``det`` command.

-  [-u username]: Another optional argument, ``-u`` or ``--user`` allows you to run the command as a
   specific user. Replace username with the desired username. For example, to run a command as user
   "abbie", you would use ``det -u abbie`` command.

-  [-m address]: This optional argument, ``-m`` or ``--master``, lets you specify the master address
   for the Determined cluster. Replace address with the actual address of the master, e.g.,
   ``localhost:8080``.

-  [-v]: The ``-v`` or ``--version`` flag is another optional argument that you can use to print the
   CLI version and exit.

-  command: This represents the specific subcommand you want to execute such as ``list``, ``pause``,
   ``logs``, or ``kill``. You'll replace command with the actual command you want to run.

-  ...: The ellipsis signifies that you can provide additional arguments, options, or values,
   depending on the subcommand you choose.

Usage Examples
==============

.. list-table::
   :header-rows: 1
   :widths: 25 35 25 15

   -  -  Task
      -  Example
      -  Command
      -  Options

   -  -  List all experiments
      -  Display a list of all experiments in the cluster.
      -  ``det experiment list``
      -

   -  -  List all experiments for a specific network address.
      -  Display a list of all experiments in the cluster at network address ``1.2.3.4``.
      -  ``det -m 1.2.3.4 e``
      -

   -  -  View a snapshot of logs
      -  Display the most recent logs for a specific command.
      -  ``det command logs <command_id>``
      -  -f, --tail

   -  -  View logs for a trial.
      -  Show the logs for trial 289 and continue streaming logs in real-time.
      -  ``det t logs -f 289``
      -  -f

   -  -  Add a label
      -  Add the label ``foobar`` to experiment 17.
      -  ``det e label add 17 foobar``
      -

   -  -  Create an experiment

      -  Create an experiment in a paused state with the configuration file ``const.yaml`` and the
         code contained in the current directory. The paused experiment is not scheduled on the
         cluster until activated.

      -  ``det e create -f --paused const.yaml .``

      -

   -  -  Describe an experiment
      -  Display information about experiment 493, including full metrics, in CSV format.
      -  ``det e describe 493 --metrics --csv``
      -

   -  -  Set max slots
      -  Ensure that experiment 85 does not use more than 4 slots in the cluster.
      -  ``det e set max-slots 85 4``
      -

   -  -  Display details about the CLI and master
      -  Show detailed information about the CLI and master. This command does not take both an
         object and an action.
      -  ``det version``
      -

   -  -  Stop (kill) the command
      -  Terminate a running command.
      -  ``det command kill <command_id>``
      -

   -  -  Set a password for the admin user
      -  Set the password for the admin user during cluster setup.
      -  ``det user change-password admin``
      -

   -  -  Create a user
      -  Create a new user named ``hoid`` who has admin privileges.
      -  ``det u create --admin hoid``
      -

***********************
 Environment Variables
***********************

-  ``DET_MASTER``: The network address of the master of the Determined installation. The value can
   be overridden using the ``-m`` flag.

-  ``DET_USER`` and ``DET_PASS``: Specifies the current Determined user and password for use when
   non-interactive behaviour is required such as scripts. ``det user login`` is preferred for normal
   usage. Both ``DET_USER`` and ``DET_PASS`` must be set together to take effect. These variables
   can be overridden by using the ``-u`` flag.

**************
 Getting Help
**************

Using the ``-h`` or ``--help`` argument on objects or actions prints a help message and exits the
CLI. For example, to print usage for the ``deploy`` command, run the following:

.. code:: bash

   det deploy -h

Similarly, you can get help for a subcommand. For example, to get help for ``deploy aws``:

.. code:: bash

   det deploy aws -h

Use Case: Getting Help for ``experiment`` Command
=================================================

Let's say we want to discover how to download the checkpoint with the best validation metric for a
specific trial. We first want to know how to get our trial ID.

-  To find this information, we'll use the ``-h`` option with the ``experiment`` command.

.. code:: bash

   det experiment -h

From the help output, we can see that the ``list`` or ``ls`` command provides a list of experiments.

-  To get usage information for this command, we'll run the following:

.. code:: bash

   det experiment ls -h

-  From the help output, we can see that the ``all`` or ``a`` option shows all experiments.
-  Now we can run the command to list all experiments including the experiment ID which is the same
   as the trial ID.

.. code:: bash

   det experiment ls -a

The CLI tool prints a list of all experiments along with the ID for each experiment. Let's say the
experiment we want to download the checkpoint for has an ID of ``5``.

-  Now that we have our experiment ID, we want to get usage information for the ``download``
   subcommand of the ``trial`` command:

.. code:: bash

   det trial download -h

The CLI prints usage information for the subcommand.

-  With this usage information, we can write a command to tell the CLI tool to download the
   checkpoint with the best validation metric for our experiment (trial):

.. code:: bash

   det trial download --best 5
