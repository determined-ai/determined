from typing import List, Optional

from determined_common import api
from determined_common.experimental import checkpoint


class ExperimentReference:
    """
    Experiment reference class used for querying relevant
    :class:`~determined.experimental.Checkpoint` instances.

    Arguments:
        experiment_id (int): The experiment ID.
        master (string, optional): The URL of the Determined master. If this
            class is obtained via :class:`determined.experimental.Determined`, the
            master URL is automatically passed into this constructor.
    """

    def __init__(self, experiment_id: int, master: str):
        self.id = experiment_id
        self._master = master

    def top_checkpoint(
        self, sort_by: Optional[str] = None, smaller_is_better: Optional[bool] = None,
    ) -> checkpoint.Checkpoint:
        """
        Return the :class:`~determined.experimental.Checkpoint` instance with the best
        validation metric as defined by the ``sort_by`` and ``smaller_is_better``
        arguments.

        Arguments:
            sort_by (string, optional): The name of the validation metric to
                order checkpoints by. If this parameter is unset the metric defined
                in the experiment configuration searcher field will be
                used.

            smaller_is_better (bool, optional): Specifies whether to sort the
                metric above in ascending or descending order. If sort_by is unset,
                this parameter is ignored. By default the smaller_is_better value
                in the experiment configuration is used.
        """
        checkpoints = self.top_n_checkpoints(
            1, sort_by=sort_by, smaller_is_better=smaller_is_better
        )

        if not checkpoints:
            raise AssertionError("No checkpoints found for experiment {}".format(self.id))

        return checkpoints[0]

    def top_n_checkpoints(
        self, limit: int, sort_by: Optional[str] = None, smaller_is_better: Optional[bool] = None,
    ) -> List[checkpoint.Checkpoint]:
        """
        Return the N :class:`~determined.experimental.Checkpoint` instances with the best
        validation metric values as defined by the ``sort_by`` and ``smaller_is_better``
        arguments. This method will return the best checkpoint from the
        top N performing distinct trials of the experiment. Only checkpoints in
        a COMPLETED state with a matching COMPLETED validation are considered.

        Arguments:
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
            reverse=not smaller_is_better, key=lambda x: x["metrics"]["validationMetrics"][sort_by]
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
