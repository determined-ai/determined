from determined.common import api
from determined.common.api import bindings


class UtilsContext:
    """
    ``UtilsContext`` gives access to various miscellaneous functions in a Determined cluster
    """

    def __init__(self, session: api.Session, trial_id: int) -> None:
        self._session = session
        self._trial_id = trial_id

    def report_task_using_checkpoint(self, checkpoint_uuid: str) -> None:
        req = bindings.v1ReportTrialSourceInfoRequest(
            trialSourceInfo=bindings.v1TrialSourceInfo(
                checkpointUuid=checkpoint_uuid,
                trialId=self._trial_id,
                trialSourceInfoType=bindings.v1TrialSourceInfoType.INFERENCE,
            )
        )
        bindings.post_ReportTrialSourceInfo(
            session=self._session,
            body=req,
        )

    # def report_task_using_checkpoint_full_obj(self, checkpoint: checkpoint.Checkpoint) -> None:
    #     req = bindings.v1ReportTrialSourceInfoRequest(
    #         trialSourceInfo=bindings.v1TrialSourceInfo(
    #             checkpointUuid=checkpoint_uuid,
    #             trialId=self._trial_id,
    #             trialSourceInfoType=bindings.v1TrialSourceInfoType.INFERENCE,
    #             # modelId="asdasdf"
    #             # modelVersion=3
    #         )
    #     )
    #     bindings.post_ReportTrialSourceInfo(
    #         session=self._session,
    #         body=req,
    #     )

    # def report_task_using_model_version(self, model_name: str, version: int) -> None:
    #     req = bindings.v1ReportTrialSourceInfoRequest(
    #         trialSourceInfo=bindings.v1TrialSourceInfo(
    #             checkpointUuid=checkpoint_uuid,
    #             trialId=self._trial_id,
    #             trialSourceInfoType=bindings.v1TrialSourceInfoType.INFERENCE,
    #             # modelId="asdasdf"
    #             # modelVersion=3
    #         )
    #     )
    #     bindings.post_ReportTrialSourceInfo(
    #         session=self._session,
    #         body=req,
    #     )
