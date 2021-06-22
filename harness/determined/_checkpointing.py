import enum

class Checkpointing:
    """
    Some checkpoint-related REST API wrappers.
    """

    def __init__(self, session, trial_id) -> None:
        self._session = session
        self._trial_id = trial_id

    def _report_checkpoint(self, uuid):
        # XXX: post this somewhere
        # self._session.post(...)
        pass
