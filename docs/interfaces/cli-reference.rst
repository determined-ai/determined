.. _cli:

########################################
 CLI Reference
########################################

This reference guide lists the Determined CLI commands, subcommands, and options.

preferrred format
Command Name, Abbreviation, Command Description, Subcommands, Example Usage

+------------------------------------------+
| Visit the CLI User Guide                 |
+==========================================+
| :ref:`cli-ug`                            |
+------------------------------------------+


****************************************
 Positional Arguments
****************************************

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
| preview-search  | Preview search                                  | det preview-search <subcommand>|
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


****************************************
 Optional Arguments
****************************************


.. code:: bash

   optional arguments:
     -h, --help            show this help message and exit
     -u username, --user username
                           run as the given user (default: None)
     -m address, --master address
                           master address (default: localhost:8080)
     -v, --version         print CLI version and exit



****************************************
 Optional Arguments
****************************************


.. list-table::
    :header-rows: 1
    :widths: 25 35 25 15
    
    * - Task
      - Example
      - Command
      - Options
    * - List all experiments
      - Display a list of all experiments in the cluster.
      - ``det experiment list``
      - 

list-table
header-rows: 1
widths: 20 40 40

Option
Description
Examples

-h, --help
Display the help message and exit the CLI.
``det experiment create -h``and ``det agent list --help``

-u username, --user username

Execute the command as the specified user (default: None).

det experiment create -u john_doe config.yaml
det agent list --user jane_doe

-m address, --master address

Specify the master address (default: localhost:8080).


det experiment create -m 192.168.1.100:8080 config.yaml
det agent list --master my-master-domain:8080

-v, --version

Print the CLI version and exit.


det -v
det --version

***********************************************
 THIS IS THE ORIGINAL TABLE DO WE PREFER THIS?
***********************************************

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





``Help``
=========

The ``help`` command allows you to view detailed information about each command in the Determined CLI. 
To use this command, provide the name of the command as an argument. Here's an example usage:

.. code-block:: bash

    det help experiment

``Auth Command``
=================

The auth command allows you to manage authentication for your Determined cluster.
This command has several subcommands, including login, logout, and whoami. Here's an example usage:

.. code-block:: bash

    det auth login

Agent Command

The agent command allows you to manage agents in your Determined cluster.
This command has several subcommands, including list, describe, and labels. Here's an example usage:

.. code-block:: bash
  
  det agent list

Checkpoint Command

The checkpoint command allows you to manage checkpoints in your Determined experiments.
This command has several subcommands, including list, describe, and delete. Here's an example usage:

.. code-block:: bash
  
  det checkpoint list

Deploy Command

The deploy command allows you to manage deployments of Determined on your Kubernetes cluster.
This command has several subcommands, including up and down. Here's an example usage:

.. code-block:: bash
  
  det deploy up

Experiment Command

The experiment command allows you to manage experiments in your Determined cluster.
This command has several subcommands, including create, list, describe, and stop. Here's an example usage:

.. code-block:: bash
  
  det experiment create config.yaml

Job Command

The job command allows you to manage jobs in your Determined cluster.
This command has several subcommands, including list, describe, and kill. Here's an example usage:

.. code-block:: bash
  
  det job list

Master Command

The master command allows you to manage the Determined master instance in your cluster.
This command has several subcommands, including start, stop, and upgrade. Here's an example usage:

.. code-block:: bash
  
  det master start

Model Command

The model command allows you to manage models in your Determined cluster.
This command has several subcommands, including list, describe, and delete. Here's an example usage:

.. code-block:: bash
  
  det model list

Notebook Command

The notebook command allows you to manage notebooks in your Determined cluster.
This command has several subcommands, including list, start, and stop. Here's an example usage:

.. code-block:: bash
  
  det notebook start

OAuth Command

The oauth command allows you to manage OAuth authentication for your Determined cluster.
This command has several subcommands, including create, list, and delete. Here's an example usage:

.. code-block:: bash
  
  det oauth list

Preview-Search Command

The preview-search command allows you to preview search results for your experiments.
This command has several subcommands, including up, down, and query. Here's an example usage:

.. code-block:: bash
  
  det preview-search query "val_loss < 0.1"

Project Command

The project command allows you to manage projects in your Determined cluster.
This command has several subcommands, including create, list, and delete. Here's an example usage:

.. code-block:: bash
  
  det project list

RBAC Command

The rbac command allows you to manage roles-based access control in your Determined cluster.
This command has several subcommands, including create, list, and delete. Here's an example usage:

.. code-block:: bash
  
  det rbac list

Resources Command

The resources command allows you to query historical resource allocation in your Determined cluster.
This command has several subcommands, including agents, slurm, and kubernetes. Here's an example usage:

.. code-block:: bash
  
  det resources agents

Shell Command

The shell command allows you to manage shells in your Determined cluster.
This command has several subcommands, including start, stop, and list. Here's an example usage:

.. code-block:: bash
  
  det shell start

Slot Command

The slot command allows you to manage slots in your Determined cluster.
This command has several subcommands, including list, describe, and delete. Here's an example usage:

.. code-block:: bash
  
  det slot list

Task Command

The task command allows you to manage tasks in your Determined cluster.
This command has several subcommands, including list, describe, and kill. Here's an example usage:

.. code-block:: bash
  
  det task list

Template Command

The template command allows you to manage configuration templates for your Determined cluster.
This command has several subcommands, including list and create. Here's an example usage:

.. code-block:: bash
  
  det template list







