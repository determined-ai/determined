.. _system-architecture:

#####################
 System Architecture
#####################

Determined consists of a single **master** and one or more **agents**. There is typically one agent
per compute server; a single machine can serve as both a master and an agent.

.. image:: /assets/images/_det-ai-sys-arch-network-light.png
   :class: only-dark
   :alt: Determined AI system architecture diagram describing master and agent components including network connectivity traffic in dark mode

.. image:: /assets/images/_det-ai-sys-arch-network-light.png
   :class: only-light
   :alt: Determined AI system architecture diagram describing master and agent components including network connectivity traffic in light mode

*Determined AI System Architecture*

|

*******
 About
*******

The **master** is the central component of the Determined system. It is responsible for

-  Storing experiment, trial, and workload metadata.
-  Scheduling and dispatching work to agents.
-  Managing provisioning and deprovisioning of agents in clouds.
-  Advancing the experiment, trial, and workload state machines over time.
-  Hosting the WebUI and the REST API.

An **agent** manages a number of **slots**, which are computing devices (typically a GPU or CPU). An
agent has no state and only communicates with the master. Each agent is responsible for

-  Discovering local computing devices (slots) and sending metadata about them to the master.
-  Running the workloads that are requested by the master.
-  Monitoring containers and sending information about them to the master.

The **task container** runs a training task or other task(s) in a containerized environment.
Training tasks are expected to have access to the data that will be used in training. The **agents**
are responsible for reporting the status of the **task container** to the master.

.. _firewall-rules:

.. _port-reference:

**********************
 Network Connectivity
**********************

When the system is configured according to the :ref:`setup requirements
<advanced-setup-requirements>`, network traffic flows to and from the master and agents as follows:

-  **Master-Compute Connection**: Compute nodes connect to the master node on the master's
   configured port.

-  **Inter-Compute Connection**: Compute nodes can connect to each other on any port.

-  **Master-Compute Reverse Connection**: The master node can establish a connection to compute
   nodes on any port.

-  **Docker Image Access**: One of the following is true:

   -  Compute nodes can access the Docker image repository, or
   -  Compute nodes already contain the relevant pre-downloaded images.

-  **Checkpoint Storage Access**:

   -  Both compute nodes and the master node can access the desired checkpoint storage.
   -  Optionally, client nodes can connect to checkpoint storage access for better performance.

-  **Database Access**: The master node connects to PostgreSQL.

-  **User Task Resources**: Compute nodes can reach any network resources necessary for user tasks,
   such as fetching packages from PyPI.

-  **Client-Master Connection**: Client machines can connect to the master node using the master's
   configured port.
