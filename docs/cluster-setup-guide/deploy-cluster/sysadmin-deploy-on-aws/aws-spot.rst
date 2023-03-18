.. _aws-spot:

####################
 Use Spot Instances
####################

This document describes how to use AWS spot instances with Determined. Spot instances can be much
cheaper than on-demand instances (up to 90% cheaper, but more often 70-80%) but they are unreliable,
so software that runs on spot instances must be fault tolerant. Unfortunately, deep learning code is
often not written with fault tolerance in mind, preventing many practitioners from using spot
instances easily. Because Determined was built with fault tolerance as a core feature, it works
seamlessly when run on top of spot instances, allowing users to reduce their training costs by
setting a single flag in the :ref:`cluster-configuration`.

Determined handles most details of working with spot instances, but knowing how spot instances work
at a high level is helpful for understanding the behavior of Determined when provisioning spot
instances.

*******************
 On-Demand vs Spot
*******************

The standard way to create an EC2 instance is to create an on-demand instance. On-demand instances
always have the same price and once you have created one, you will keep that instance indefinitely.

The number of EC2 instances desired by customers varies over time. AWS wants to have enough hardware
so that during peak usage periods, all customers are able to get as many on-demand instances as they
need. This means that during non-peak periods (which is most of the time), AWS has extra hardware
capacity sitting around unused. AWS makes these extra instances available to customers via the spot
market as `spot instances <https://aws.amazon.com/ec2/spot/>`_. They can be rented at a much reduced
cost, but the trade-off is that, if AWS needs instances to satisfy on-demand users, they can reclaim
your spot instance and give it to the on-demand user. If all available instances are in use by
on-demand customers, you will not be able to launch any spot instances.

Because spot capacity depends on supply and demand, your ability to launch spot instances will vary
by region/availability zone, as well as instance type. GPU instances, particularly those with the
most modern GPUs, are often in high demand and they may sometimes be difficult to get as spot
instances.

**************************************
 Using Spot Instances with Determined
**************************************

Because they can be reclaimed, using spot instances requires you to have good fault tolerance built
into your software. Determined was built with fault-tolerance as a core feature, so using spot
instances is usually as easy as configuring a :ref:`resource pool <resource-pools>` with ``spot:
true`` in the master configuration. Here is a fragment of a master configuration file that defines a
resource pool with up to 10 g4dn.metal spot instances:

.. code:: yaml

   resource_pools:
     - pool_name: aws-spot-g4dn-metal
       provider:
         type: aws
         max_instances: 10
         instance_type: g4dn.metal
         spot: true
         # ...

When using ``det deploy`` to install Determined on AWS, you can also specify ``--spot`` when running
``det deploy aws up`` to cause the default CPU and GPU resource pools to use spot instances.

AWS might not always have the capacity to fulfill spot requests. When AWS is out of capacity, the
Determined cluster will wait until AWS has capacity and can fulfill the requests. This will be
visible in the master logs. A useful approach is to configure the master with two resource pools,
one that uses spot instances and another that uses on-demand instances. This allows users to easily
select whether to use spot or on-demand instances for a given job by setting
``resources.resource_pool`` appropriately in their experiment configuration file.

**************
 Spot Pricing
**************

Unlike on-demand instances, the market price of a spot instance varies. Once the spot instance has
been created, the hourly cost to run that instance is constant, but if you try to create another
spot instance, the price may have changed. The spot price is typically around 70% less than the
on-demand price (see `AWS's spot advisor <https://aws.amazon.com/ec2/spot/instance-advisor/>`_ for
up-to-date information), but it can technically rise as high as the on-demand price.

AWS and Determined allow you to specify the maximum price that you are willing to pay for a spot
instance. If the market price is above that number, the spot instance will not be created until the
price falls under your maximum price. You can set this value via the ``spot_max_price`` field in the
master configuration. The market price being above your ``spot_max_price`` is another reason why
Determined may not be creating spot instances when you expect it to. If this is preventing
Determined from creating spot instances, this will be visible in the master logs.

Many users want to reduce costs by using spot instances, but have deadlines and are not willing to
delay their experiments. In this case, it may be best to not set ``spot_max_price`` and pay whatever
the market price is. Your mileage may vary, but at Determined, we have regularly seen 70% cost
reductions when using V100s, without specifying a ``spot_max_price``.

*****************
 Troubleshooting
*****************

Some users may encounter the following error in the Determined master logs the first time they try
to use spot instances:

.. code::

   AWS error while launching spot instances, AuthFailure.ServiceLinkedRoleCreationNotPermitted, The provided credentials do not have permission to create the service-linked role for EC2 Spot Instances.

When this error occurs, please check the `AWS documentation
<https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/spot-requests.html#service-linked-roles-spot-instance-requests>`__.
Most likely, you will need to use the AWS CLI to create the ``AWSServiceRoleForEC2Spot`` role:

.. code::

   aws iam create-service-linked-role --aws-service-name spot.amazonaws.com
