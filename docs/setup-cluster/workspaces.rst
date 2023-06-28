.. _workspaces:

#########################
 Workspaces and Projects
#########################

Experiments can be organized into *projects* and *workspaces*. A project is a collection of
experiments, and a workspace is a collection of projects. See the :doc:`Experiment Configuration
</reference/training/experiment-config-reference>` document, which specifies the location of a newly
created experiment. If a workspace and project are not specified, the experiment is created in the
default, ``Uncategorized`` project. Experiments can be moved between projects, and projects can be
moved between workspaces.

*****************
 Getting Started
*****************

Initially, a Determined installation has one workspace containing one project, both titled
``Uncategorized``. Both are considered immutable, which means that they cannot be renamed, archived,
or deleted, and the project cannot be moved to a different workspace, although, experiments can be
moved in or out of an ``Uncategorized`` project at any time.

Use ``det workspace create`` and ``det project create`` to create workspaces and projects for
individuals or teams using Determined. This is recommended for larger teams.

.. code::

   det workspace create <workspace name>
   det project create <workspace name> <project name>

*******
 Usage
*******

WebUI
=====

After logging in, the default landing page for the WebUI is the dashboard. This is where experiments
that do not have a configuration file ``workspace`` and ``project`` are put, including experiments
created before the workspaces feature was released.

Click the navigation sidebar **Workspaces** button to go to the **Workspaces List** page. This page
shows all of the workspaces that currently exist on a cluster.

Click the icon in the upper right corner of a workspace card to access the action menu. The action
menu always contains the option to pin a workspace, which creates an easy access link to the
workspace on the sidebar. If the workspace was created by the currently logged-in user, or if the
current user is an administrator, the action menu also provides the options to edit the workspace
name, archive the workspace, or delete the workspace. Deleting a workspace is permanent and also
deletes all projects contained within it. A workspace cannot be deleted if its projects contain
experiments.

Click a workspace card to go to the **Workspace Details** page. This page shows all currently
selected workspace projects. If a project was created by the currently logged-in user, or if the
current user is an administrator, click the icon in the upper right corner of the project card to
bring up another action menu. This action menu contains the options to edit the project name and
description, move the project to a different workspace, archive the project, or delete the project
if it does not contain experiments.

Click a project card to go to the **Project Details** page. This page shows the experiments that
currently exist for the selected project. The Notes tab lets users create, read, edit, and delete
multiple pages of notes about the selected project.

CLI
===

In the CLI, use the ``det workspace`` and ``det project`` commands to interact with workspaces and
projects. Use the ``-h`` flag to get a list of all possible commands.

.. code::

   det workspace -h
   det project -h
