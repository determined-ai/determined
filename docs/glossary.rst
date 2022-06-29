########
Glossary
########

`A`_ - `B`_ - `C`_ - `D`_ - `E`_ - `G`_ - `H`_ - `I`_ - `J`_ - `L`_ - `M`_ - `N`_ - `O`_ - `P`_ - `R`_ - `S`_ - `T`_ - `U`_ - `W`_

***
 A
***

.. glossary::

    **administrator**
      TBD

    **agent**
      Determined consists of a single master and one or more agents. There is typically one agent per compute server; a single machine can serve as both a master and an agent. An agent manages a number of slots, which are CPU or GPU computing devices. An agent has no state and only communicates with the master. Each agent is responsible for:

      -  Discovering local computing devices (slots) and sending metadata about them to the master.
      -  Running the workloads that are requested by the master.
      -  Monitoring containers and sending information about them to the master.

      agent-instance-type

      agentrole

      agentsecuritygroupgroupid

      agent-user

      compute server

      configuration

      determined-agent-policy

      determined-agent-username

      determined-agentversionlinuxamddebrpm

      dynamic agents

      slot

    **auxiliary task**
      TBD

***
 B
***

.. glossary::

    **batch**
      TBD

    **batchnumber**
      TBD

    **bucket**
      TBD

***
 C
***

.. glossary::

    **checkpoint, checkpointing**
      TBD

    **cluster**
      TBD

    **command**
      Commands execute a user-specified program on the cluster. Commands let you use a Determined cluster and its GPUs without needing to write trial API code. Commands are useful for running existing code in batch mode.

    **command line interface (CLI)**
      You can interact with Determined using a command-line interface (CLI). The CLI is distributed as a Python wheel package. After the wheel is installed, invoke the CLI using the ``det`` command.

    **configuration file**
      TBD

    **context, context directory**
      TBD

    **created**
      TBD

    **custom environment**
      TBD

***
 D
***

.. glossary::

    **dashboard**
      TBD

    **data layer**
      TBD

    **dataloader**
      TBD

    **dataset**
      TBD

    **defaultscheduler**
      TBD

    **device**
      TBD

***
 E
***

.. glossary::

    **elastic infrastructure**
      TBD

    **experiment**
      An experiment represents the basic unit of running the model training code. An experiment is a collection of one or more trials that are exploring a user-defined hyperparameter space. An experiment can train a single model with a single trial or can define a search over a user-defined hyperparameter space. To create an experiment, create a configuration file that defines the kind of experiment we want to run.

      structured model training workloads

      lifecycle

      profiling (performance)

***
 G
***

.. glossary::

    **globalbatchsize**
      TBD

    **group**
      TBD

***
 H
***

.. glossary::

    **harness**
      TBD

***
 I
***

.. glossary::

    **inbound**
      TBD

    **instance**
      TBD

***
 J
***

.. glossary::

    **job**
      TBD

***
 L
***

.. glossary::

    **launcher, launching**
      TBD

    **loader**
      TBD

    **log, logging**
      detloggingtype

***
 M
***

.. glossary::

    **machine**
      TBD

    **manager**
      TBD

    **master**
      Determined consists of a single master and one or more agents. A single machine can serve as both a master and an agent. The master is the central component of the Determined system. It is responsible for:

      - Storing experiment, trial, and workload metadata.
      - Scheduling and dispatching work to agents.
      - Managing provisioning and deprovisioning of agents in clouds.
      - Advancing the experiment, trial, and workload state machines over time.
      - Hosting the WebUI and the REST API.

      The agents are responsible for reporting the states of trial runner to the master.

      configuration (masteryaml)

      determined-master-service-name

      determined-masterversionlinuxamddebrpm

      proxy

    **maxslotsperpod**
      TBD

    **metric**
      det-state-metrics

    **model**
      checkpoint

      dgetmodel

      dgetmodelmodelname

      dgetmodelsdescriptionocr

      Python class

      registry

      sets

      versioning

    **model-hub**
      TBD

***
 N
***

.. glossary::

    **namespace**
      TBD

    **node**
      TBD

    **notebook**
      TBD

***
 O
***

.. glossary::

    **optimizer**
      TBD

***
 P
***

.. glossary::

    **plugin**
      TBD

    **pool**
      poolname

    **ported**
      TBD

    **priority**
      TBD

***
 R
***

.. glossary::

    **refcluster-configuration**
      TBD

    **registry**
      poolname

    **resource**
      TBD

    **resource group**
      TBD

    **resource pool**
      TBD

    **resource sharing**
      TBD

    **role**
      TBD

***
 S
***

.. glossary::

    **schedule, scheduling**
      policies

      schedulable

      scheduled

      scheduler

    **service**
      TBD

    **sets**
      TBD

    **slot**
      An agent manages a number of slots, which are CPU or GPU computing devices.

***
 T
***

.. glossary::

    **task**
      compute task

      TensorBoards, notebooks, commands, shells

    **template, templating**
      Many configuration files within an organization might contain similar settings. One way to reduce this redundancy is to use configuration templates so you can define settings shared by multiple experiments in a single YAML file, which can then be referenced by configurations that require those settings.

      Each configuration template has a unique name and is stored on the master. If a configuration specifies a template, the effective configuration of the task is the result of merging the configuration file and template file. This expanded configuration is stored so subsequent changes to a template do not affect the reproducibility of experiments that used a previous version of the configuration template.

      A single configuration file can use at most one configuration template. A configuration template cannot use another configuration template.

    **toleration**
      TBD

    **train, training**
      distributed

      trainer

      units (epochs, records, batches)

    **trial**
      A trial is a training task with a defined set of hyperparameters. A common degenerate case is an experiment with a single trial, which corresponds to training a single deep learning model.

      APIs

      checkpoint

      determinedtrial

      determinedtrialcontextgetglobalbatchsize

      maxtrial

      metadata

      workload

    **trial runner**
      The trial runner runs a trial in a containerized environment. The trial runner is expected to have access to the data used in training. 

    **tuning**
      TBD

***
 U
***

.. glossary::

    **user**
      detuser

      determined-username

      determined-username-agent

***
 W
***

.. glossary::

    **WebUI**
      TBD

    **worker**
      TBD

    **workload**
      containerized
