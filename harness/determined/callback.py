from typing import Any, Dict, List


class Callback(object):
    """
    Generic callback interface to plug into Determined actions.
    """

    def on_trial_begin(self) -> None:
        """Executed before the start of the first training step of a trial."""
        pass

    def on_train_step_begin(
        self, step_id: int, num_batches: int, total_batches_processed: int
    ) -> None:
        """Executed at the beginning of a training step."""
        pass

    def on_train_step_end(
        self,
        step_id: int,
        num_batches: int,
        total_batches_processed: int,
        metrics: List[Dict[str, Any]],
    ) -> None:
        """
        Executed at the end of a training step.

        Args:
            metrics: A list of Python dictionaries for this training step,
                     where each dictionary contains the metrics of a single
                     training batch.
        """
        pass

    def on_validation_step_begin(self, step_id: int, total_batches_processed: int) -> None:
        """Executed at the beginning of a validation step."""
        pass

    def on_validation_step_end(
        self, step_id: int, total_batches_processed: int, metrics: Dict[str, Any]
    ) -> None:
        """
        Executed at the end of a validation step.

        Args:
            metrics: A Python dictionary that contains the metrics
                     for this validation step.
        """
        pass
