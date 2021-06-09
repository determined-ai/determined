import enum
import sys
import time
from typing import Any, Dict, List, Optional, cast

from determined.common.experimental import checkpoint, session


class ExperimentState(enum.Enum):
    UNSPECIFIED = "STATE_UNSPECIFIED"
    ACTIVE = "STATE_ACTIVE"
    PAUSED = "STATE_PAUSED"
    STOPPING_COMPLETED = "STATE_STOPPING_COMPLETED"
    STOPPING_CANCELED = "STATE_STOPPING_CANCELED"
    STOPPING_ERROR = "STATE_STOPPING_ERROR"
    COMPLETED = "STATE_COMPLETED"
    CANCELED = "STATE_CANCELED"
    ERROR = "STATE_ERROR"
    DELETED = "STATE_DELETED"


class _GetExperimentResponse:
    def __init__(self, raw: Any):
        if not isinstance(raw, dict):
            raise ValueError(f"GetExperimentResponse must be a dict; got {raw}")
        if "config" not in raw:
            raise ValueError(f"GetExperimentResponse must have a config field; got {raw}")

        # We only parse the config and experiment.state because that is all the python sdk needs.

        config = raw["config"]
        if not isinstance(config, dict):
            raise ValueError(f'GetExperimentResponse["config"] must be a dict; got {config}')
        self.config = cast(Dict[str, Any], config)

        if "experiment" not in raw:
            raise ValueError(f"GetExperimentResponse must have an experiment field; got {raw}")
        exp = raw["experiment"]
        if not isinstance(exp, dict):
            raise ValueError(f'GetExperimentResponse["experiment"] must be a dict; got {exp}')
        if "state" not in exp:
            raise ValueError(f'GetExperimentResponse["experiment"] must have a state; got {exp}')
        state = exp["state"]
        if not isinstance(state, str):
            raise ValueError(
                f'GetExperimentResponse["experiment"]["state"] must be a str; got {state}'
            )

        self.state = ExperimentState(state)


class ExperimentReference:
    """
    An ExperimentReference object is usually obtained from
    ``determined.experimental.client.create_experiment()``
    or ``determined.experimental.client.get_experiment()``.

    Helper class that supports querying the set of checkpoints associated with an
    experiment.
    """

    def __init__(
        self,
        experiment_id: int,
        session: session.Session,
    ):
        self._id = experiment_id
        self._session = session

    @property
    def id(self) -> int:
        return self._id

    def _get(self) -> _GetExperimentResponse:
        """
        _get fetches the main GET experiment endpoint and parses the response.
        """
        exp_resp = self._session.get(f"/api/v1/experiments/{self.id}").json()
        return _GetExperimentResponse(exp_resp)

    def activate(self) -> None:
        self._session.post(f"/api/v1/experiments/{self.id}/activate")

    def archive(self) -> None:
        self._session.post(f"/api/v1/experiments/{self.id}/archive")

    def cancel(self) -> None:
        self._session.post(f"/api/v1/experiments/{self.id}/cancel")

    def delete(self) -> None:
        """
        Delete an experiment and all its artifacts from persistent storage.

        You must be authenticated as admin to delete an experiment.
        """
        self._session.delete(f"/api/v1/experiments/{self.id}")

    def get_config(self) -> Dict[str, Any]:
        return self._get().config

    def kill(self) -> None:
        self._session.post(f"/api/v1/experiments/{self.id}/kill")

    def pause(self) -> None:
        self._session.post(f"/api/v1/experiments/{self.id}/pause")

    def unarchive(self) -> None:
        self._session.post(f"/api/v1/experiments/{self.id}/unarchive")

    def wait(self, interval: int = 5) -> ExperimentState:
        """
        Wait for the experiment to reach a complete or terminal state.

        Arguments:
            interval (int, optional): An interval time in seconds before checking
                next experiement state.
        """
        elapsed_time = 0
        while True:
            exp = self._get()
            if exp.state in (
                ExperimentState.COMPLETED,
                ExperimentState.CANCELED,
                ExperimentState.DELETED,
                ExperimentState.ERROR,
            ):
                return exp.state
            elif exp.state == ExperimentState.PAUSED:
                raise ValueError(
                    f"Experiment {self.id} is in paused state. Make sure the experiment is active."
                )
            else:
                # ACTIVE, STOPPING_COMPLETED, etc.
                time.sleep(interval)
                elapsed_time += interval
                if elapsed_time % 60 == 0:
                    print(
                        f"Waiting for Experiment {self.id} to complete. "
                        f"Elapsed {elapsed_time / 60} minutes",
                        file=sys.stderr,
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
        r = self._session.get(
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
                checkpoint_refs.append(checkpoint.Checkpoint.from_json(ckpt, self._session))
                t_ids.add(ckpt["trialId"])

        return checkpoint_refs[:limit]

    def __repr__(self) -> str:
        return "Experiment(id={})".format(self.id)
