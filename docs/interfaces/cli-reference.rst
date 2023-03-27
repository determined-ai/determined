.. _cli:

###############
 CLI Reference
###############

This reference guide lists the Determined CLI commands, subcommands, and options.

preferrred format Command Name, Abbreviation, Command Description, Subcommands, Example Usage

+------------------------------------------+
| Visit the CLI User Guide                 |
+==========================================+
| :ref:`cli-ug`                            |
+------------------------------------------+

**********************
 Positional Arguments
**********************

Needs to be: Command, Desc, Subcommands, Usage Example

+-----------------+-------------------------------------------------+--------------------------------+
| Option          | Purpose                                         | Usage                          |
+=================+=================================================+================================+
| ``help``        | Show help for this command                      | det help <command>             |
+-----------------+-------------------------------------------------+--------------------------------+
| ``auth``        | Manage auth                                     | det auth <subcommand>          |
+-----------------+-------------------------------------------------+--------------------------------+
| agent (a)       | Manage agents                                   | det agent <subcommand>         |
+-----------------+-------------------------------------------------+--------------------------------+
| command (cmd)   | Manage commands                                 | det command <subcommand>       |
+-----------------+-------------------------------------------------+--------------------------------+
| checkpoint (c)  | Manage checkpoints                              | det checkpoint <subcommand>    |
+-----------------+-------------------------------------------------+--------------------------------+
| deploy (d)      | Manage deployments                              | det deploy <subcommand>        |
+-----------------+-------------------------------------------------+--------------------------------+
| experiment (e)  | Manage experiments                              | det experiment <subcommand>    |
+-----------------+-------------------------------------------------+--------------------------------+
| job (j)         | Manage jobs                                     | det job <subcommand>           |
+-----------------+-------------------------------------------------+--------------------------------+
| master          | Manage master                                   | det master <subcommand>        |
+-----------------+-------------------------------------------------+--------------------------------+
| model (m)       | Manage models                                   | det model <subcommand>         |
+-----------------+-------------------------------------------------+--------------------------------+
| notebook        | Manage notebooks                                | det notebook <subcommand>      |
+-----------------+-------------------------------------------------+--------------------------------+
| oauth           | Manage OAuth                                    | det oauth <subcommand>         |
+-----------------+-------------------------------------------------+--------------------------------+
| preview-search  | Preview search                                  | det preview-search             |
|                 |                                                 | <subcommand>                   |
+-----------------+-------------------------------------------------+--------------------------------+
| project (p)     | Manage projects                                 | det project <subcommand>       |
+-----------------+-------------------------------------------------+--------------------------------+
| rbac            | Manage roles based access controls              | det rbac <subcommand>          |
+-----------------+-------------------------------------------------+--------------------------------+
| resources (res) | Query historical resource allocation            | det resources <subcommand>     |
+-----------------+-------------------------------------------------+--------------------------------+
| shell           | Manage shells                                   | det shell <subcommand>         |
+-----------------+-------------------------------------------------+--------------------------------+
| slot (s)        | Manage slots                                    | det slot <subcommand>          |
+-----------------+-------------------------------------------------+--------------------------------+
| task            | Manage tasks (commands, experiments, notebooks, | det task <subcommand>          |
|                 | shells, tensorboards)                           |                                |
+-----------------+-------------------------------------------------+--------------------------------+
| template (tpl)  | Manage config templates                         | det template <subcommand>      |
+-----------------+-------------------------------------------------+--------------------------------+
| tensorboard     | Manage TensorBoard instances                    | det tensorboard <subcommand>   |
+-----------------+-------------------------------------------------+--------------------------------+
| trial (t)       | Manage trials                                   | det trial <subcommand>         |
+-----------------+-------------------------------------------------+--------------------------------+
| user-group      | Manage user groups                              | det user-group <subcommand>    |
+-----------------+-------------------------------------------------+--------------------------------+
| user (u)        | Manage users                                    | det user <subcommand>          |
+-----------------+-------------------------------------------------+--------------------------------+
| version         | Show version information                        | det version                    |
+-----------------+-------------------------------------------------+--------------------------------+
| workspace (w)   | Manage workspaces                               | det workspace <subcommand>     |
+-----------------+-------------------------------------------------+--------------------------------+

***************************************
 Positional Arguments different format
***************************************

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

.. list-table:: Determined AI CLI Reference Guide
   :header-rows: 1
   :widths: 20 40 20 20

   -  -  Command
      -  Description
      -  Subcommands
      -  Example Usage

   -  -  ``Help``
      -  View detailed information about each command in the Determined CLI.
      -
      -  ``det help experiment``

   -  -  Auth
      -  Manage authentication for your Determined cluster.
      -  login, logout, whoami
      -  det auth login

   -  -  Agent
      -  Manage agents in your Determined cluster.
      -  list, describe, labels
      -  det agent list

   -  -  Checkpoint
      -  Manage checkpoints in your Determined experiments.
      -  list, describe, delete
      -  det checkpoint list

   -  -  Deploy
      -  Manage deployments of Determined on your Kubernetes cluster.
      -  up, down
      -  det deploy up

   -  -  Experiment
      -  Manage experiments in your Determined cluster.
      -  create, list, describe, stop
      -  det experiment create config.yaml

   -  -  Job
      -  Manage jobs in your Determined cluster.
      -  list, describe, kill
      -  det job list

   -  -  Master
      -  Manage the Determined master instance in your cluster.
      -  start, stop, upgrade
      -  det master start

   -  -  Model
      -  Manage models in your Determined cluster.
      -  list, describe, delete
      -  det model list

   -  -  Notebook
      -  Manage notebooks in your Determined cluster.
      -  list, start, stop
      -  det notebook start

   -  -  OAuth
      -  Manage OAuth authentication for your Determined cluster.
      -  create, list, delete
      -  det oauth list

   -  -  Preview-Search
      -  Preview search results for your experiments.
      -  up, down, query
      -  det preview-search query "val_loss < 0.1"

   -  -  Project
      -  Manage projects in your Determined cluster.
      -  create, list, delete
      -  det project list

   -  -  RBAC
      -  Manage roles-based access control in your Determined cluster.
      -  create, list, delete
      -  det rbac list

   -  -  Resources
      -  Query historical resource allocation in your Determined cluster.
      -  agents, slurm, kubernetes
      -  det resources agents

   -  -  Shell
      -  Manage shells in your Determined cluster.
      -  start, stop, list
      -  det shell start

   -  -  Slot
      -  Manage slots in your Determined cluster.
      -  list, describe, delete
      -  det slot list

   -  -  Task
      -  Manage tasks in your Determined cluster.
      -  list, describe, kill
      -  det task list

   -  -  Template
      -  Manage configuration templates for your Determined cluster.
      -  list, create
      -  det template list

