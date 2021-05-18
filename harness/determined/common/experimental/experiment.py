import time
from typing import List, Optional

from determined._swagger.client.api.experiments_api import ExperimentsApi
from determined._swagger.client.models.determinedexperimentv1_state import (
    Determinedexperimentv1State,
)
from determined.common import api
from determined.common.experimental import checkpoint


class ExperimentReference:
    """
    Helper class that supports querying the set of checkpoints associated with an
    experiment.

    Arguments:
        experiment_id (int): The ID of this experiment.
    """

    def __init__(
        self,
        experiment_id: int,
        master: str,
        api_ref: ExperimentsApi,
    ):
        self.id = experiment_id
        self._master = master
        self._experiments = api_ref

    def activate(self) -> None:
        self._experiments.determined_activate_experiment(id=self.id)

    def archive(self) -> None:
        self._experiments.determined_archive_experiment(id=self.id)

    def cancel(self) -> None:
        self._experiments.determined_cancel_experiment(id=self.id)

    def delete(self) -> None:
        self._experiments.determined_delete_experiment(id=self.id)

    def get_config(self) -> object:
        exp_resp = self._experiments.determined_get_experiment(experiment_id=self.id)
        return exp_resp.config

    def kill(self) -> None:
        self._experiments.determined_kill_experiment(id=self.id)

    def pause(self) -> None:
        self._experiments.determined_pause_experiment(id=self.id)

    def unarchive(self) -> None:
        self._experiments.determined_unarchive_experiment(id=self.id)

    def wait(self, interval: int = 5) -> None:
        """
        Wait for experiment to reach complete or a terminal state.

        Arguments:
            interval (int, optional): An interval time in seconds before checking
            next experiement state.
        """
        elapsed_time = 0
        while True:
            exp_resp = self._experiments.determined_get_experiment(experiment_id=self.id)
            if exp_resp.experiment.state in (
                Determinedexperimentv1State.COMPLETED,
                Determinedexperimentv1State.CANCELED,
                Determinedexperimentv1State.DELETED,
                Determinedexperimentv1State.ERROR,
            ):
                break
            elif exp_resp.experiment.state == Determinedexperimentv1State.PAUSED:
                raise ValueError(
                    "Experiment {} is in paused state. Make sure the experiment is active.".format(
                        self.id
                    )
                )
            else:
                # ACTIVE, STOPPING_COMPLETED, etc.
                time.sleep(interval)
                elapsed_time += interval
                if elapsed_time % 60 == 0:
                    print(
                        "Waiting for Experiment {} to complete. Elapsed {} minutes".format(
                            self.id, elapsed_time / 60
                        )
                    )

    def top_checkpoint(
        self,
        sort_by: Optional[str] = None,
        smaller_is_better: Optional[bool] = None,
    ) -> checkpoint.Checkpoint:
        """
        Return the :class:`~determined.experimental.Checkpoint` for this experiment that
        has the best validation metric, as defined by the ``sort_by`` and ``smaller_is_better``
        arguments.

        Arguments:
            sort_by (string, optional): The name of the validation metric to
                order checkpoints by. If this parameter is not specified, the metric
                defined in the experiment configuration ``searcher`` field will be used.

            smaller_is_better (bool, optional): Specifies whether to sort the
                metric above in ascending or descending order. If ``sort_by`` is unset,
                this parameter is ignored. By default, the value of ``smaller_is_better``
                from the experiment's configuration is used.
        """
        checkpoints = self.top_n_checkpoints(
            1, sort_by=sort_by, smaller_is_better=smaller_is_better
        )

        if not checkpoints:
            raise AssertionError("No checkpoints found for experiment {}".format(self.id))

        return checkpoints[0]

    def top_n_checkpoints(
        self,
        limit: int,
        sort_by: Optional[str] = None,
        smaller_is_better: Optional[bool] = None,
    ) -> List[checkpoint.Checkpoint]:
        """
        Return the N :class:`~determined.experimental.Checkpoint` instances with the best
        validation metrics, as defined by the ``sort_by`` and ``smaller_is_better``
        arguments. This method will return the best checkpoint from the
        top N best-performing distinct trials of the experiment. Only checkpoints in
        a ``COMPLETED`` state with a matching ``COMPLETED`` validation are considered.

        Arguments:
            limit (int): The maximum number of checkpoints to return.

            sort_by (string, optional): The name of the validation metric to use for
                sorting checkpoints. If this parameter is unset, the metric defined
                in the experiment configuration searcher field will be
                used.

            smaller_is_better (bool, optional): Specifies whether to sort the
                metric above in ascending or descending order. If ``sort_by`` is unset,
                this parameter is ignored. By default, the value of ``smaller_is_better``
                from the experiment's configuration is used.
        """
        r = api.get(
            self._master,
            "/api/v1/experiments/{}/checkpoints".format(self.id),
            params={
                "states": checkpoint.CheckpointState.COMPLETED.value,
                "validation_states": checkpoint.CheckpointState.COMPLETED.value,
            },
        )
        checkpoints = r.json()["checkpoints"]

        if not checkpoints:
            raise AssertionError("No checkpoint found for experiment {}".format(self.id))

        if not sort_by:
            sort_by = checkpoints[0]["experimentConfig"]["searcher"]["metric"]
            smaller_is_better = checkpoints[0]["experimentConfig"]["searcher"]["smaller_is_better"]

        checkpoints.sort(
            reverse=not smaller_is_better,
            key=lambda x: (x["metrics"]["validationMetrics"][sort_by], x["trialId"]),
        )

        # Ensure returned checkpoints are from distinct trials.
        t_ids = set()
        checkpoint_refs = []
        for ckpt in checkpoints:
            if ckpt["trialId"] not in t_ids:
                checkpoint_refs.append(checkpoint.Checkpoint.from_json(ckpt, self._master))
                t_ids.add(ckpt["trialId"])

        return checkpoint_refs[:limit]

    def __repr__(self) -> str:
        return "Experiment(id={})".format(self.id)
