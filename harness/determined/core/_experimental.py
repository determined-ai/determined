from typing import TYPE_CHECKING

from determined.common import api
from determined.common.api import bindings

if TYPE_CHECKING:
    # These modules are only needed for type checking and
    # cause a circular dependency issue. This bypasses it.
    from determined.experimental import checkpoint, model


class ExperimentalCoreContext:
    """
    ``ExperimentalCoreContext`` gives access to experimental functions in a Determined cluster.
    """

    def __init__(self, session: api.Session, trial_id: int) -> None:
        self._session = session
        self._trial_id = trial_id

    def report_task_using_checkpoint(self, checkpoint: "checkpoint.Checkpoint") -> None:
        """
        Associate ``checkpoint`` with the current task. This links together the metrics
        reporting so that any metrics which are reported to the current task will be
        visible when querying for metrics associated with this checkpoint

        Args:
            checkpoint (checkpoint.Checkpoint): The checkpoint to associate with this task
        """
        req = bindings.v1ReportTrialSourceInfoRequest(
            trialSourceInfo=bindings.v1TrialSourceInfo(
                checkpointUuid=checkpoint.uuid,
                trialId=self._trial_id,
                trialSourceInfoType=bindings.v1TrialSourceInfoType.INFERENCE,
            )
        )
        bindings.post_ReportTrialSourceInfo(
            session=self._session,
            body=req,
        )

    def report_task_using_model_version(self, model_version: "model.ModelVersion") -> None:
        """
        Associate ``model_version`` with the current task. This links together the metrics
        reporting so that any metrics which are reported to the current task will be
        visible when querying for metrics associated with this model version

        Args:
            model_Version (model.ModelVersion): The model version to associate with this task
        """
        assert model_version.checkpoint
        req = bindings.v1ReportTrialSourceInfoRequest(
            trialSourceInfo=bindings.v1TrialSourceInfo(
                checkpointUuid=model_version.checkpoint.uuid,
                trialId=self._trial_id,
                trialSourceInfoType=bindings.v1TrialSourceInfoType.INFERENCE,
                modelId=model_version.model_id,
                modelVersion=model_version.model_version,
            )
        )
        bindings.post_ReportTrialSourceInfo(
            session=self._session,
            body=req,
        )


class DummyExperimentalCoreContext(ExperimentalCoreContext):
    """A Dummy Experimental Context for mypy"""

    def __init__(self) -> None:
        pass

    def report_task_using_checkpoint(self, checkpoint: "checkpoint.Checkpoint") -> None:
        pass

    def report_task_using_model_version(self, model_version: "model.ModelVersion") -> None:
        pass
