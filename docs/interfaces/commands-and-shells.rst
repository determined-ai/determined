.. _commands-and-shells:

#####################
 Commands and Shells
#####################

Determined commands and shells provide support for running code on a Determined cluster without
writing a model. This page describes how to manage GPU-powered batch commands and interactive
shells.

Commands and shells are started through the Determined command-line interface (CLI). To learn more,
including installation instructions, visit the :ref:`Determined CLI user guide <cli-ug>` or
:ref:`Determined CLI Reference <cli-reference>`.

Commands execute a user-specified program on the cluster. Commands are useful for running existing
code in batch mode. Shells start SSH servers that let you use cluster resources interactively.
Shells provide access to the cluster in the form of interactive `SSH
<https://en.wikipedia.org/wiki/SSH_(Secure_Shell)>`_ sessions.

**********
 Commands
**********

Determined commands are manipulated with CLI commands starting with ``det command``, abbreviated as
``det cmd``. The main subcommand is ``det cmd run``, which runs a command in the cluster and streams
its output. For example, the following CLI command uses ``nvidia-smi`` to display information about
the GPUs available to tasks in the container:

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
