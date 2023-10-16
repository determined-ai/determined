.. _system-architecture:

#####################
 System Architecture
#####################

Determined consists of a single **master** and one or more **agents**. There is typically one agent
per compute server; a single machine can serve as both a master and an agent.

.. image:: /assets/images/_det-ai-sys-arch-01-dark.png
   :class: only-dark
   :alt: Determined AI system architecture diagram describing master and agent components in dark mode

.. image:: /assets/images/_det-ai-sys-arch-01-light.png
   :class: only-light
   :alt: Determined AI system architecture diagram describing master and agent components in light mode

*Determined AI System Architecture*

|

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
