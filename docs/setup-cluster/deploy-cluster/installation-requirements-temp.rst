:orphan:

.. _duplicate-advanced-setup-requirements:

###########################
 Installation Requirements
###########################

.. meta::
   :description: Before setting up Determined, ensure your system meets these requirements.

Before setting up Determined via the :ref:`setup-checklist-temp`, ensure your system meets these
requirements. These requirements are aimed at administrators who are setting up Determined for the
first time.

*************************
 PostgreSQL Requirements
*************************

ADD CONTENT EDITORIAL This content is not meant to provide instructions for installing PostgreSQL.
EDITORIAL This content should provide any prerequisites the admin needs to fulfill before attempting
to install PostgreSQL.

******************
 SSO Requirements
******************

ADD CONTENT EDITORIAL Likewise this content is meant to provide prerequisites the admin needs to
fulfill before attempting to set up SSO.

*********************************
 Non-Root Container Requirements
*********************************

Might be docs for this

********************************
 TLS Configuration Requirements
********************************

-  master needs a full cert chain, including the root cert

-  if master cert is not signed by a well-known CA:

   -  agents need to be configured with the master cert name and cert file in agent.yaml
   -  clients need to be configured with `DET_MASTER_CERT_NAME` and `DET_MASTER_CERT_FILE`, or be
      willing to trust-on-first-use

.. _network-connectivity-requirements:

***********************************
 Network Connectivity Requirements
***********************************

THIS IS FIREWALL RULES

Your system must meet network connectivity requirements.

See also: :ref:`internet access <internet-access>` and :ref:`firewall-rules`.

-  Compute nodes must be able to connect to master node on master's configured port.

-  Compute nodes must be able to connect to each other on any port.

-  Master node must be able to connect to compute nodes on any port.

-  Compute nodes must be able to reach the docker image repository, or have the relevant images
   pre-downloaded.

-  Compute nodes must have access to the desired checkpoint storage.

-  Master node must also have access to desired checkpoint storage.

   -  (this isn't technically required, but it's coming soon)

-  Optionally, client nodes may have access to checkpoint storage for higher performance checkpoint
   access.

-  Master node must have access to postgres

-  Compute nodes must have access to network resources required by user tasks. Frequently this
   includes installing packages from pypi, for instance.

-  Client machines must have access to the master node on master's configured port.
