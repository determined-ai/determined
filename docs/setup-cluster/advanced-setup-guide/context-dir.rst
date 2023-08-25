:orphan:

.. _context-directory:

####################################
 Transferring the Context Directory
####################################

You can use the ``-c <directory>`` option to transfer files from a directory on your local machine,
called the context directory, to the container. The context directory contents are placed in the
container working directory before the command or shell run. Files in the context can be accessed
using relative paths.

.. code::

   $ mkdir context
   $ echo 'print("hello world")' > context/run.py
   $ det cmd run -c context python run.py

The total size of the files in the context directory must be less than 95 MB. Larger files, such as
datasets, must be mounted into the container, downloaded after the container starts, or included in
a :ref:`custom Docker image <custom-docker-images>`.
