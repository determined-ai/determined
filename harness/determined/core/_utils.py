from determined.common import api
from determined.common.api import bindings

# from determined.experimental import Checkpoint


class UtilsContext:
    """
    ``UtilsContext`` gives access to various miscellaneous functions in a Determined cluster
    """

    def __init__(self, session: api.Session, trial_id: int) -> None:
        self._session = session
        self._trial_id = trial_id

    # def report_task_using_checkpoint(self, checkpoint_uuid: str) -> None:
    #     req = bindings.v1ReportTrialSourceInfoRequest(
    #         trialSourceInfo=bindings.v1TrialSourceInfo(
    #             checkpointUuid=checkpoint_uuid,
    #             trialId=self._trial_id,
    #             trialSourceInfoType=bindings.v1TrialSourceInfoType.INFERENCE,
    #         )
    #     )
    #     bindings.post_ReportTrialSourceInfo(
    #         session=self._session,
    #         body=req,
    #     )

    # def report_task_using_checkpoint_full_obj(self, checkpoint: Checkpoint) -> None:
    def report_task_using_checkpoint(self, checkpoint) -> None:
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

    def report_task_using_model_version(self, model_version) -> None:
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
