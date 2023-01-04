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

The **trial runner** runs a trial in a containerized environment. So the trial runners are expected
to have access to the data that will be used in training. The **agents** are responsible for
reporting the states of **trial runner** to the master.

**Diagram**:

.. code::

   ┌─────────────────────────────Deployment─┐
   │                                        │
   │ ┌─Cluster────────────────┐             │
   │ │                        │             │
   │ │    ┌────────────────┐  │             │
   │ │   ┌┼┼┼┼┼┼┼┼┼┼┼┼┼┼┼┼┼│  │             │
   │ │  ┌┴───────────────┼┼│  │             │
   │ │  │                │┼┤◄─┼───┐         │
   │ │  │ Agent(s)...    │┼│  │   │         │
   │ │  │ ┌───────────┐  │┼│  │   │         │
   │ │  │ │Trial(s)...├┐ │┼│  │   │         │
   │ │  │ └┬──────────┼│ │┼│  │   │         │
   │ │  │  └───────────┘ │┼┘  │   │         │
   │ │  │                ├┘   │   ▼         │
   │ │  └────────────────┘    │ ┌─────────┐ │
   │ │        ▲               │ │         │ │
   │ │        │               │ │ Storage │ │
   │ │        ▼               │ │         │ │
   │ │  ┌───────────────┐     │ └────┬────┘ │
   │ │  │               │     │   ▲  │      │
   │ │  │    Master     │     │   │  │      │
   │ │  │               │◄────┼───┘  │      │
   │ │  └───────────────┘     │      │      │
   │ │     ▲        ▲         │      │      │
   │ │     │        │         │      │      │
   │ └─────┼────────┼─────────┘      │      │
   │       │        │                │      │
   └───────┼────────┼────────────────┼──────┘
           │        │      ▲         │
           ▼        ▼      │         │
   ┌──────────┐  ┌─────────┴────┐    │
   │          │  │              │    │
   │ Web View │  │ Command Line │◄───┘
   │          │  │              │
   └──────────┘  └──────────────┘
