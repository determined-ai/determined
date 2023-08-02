.. _cli-reference:

##########################
 Determined CLI Reference
##########################

.. meta::
   :description: Browse this complete description of the Determined command-line interface that tells you how to print the built-in documentation and formulate your cli commands.

+-----------------------------------------------+
| User Guide                                    |
+===============================================+
| :ref:`Determined CLI User Guide <cli-ug>`     |
+-----------------------------------------------+

The Determined CLI has built-in documentation that you can access by using the help command or
``-h`` and ``--help`` flags. To see a comprehensive list of nouns and abbreviations, simply call
``det help`` or ``det -h``. Each noun has its own set of associated verbs, which are detailed in the
help documentation. For example, to learn more about individual experiment commands, you can type
``det experiment help``.

.. code::

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

********
 Syntax
********

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
