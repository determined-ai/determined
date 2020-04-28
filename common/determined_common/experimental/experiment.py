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
        q = api.GraphQLQuery(self._master)
        exp = q.op.experiments_by_pk(id=self.id)
        checkpoints = exp.best_checkpoints_by_metric(
            args={"lim": limit, "metric": sort_by, "smaller_is_better": smaller_is_better}
        )

        checkpoints.state()
        checkpoints.uuid()
        checkpoints.resources()

        validation = checkpoints.validation()
        validation.metrics()
        validation.state()

        step = checkpoints.step()
        step.id()
        step.start_time()
        step.end_time()
        step.trial.experiment.config()
        step.trial.id()

        resp = q.send()

        checkpoints_resp = resp.experiments_by_pk.best_checkpoints_by_metric

        if not checkpoints_resp:
            return []

        experiment_conf = checkpoints_resp[0].step.trial.experiment.config
        sib = (
            smaller_is_better
            if smaller_is_better is not None
            else experiment_conf["searcher"]["smaller_is_better"]
        )

        sort_metric = sort_by if sort_by is not None else experiment_conf["searcher"]["metric"]
        ordered_checkpoints = sorted(
            checkpoints_resp,
            key=lambda c: c.validation.metrics["validation_metrics"][sort_metric],
            reverse=not sib,
        )

        return [
            checkpoint.Checkpoint(
                ckpt.uuid,
                ckpt.step.trial.experiment.config["checkpoint_storage"],
                ckpt.step.trial.experiment.config["batches_per_step"] * ckpt.step.id,
                ckpt.step.start_time,
                ckpt.step.end_time,
                ckpt.resources,
                ckpt.validation,
            )
            for ckpt in ordered_checkpoints
        ]
