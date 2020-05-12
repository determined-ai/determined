from typing import List, Optional

from determined_common import api
from determined_common.experimental import checkpoint


class ExperimentReference:
    """
    Experiment reference class used for querying relevant
    :py:class:`det.experimental.Checkpoint` instances.

    Arguments:
        experiment_id (int): The experiment ID.
        master (string, optional): The URL of the Determined master. If this
            class is obtained via :py:class:`det.experimental.Determined` the
            master URL is automatically passed into this constructor.
    """

    def __init__(self, experiment_id: int, master: str):
        self.id = experiment_id
        self._master = master

    def top_checkpoint(
        self, sort_by: Optional[str] = None, smaller_is_better: Optional[bool] = None,
    ) -> checkpoint.Checkpoint:
        """
        Return the :py:class:`det.experimental.Checkpoint` instance with the best
        validation metric as defined by the `sort_by` and `smaller_is_better`
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
        self, limit: int, sort_by: Optional[str] = None, smaller_is_better: Optional[bool] = None
    ) -> List[checkpoint.Checkpoint]:
        """
        Return the N :py:class:`det.experimental.Checkpoint` instances with the best
        validation metric values as defined by the `sort_by` and `smaller_is_better`
        arguments. This command will return the best checkpoint from the
        top N performing distinct trials of the experiment.

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
        r = api.get(self._master, "checkpoints", params={"experiment_id": self.id}).json()

        if not r:
            raise AssertionError("No checkpoint found for trial {}".format(self.id))

        if not sort_by:
            sort_by = r[0]["metric"]
            smaller_is_better = r[0]["smaller_is_better"]

        r.sort(
            reverse=not smaller_is_better, key=lambda x: x["metrics"]["validation_metrics"][sort_by]
        )

        return [checkpoint.from_json(ckpt) for ckpt in r[:limit]]