********************
 Optional Arguments
********************

+-------------------------------------+---------------------------------------------------------------+
| Argument                            | Description                                                   |
+=====================================+===============================================================+
| -h, --help                          | Show help for this command                                    |
+-------------------------------------+---------------------------------------------------------------+
| -u username, --user username        | Execute the command as the specified user (default: None)     |
+-------------------------------------+---------------------------------------------------------------+
| -m address, --master address        | Specify the master address (default: localhost:8080)          |
+-------------------------------------+---------------------------------------------------------------+
| -v, --version                       | Print the CLI version and exit                                |
+-------------------------------------+---------------------------------------------------------------+

*************************
 COMMANDS TEST OF FORMAT
*************************

``Help``
========

The ``help`` command allows you to view detailed information about each command in the Determined
CLI. To use this command, provide the name of the command as an argument. Here's an example usage:

.. code:: bash

   det help experiment

``Auth``
========

The auth command allows you to manage authentication for your Determined cluster. This command has
several subcommands, including login, logout, and whoami. Here's an example usage:

.. code:: bash

   det auth login

``Agent``
=========

The agent command allows you to manage agents in your Determined cluster. This command has several
subcommands, including list, describe, and labels. Here's an example usage:

.. code:: bash

   det agent list

``Checkpoint``
==============

The checkpoint command allows you to manage checkpoints in your Determined experiments. This command
has several subcommands, including list, describe, and delete. Here's an example usage:

.. code:: bash

   det checkpoint list

``Deploy``
==========

The deploy command allows you to manage deployments of Determined on your Kubernetes cluster. This
command has several subcommands, including up and down. Here's an example usage:

.. code:: bash

   det deploy up

``Experiment``
==============

The experiment command allows you to manage experiments in your Determined cluster. This command has
several subcommands, including create, list, describe, and stop. Here's an example usage:

.. code:: bash

   det experiment create config.yaml

``Job``
=======

The job command allows you to manage jobs in your Determined cluster. This command has several
subcommands, including list, describe, and kill. Here's an example usage:

.. code:: bash

   det job list

``Master``
==========

The master command allows you to manage the Determined master instance in your cluster. This command
has several subcommands, including start, stop, and upgrade. Here's an example usage:

.. code:: bash

   det master start

``Model``
=========

The model command allows you to manage models in your Determined cluster. This command has several
subcommands, including list, describe, and delete. Here's an example usage:

.. code:: bash

   det model list

``Notebook``
============

The notebook command allows you to manage notebooks in your Determined cluster. This command has
several subcommands, including list, start, and stop. Here's an example usage:

.. code:: bash

   det notebook start

``OAuth``
=========

The oauth command allows you to manage OAuth authentication for your Determined cluster. This
command has several subcommands, including create, list, and delete. Here's an example usage:

.. code:: bash

   det oauth list

``Preview-Search``
==================

The preview-search command allows you to preview search results for your experiments. This command
has several subcommands, including up, down, and query. Here's an example usage:

.. code:: bash

   det preview-search query "val_loss < 0.1"

``Project``
===========

The project command allows you to manage projects in your Determined cluster. This command has
several subcommands, including create, list, and delete. Here's an example usage:

.. code:: bash

   det project list

``RBAC``
========

The rbac command allows you to manage roles-based access control in your Determined cluster. This
command has several subcommands, including create, list, and delete. Here's an example usage:

.. code:: bash

   det rbac list

``Resources``
=============

The resources command allows you to query historical resource allocation in your Determined cluster.
This command has several subcommands, including agents, slurm, and kubernetes. Here's an example
usage:

.. code:: bash

   det resources agents

``Shell``
=========

The shell command allows you to manage shells in your Determined cluster. This command has several
subcommands, including start, stop, and list. Here's an example usage:

.. code:: bash

   det shell start

``Slot``
========

The slot command allows you to manage slots in your Determined cluster. This command has several
subcommands, including list, describe, and delete. Here's an example usage:

.. code:: bash

   det slot list

``Task``
========

The task command allows you to manage tasks in your Determined cluster. This command has several
subcommands, including list, describe, and kill. Here's an example usage:

.. code:: bash

   det task list

``Template``
============

The template command allows you to manage configuration templates for your Determined cluster. This
command has several subcommands, including list and create. Here's an example usage:

.. code:: bash

   det template list

******************************************
 THIS IS THE ORIGINAL TABLE TO BE REMOVED
******************************************

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
