.. _commands-and-shells:

#####################
 Commands and Shells
#####################

In addition to structured model training workloads, which are handled using :ref:`experiments
<experiments>`, Determined also supports more free-form tasks using *commands* and *shells*.

Commands execute a user-specified program on the cluster. Shells start SSH servers that allow using
cluster resources interactively.

Commands and shells enable developers to use a Determined cluster and its GPUs without having to
write code conforming to the trial APIs. Commands are useful for running existing code in a batch
manner; shells provide access to the cluster in the form of interactive `SSH
<https://en.wikipedia.org/wiki/SSH_(Secure_Shell)>`_ sessions.

This document provides an overview of the most common CLI commands related to shells and commands.

********
Commands
********

+----------------------------------------------------+
| CLI Reference                                      |
+====================================================+
| :doc:`/reference/determined/cli`                   |
+----------------------------------------------------+

The Command-Line Interface (CLI) is distributed as a Python wheel package. After the
wheel is installed, use the CLI ``det`` command to interact with the cluster.

CLI commands start with ``det command``, abbreviated as ``det cmd``.
The main subcommand is ``det cmd run``, which runs a command in the cluster and streams its output.
For example, the following CLI command uses ``nvidia-smi`` to display information about the GPUs
available to tasks in the container:

.. code::

   det cmd run nvidia-smi

More complex commands including shell constructs can be run as well, as long as they are quoted to
prevent interpretation by the local shell:

.. code::

   det cmd run 'for x in a b c; do echo $x; done'

``det cmd run`` will stream output from the command until it finishes, but the command will continue
executing and occupying cluster resources even if the CLI is interrupted or killed (e.g., due to
Control-C being pressed). In order to stop the command or view further output from it, you'll need
its UUID, which can be obtained from the output of either the original ``det cmd run`` or ``det cmd
list``. Once you have the UUID, run ``det cmd logs <UUID>`` to view a snapshot of logs, ``det cmd
logs -f <UUID>`` to view the current logs and continue streaming future output, or ``det cmd kill
<UUID>`` to stop the command.

.. _install-cli:

Installation
============

Users can also interact with Determined using a command-line interface. The CLI is distributed as a
Python wheel package; once the wheel has been installed (see :ref:`install-cli` for details), the
CLI can be used via the ``det`` command.

Each ML engineer that wants to use Determined should install a copy of the Determined CLI on their
local development machine. The CLI can be installed via ``pip``:

.. code::

   pip install determined

The CLI requires Python >= 3.6. We suggest installing the CLI into a `virtualenv
<https://virtualenv.pypa.io/en/latest/>`__, although this is optional. To install the CLI into a
virtualenv, first activate the virtualenv and then type the command above.

After the CLI has been installed, it should be configured to connect to the Determined master at the
appropriate IP address. This can be accomplished by setting the ``DET_MASTER`` environment variable:

.. code::

   export DET_MASTER=<master IP>

You may want to place this into the appropriate configuration file for your login shell (e.g.,
``.bashrc``).

More information about using the Determined CLI can be found by running ``det --help``.

Usage
=====

CLI subcommands usually follow a ``<noun> <verb>`` form, similar to the paradigm of `ip
<http://www.policyrouting.org/iproute2.doc.html>`__. Certain abbreviations are supported, and a
missing verb is the same as ``list``, when possible.

For example, the different commands within each of the blocks below all do the same thing:

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
Each noun also provides a ``help`` verb that describes the possible verbs for that noun. Or you can
provide ``-h`` or ``--help`` as an argument anywhere will cause the CLI to exit after printing help
text for the object or action specified up to that point.

Examples
========

-  ``det e``, ``det experiment``, ``det experiment list``: Show information about experiments in the
   cluster.

-  ``det -m 1.2.3.4 e``, ``DET_MASTER=1.2.3.4 det e``: Show information about experiments in the
   cluster at the network address ``1.2.3.4``.

-  ``det t logs -f 289``: Show the existing logs for trial 289 and continue showing new logs as they
   come in.

-  ``det e label add 17 foobar``: Add the label "foobar" to experiment 17.

-  ``det e describe 493 --metrics --csv``: Display information about experiment 493, including full
   metrics information, in CSV format.

-  ``det e create -f --paused const.yaml .``: Create an experiment with the configuration file
   ``const.yaml`` and the code contained in the current directory. The experiment will be created in
   a paused state (that is, it will not be scheduled on the cluster until it is activated).

-  ``det e set max-slots 85 4``: Ensure that experiment 85 does not take up more than 4 slots in the
   cluster.

-  ``det u create --admin hoid``: Create a new user named "hoid" with admin privileges.

-  ``det version``: Show detailed information about the CLI and master. Note that this command does
   not take both an object and an action.

******
Shells
******

Shell-related CLI commands start with ``det shell``. To start a persistent SSH server container in
the Determined cluster and connect an interactive session to it, use ``det shell start``:

.. code::

   det shell start

After starting a server with ``det shell start``, you can make another independent connection to the
same server by running ``det shell open <UUID>``. The UUID can be obtained from the output of either
the original ``det shell start`` command or ``det shell list``:

.. code::

   $ det shell list
    Id                                   | Owner      | Description                  | State   | Exit Status
   --------------------------------------+------------+------------------------------+---------+---------------
    d75c3908-fb11-4fa5-852c-4c32ed30703b | determined | Shell (annually-alert-crane) | RUNNING | N/A
   $ det shell open d75c3908-fb11-4fa5-852c-4c32ed30703b

Optionally, you can provide extra options to pass to the SSH client when using ``det shell start``
or ``det shell open`` by including them after ``--``. For example, this command will start a new
shell and forward a port from the local machine to the container:

.. code::

   det shell start -- -L8080:localhost:8080

In order to stop the SSH server container and free up cluster resources, run ``det shell kill
<UUID>``.
