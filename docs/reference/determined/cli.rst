.. _cli:

#######################################
 Command-line Interface (CLI) Reference
#######################################

CLI user guide: :doc:`/interfaces/commands-and-shells`

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
