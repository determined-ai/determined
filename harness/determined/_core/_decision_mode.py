import enum


class DecisionMode(enum.Enum):
    """
    DecisionMode defines how certain decisions will be made in distributed computing situations.
    """

    """
    ChiefOnly indicates that only the chief needs to know about a decision.
    """
    ChiefOnly = "CHIEF_ONLY"

    """
    WorkersAskChief indicates that all workers need to know about a decision and that the decision
    must be synchronized across workers.

    In the preemption case, this would indicate that all workers will call should_preempt() in-step.
    With WorkersAskChief, only the chief will actually communicate with the master, then the chief
    will broadcast its decision to all workers.

    WorkersAskChief requires that all workers call the API in-step, but it guarantees that all
    workers will have the same decision afterwards.
    """
    WorkersAskChief = "WORKERS_ASK_CHIEF"

    """
    WorkersAskMaster indicates that all workers will make their own decision independently by
    communicating to the master.

    In the preemption case, this would indicate that all workers will call should_preempt(), but it
    need not be in-step.  Each worker will receive the preemption signal at roughly the same time,
    but it is the responsibility of the calling code to tolerate situations where some workers have
    exited due to preemption and others have not.


    """
    WorkersAskMaster = "WORKERS_ASK_MASTER"
