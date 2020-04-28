from typing import Optional

from determined_common.experimental.checkpoint import Checkpoint, get_checkpoint
from determined_common.experimental.experiment import ExperimentReference
from determined_common.experimental.session import Session
from determined_common.experimental.trial import TrialReference


class Determined:
    """
    Determined gives access to Determined API objects.

    Arguments:
        master (string, optional): The URL of the Determined master. If
            this argument is not specified environment variables DET_MASTER and
            DET_MASTER_ADDR will be checked for the master URL in that order.
        user (string, optional): The Determined username used for
            authentication. (default: ``determined``)
    """

    def __init__(
        self, master: Optional[str] = None, user: Optional[str] = None,
    ):
        self._session = Session(master, user)

    def get_experiment(self, experiment_id: int) -> ExperimentReference:
        """
        Get the :py:class:`det.experimental.ExperimentReference` representing the
        experiment with the provided experiment ID.
        """
        return ExperimentReference(experiment_id, self._session._master)

    def get_trial(self, trial_id: int) -> TrialReference:
        """
        Get the :py:class:`det.experimental.TrialReference` representing the
        trial with the provided trial ID.
        """
        return TrialReference(trial_id, self._session._master)

    def get_checkpoint(self, uuid: str) -> Checkpoint:
        """
        Get the :py:class:`det.experimental.Checkpoint` representing the
        checkpoint with the provided UUID.
        """
        return get_checkpoint(uuid, self._session._master)
