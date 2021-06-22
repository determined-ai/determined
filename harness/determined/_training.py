import enum
from typing import Optional

class EarlyExitReason(enum.Enum):
    INVALID_HP = "INVALID_HP"
    INCOMPLETE = "INCOMPLETE"

class Training:
    """
    Some training-related REST API wrappers.
    """

    def __init__(self, session, trial_id, run_id) -> None:
        self._session = session
        self._trial_id = trial_id
        self._run_id = run_id

    def set_status(self, status: str) -> None:
        # XXX: post this somewhere
        # self._session.post(...)
        pass

    def get_last_validation(self) -> Optional[int]:
        # XXX: post this somewhere
        # self._session.post(...)
        return None

    # XXX: "total_batches"
    def report_training_metrics(self, batches_trained, metrics):
        print("REPORTING TRAINING METRICS")
        # XXX: batch metrics
        # XXX: total_records
        # XXX: total_epochs
        body = {
            "trial_run_id": self._run_id,
            "total_batches": batches_trained,
            "metrics": metrics,
        }
        self._session.post(f"/api/v1/trials/{self._trial_id}/training_metrics", body=body)

    # XXX: "total_batches"
    def report_validation_metrics(self, batches_trained, metrics):
        # XXX: batch metrics
        # XXX: total_records
        # XXX: total_epochs
        body = {
            "trial_run_id": self._run_id,
            "total_batches": batches_trained,
            "metrics": metrics,
        }
        self._session.post(f"/api/v1/trials/{self._trial_id}/validation_metrics", body=body)

    def report_early_exit(self, reason):
        # XXX: post this somewhere
        # self._session.post(...)
        pass

    def is_best_validation_of_experiment(self, metric):
        # XXX: post this somewhere
        return False
