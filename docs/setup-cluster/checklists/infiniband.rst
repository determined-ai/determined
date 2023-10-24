.. _infiniband:

########################
 Configuring InfiniBand
########################

To configure InfiniBand, you'll need to edit your ``master.yaml`` file.

***************
 Prerequisites
***************

Before starting, ensure your system meets the following requirements:

-  ``ib0`` active/up in ``ifconfig`` on the master and all the compute nodes
-  ``/dev/infiniband`` configured on the master and all the compute nodes
-  InfiniBand connectivity is in place (can be verified using ``ibping``)

*******************************
 Test if InfiniBand is Present
*******************************

To test whether InfiniBand is present on the master and on a Compute Node (e.g., ``cn16``):

.. code:: bash

   [root@master ~]# ibv_devices

To retrieve hardware information:

.. code:: bash

   [root@master ~]# ibv_devinfo -d mlx5_0

Make note of port number ``1``, the number of active ports (e.g., ``PORT_ACTIVE (4)``) and the
``port_lid``. These details are useful for the ``ibping`` test:

.. code:: bash

   [root@master ~]# ibstat mlx5_0

To test the IB ports on both master and cn16 (``PING cn16 -> Master``):

-  Activate the ``ibping`` server on the master:

   .. code:: bash

      [root@master ~]# ibping -S -C mlx5_0 -P 1

   .. note::

      -P 1 → Port 1 on device mlx5_0.

-  Then, with the ``ibping`` client on ``cn16``, ping the master ``port = 1`` five times using
   ``port_lid=1``:

   .. code:: bash

      [root@cn16 ~]# ibping -c 5 -C mlx5_0 -P 1 -L 47

   The expected result should be no packet loss, indicating full connectivity.

To test the IB ports across the master and all compute nodes (``PING cnXX -> Master``):

.. code:: bash

   [root@cn16 ~]# pdsh -w cn[02-79] "ibping -c 5 -C mlx5_0 -P 1 -L 47" | dshbak -c > ./cluster_wide_ib_ping_test.txt

Review the resulting text file for any packet loss or connectivity issues.

***************************************************************
 Configure the Master to Use ``/dev/infiniband`` in Containers
***************************************************************

Update the ``master.yaml`` by adding the following lines:

.. code:: yaml

   task_container_defaults:
     shm_size_bytes: 4294967296
     dtrain_network_interface: ib0
     add_capabilities:
       - IPC_LOCK
     devices:
       - host_path: /dev/infiniband/
         container_path: /dev/infiniband/

.. note::

   The shared memory configuration isn't mandatory for activating IB. However, it's best practice
   for training large models.

Restart the master:

.. code:: bash

   systemctl restart determined-master

Verify the master's status:

.. code:: bash

   systemctl status determined-master
