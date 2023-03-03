#####################
 System Architecture
#####################

Determined consists of a single **master** and one or more **agents**. There is typically one agent
per compute server; a single machine can serve as both a master and an agent.

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

.. image:: /assets/images/det-ai-sys-arch-01-dark.png
   :class: only-dark

.. image:: /assets/images/det-ai-sys-arch-01-light.png
   :class: only-light

The **task container** runs a training task or other task(s) in a containerized environment.
Training tasks are expected to have access to the data that will be used in training. The **agents**
are responsible for reporting the status of the **task container** to the master.
