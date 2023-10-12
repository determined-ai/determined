.. _cli-ug:

################
 Determined CLI
################

+-----------------------------------------------+
| Reference                                     |
+===============================================+
| :doc:`/reference/cli-reference`               |
+-----------------------------------------------+

To use Determined, you'll need, at minimum, the Determined command-line interface (Determined CLI)
and a Determined cluster. The Determined CLI includes the ``det`` command-line tools for interacting
with a Determined cluster. This page contains instructions for using the CLI, including installion
and upgrade.

.. warning::

   Although Determined supports password-based authentication, communication between the Determined
   CLI, Determined WebUI, and Determined master does not take place over an encrypted channel by
   default.

.. note::

   All users should install the Determined CLI on their local development machine.

.. note::

   You can also interact with Determined using the :ref:`web interface (WebUI) <web-ui-if>`.

.. _install-cli:

**************
 Installation
**************

The CLI is distributed as a Python wheel package and requires Python >= 3.7. We recommend setting up
a `virtualenv <https://virtualenv.pypa.io/en/latest/>`__ and using the ``pip`` utility to install
``determined`` into the environment:

.. code::

   pip install determined

.. include:: ../../_shared/note-pip-install-determined.txt

After installing the CLI, configure it to connect to the Determined master at the appropriate IP
address. To do this, set the ``DET_MASTER`` environment variable:

.. code::

   export DET_MASTER=<master IP>

Place this into the appropriate configuration file for your login shell, such as ``.bashrc``.

Environment Variables
=====================

-  ``DET_MASTER``: The network address of the master of the Determined installation. The value can
   be overridden using the ``-m`` flag.

-  ``DET_USER`` and ``DET_PASS``: Specifies the current Determined user and password for use when
   non-interactive behaviour is required such as scripts. ``det user login`` is preferred for normal
   usage. Both ``DET_USER`` and ``DET_PASS`` must be set together to take effect. These variables
   can be overridden by using the ``-u`` flag.

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

   pip uninstall determined

This command uninstalls the ``determined`` library, including the Determined CLI, from your system.

Upgrading
=========

To upgrade the Determined CLI to the latest version, use the following command:

.. code:: bash

   pip install --upgrade determined

This command upgrades ``determined`` (along with the Determined CLI) to the latest available
version.

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

*****************
 Getting Started
*****************

After installing the Determined CLI, you can start using it to interact with your Determined
cluster. The CLI is invoked with the ``det`` command.

CLI subcommands usually follow a ``<noun> <verb>`` form, similar to the paradigm of `ip
<http://www.policyrouting.org/iproute2.doc.html>`__. Certain abbreviations are supported, and a
missing verb is the same as ``list``, when possible. The following examples show different ways to
achieve the same outcome using the full ``<noun> <verb>`` form, then with an abbreviation, and
finally with an implicit ``list``:

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

****************
 Usage Examples
****************

.. list-table::
   :header-rows: 1
   :widths: 25 35 25 15

   -  -  Task
      -  Example
      -  Command
      -  Options

   -  -  List all experiments.
      -  Display a list of all experiments in the cluster.
      -  ``det experiment list``
      -

   -  -  List all experiments for a specific network address.
      -  Display a list of all experiments in the cluster at network address ``1.2.3.4``.
      -  ``det -m 1.2.3.4 e``
      -

   -  -  View a snapshot of logs.
      -  Display the most recent logs for a specific command.
      -  ``det command logs <command_id>``
      -  -f, --tail

   -  -  View logs for a trial.
      -  Show the logs for trial 289 and continue streaming logs in real-time.
      -  ``det t logs -f 289``
      -  -f

   -  -  Add a label.
      -  Add the label ``foobar`` to experiment 17.
      -  ``det e label add 17 foobar``
      -

   -  -  Create an experiment.

      -  Create an experiment in a paused state with the configuration file ``const.yaml`` and the
         code contained in the current directory. The paused experiment is not scheduled on the
         cluster until activated.

      -  ``det e create -f --paused const.yaml .``

      -

   -  -  Describe an experiment.
      -  Display information about experiment 493, including full metrics, in CSV format.
      -  ``det e describe 493 --metrics --csv``
      -

   -  -  Set max slots.
      -  Ensure that experiment 85 does not use more than 4 slots in the cluster.
      -  ``det e set max-slots 85 4``
      -

   -  -  Display details about the CLI and master.
      -  Show detailed information about the CLI and master. This command does not take both an
         object and an action.
      -  ``det version``
      -

   -  -  Stop (kill) a command.
      -  Terminate a running command.
      -  ``det command kill <command_id>``
      -

   -  -  Set a password for the admin user.
      -  Set the password for the admin user during cluster setup.
      -  ``det user change-password admin``
      -

   -  -  Create a user.
      -  Create a new user named ``hoid`` who has admin privileges.
      -  ``det u create --admin hoid``
      -

.. container:: child-articles

   .. toctree::
      :glob:
      :maxdepth: 2

      ./*
