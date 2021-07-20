# Naming Things

The RFC proposed here is the combined thoughts and ideas of many people on the
team.

## Why now?

* Job Queue Project is going to be showing users names, so we should make sure
  we agree on the names we pick.

* Push Architecture is going to bake some of these decisions into APIs and into
  the database, so we should make sure we agree on the names we pick.

## Brainstorm of fundamental concepts (with neutral names)

* An "Allocation's lifetime":
  1. request resources
  2. get allocated
  3. some pod(s)/container(s) run
  4. then it finishes
  5. then it's off the cluster

* Interactive things (Notebooks/commands/shells/tensorboards) (NCST) usually
  consist of only one allocation lifetime

* A "Work Unit":
  * A single, maybe checkpointable/retryable, unit of work
  * if checkpointable/retryable, it might occur over multiple allocation
    lifetimes
  * e.g. a Trial
  * Not about atomicity at all.
  * NCST also consist of a single Work Unit
  * 90% of searchers are single searchers, so most experiments are most often
    a single Work Unit.
  * can be preemptable or not
  * The generic thing we proposed instead of Distributed Batch inference would
    be a single Work Unit, with access to non-training parts of the Generic API

* A "submission":
  * Something submitted by a single cli command or webui action
  * Experiments or NCST both count, Trials do not

* Project currently known as "Job Queue Project" will likely be reordering
  submissions
  * note that an Experiment and a Command are not peers when considered from
    the allocation lifetime perspective

* The thing that was dreamed up during Distributed Batch Inference was kind of
  like a trial (restartable, checkpointable, persisted in the database), so it
  should also consist of one or more Allocation Lifetimes.

## Assigning names to things, Draft 4:

* **Run**: Allocation Lifetime
  * Pros:
    * better proper noun than try, retry, restart, start, attempt, etc
  * Cons:
    * kinda sounds like a "full start-to-finish"
    * "runs" in sigopt/w&b/grid.ai is more like a "trial"

* **Job**: Work Unit
  * Pros:
    * exact word match to k8s jobs
    * one vocabulary that doesn't have both 'task' and 'job'
  * Cons:
    * not an exact match to slurm jobs (which seem more like a "submission")
    * doesn't leave us a good name for "submission"

* ??? = Submission?

A user submits an Experiment (which is only known as an "experiment"), which
consists of multiple Trials (Jobs), each of which might complete over several
Runs.

## Assigning names to things, Draft 3:

* **Run**: Allocation Lifetime
* **Task**: Work Unit
* **Job**: Submission

A user submits an Experiment (a type of Job), which consists of multiple Trials
(Tasks), each of which might complete over several Runs.

* Pros:
  * have a good name for a "submission"

* Cons:
  * "Jobs" and "Tasks" in the same vocabulary is less-than-ideal


## Other Related Concepts:

* **Workflow**: (or "pipelines"?) A collection of Jobs (or sub-workflows) which
  are executed in a user-specified order, or user-specified conditions.
  * (This doesn't exist in the system yet.)
  * (let's punt on "workflow" vs "pipeline")
  * Workflows could be nested; i.e. there's not a higher grouping.
  * Eventually, an Experiment could be a special case of a Workflow:
    * "Start these Trial Jobs Now"
    * "Run this GC Job after all Trial Jobs are done"

* **Interactive Job**: A Job which the user plans on interact with right now;
  the human who asked for it is likely blocked until it starts, and the Job is
  really only meaningful while the user is interacting with it.

* **Project**: (RIP sidney) A user-facing collection of Jobs, checkpoints, and
  user-specified metadata.
  * Semantic grouping of results from work
  * Not a grouping of work itself
  * This doesn't exist in the system yet, but model registry is close.
  * Multiple teams might share a cluster, but each team want to only interact
    with the Jobs/Workflow/CHeckpoints/metadata relevant to them.
  * Could have fancy RBAC properties.

## Names not assigned here:

* Real name for a "submission"?

* "Tasks" is not defined in this nomenclature... what do we do about that?
  * What do we do with the things currently called tasks?
  * (that's a different thing for webui vs cli)

* Better name for NCST?

* "Service": A Job which runs indefinitely.
  * currently would apply to the same jobs as "Interactive Jobs", but
    "Interactive" seems more relevant to the user


Vocab Links:
    k8s jobs: https://kubernetes.io/docs/concepts/workloads/controllers/job/
    slurm jobs: (not clear) https://researchcomputing.princeton.edu/learn/glossary
        they have "array jobs" which are like experiments, sort of?
    sigopt runs: https://app.sigopt.com/docs/runs/overview
