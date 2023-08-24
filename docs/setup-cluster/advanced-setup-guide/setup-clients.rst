:orphan:

.. _setup-clients:

###############
 Setup Clients
###############

You can set up clients for interacting with the Determined Master through the Determined CLI. Follow
these instructions to set up clients.

**************************************************
 Step 1 - Set ``DET_MASTER`` Environment Variable
**************************************************

Set the ``DET_MASTER`` environment variable, which is the network address of the Determined master.
You can override the value in the command line using the ``-m`` option.

*************************************
 Step 2 - Install the Determined CLI
*************************************

The Determined CLI is a command-line tool that lets you launch new experiments and interact with a
Determined cluster. The CLI can be installed on any machine you want to use to access Determined. To
install the CLI, follow the :ref:`installation <install-cli>` instructions.

The ``-m`` or ``--master`` flag determines the network address of the Determined master that the CLI
connects to. If this flag is not specified, the value of the ``DET_MASTER`` environment variable is
used; if that environment variable is not set, the default address is ``localhost``. The master
address can be specified in three different formats:

-  ``example.org:port`` (if ``port`` is omitted, it defaults to ``8080``)
-  ``http://example.org:port`` (if ``port`` is omitted, it defaults to ``80``)
-  ``https://example.org:port`` (if ``port`` is omitted, it defaults to ``443``)

Examples:

.. code:: bash

   # Connect to localhost, port 8080.
   $ det experiment list

   # Connect to example.org, port 8888.
   $ det -m example.org:8888 e list

   # Connect to example.org, port 80.
   $ det -m http://example.org e list

   # Connect to example.org, port 443.
   $ det -m https://example.org e list

   # Connect to example.org, port 8080.
   $ det -m example.org e list

   # Set default Determined master address to example.org, port 8888.
   $ export DET_MASTER="example.org:8888"
