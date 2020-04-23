from typing import List, Optional

from determined_common import api, check
from determined_common.api import gql
from determined_common.experimental import checkpoint


class TrialReference:
    """
    Trial reference class used for querying relevant
    :py:class:`det.experimental.Checkpoint` instances.

    Arguments:
        trial_id (int): the trial ID.
        master (string, optional): The URL of the Determined master. If this
            class is obtained via :py:class:`det.experimental.Determined` the
            master URL is automatically passed into this constructor.
    """

    def __init__(self, trial_id: int, master: str):
        self.id = trial_id
        self._master = master

    def top_checkpoint(
        self, sort_by: Optional[str] = None, smaller_is_better: Optional[bool] = None,
    ) -> checkpoint.Checkpoint:
        """
        Return the :py:class:`det.experimental.Checkpoint` instance with the best
        validation metric as defined by the `sort_by` and `smaller_is_better`
        arguments.

        Arguments:
            sort_by (string, optional): the name of the validation metric to
                order checkpoints by. If this parameter is unset the metric defined
                in the related experiment configuration searcher field will be
                used.

            smaller_is_better (bool, optional): specifies whether to sort the
                metric above in ascending or descending order. If sort_by is unset,
                this parameter is ignored. By default the smaller_is_better value
                in the related experiment configuration is used.
        """
        return self.select_checkpoint(
            best=True, sort_by=sort_by, smaller_is_better=smaller_is_better
        )

    def select_checkpoint(
        self,
        latest: bool = False,
        best: bool = False,
        uuid: Optional[str] = None,
        sort_by: Optional[str] = None,
        smaller_is_better: Optional[bool] = None,
    ) -> checkpoint.Checkpoint:
        """
        Return the :py:class:`det.experimental.Checkpoint` instance with the best
        validation metric as defined by the `sort_by` and `smaller_is_better`
        arguments.

        Exactly one of the best, latest, or uuid parameters must be set.

        Arguments:
            latest (bool, optional): return the most recent checkpoint.

            best (bool, optional): return the checkpoint with the best validation
                metric as defined by the `sort_by` and `smaller_is_better`
                arguments. If `sort_by` and `smaller_is_better` are not
                specified, the values from the associated experiment
                configuration will be used.

            uuid (string, optional): return the checkpoint for the specified uuid.

            sort_by (string, optional): the name of the validation metric to
                order checkpoints by. If this parameter is unset the metric defined
                in the related experiment configuration searcher field will be
                used.

            smaller_is_better (bool, optional): specifies whether to sort the
                metric above in ascending or descending order. If sort_by is unset,
                this parameter is ignored. By default the smaller_is_better value
                in the related experiment configuration is used.
        """
        check.eq(
            sum([int(latest), int(best), int(uuid is not None)]),
            1,
            "Exactly one of latest, best, or uuid must be set",
        )

        check.eq(
            sort_by is None,
            smaller_is_better is None,
            "sort_by and smaller_is_better must be set together",
        )

        if sort_by is not None and not best:
            raise AssertionError(
                "sort_by and smaller_is_better parameters can only be used with --best"
            )

        q = api.GraphQLQuery(self._master)

        if sort_by is not None:
            checkpoint_gql = q.op.best_checkpoint_by_metric(
                args={"tid": self.id, "metric": sort_by, "smaller_is_better": smaller_is_better},
            )
        else:
            where = gql.checkpoints_bool_exp(
                state=gql.checkpoint_state_comparison_exp(_eq="COMPLETED"),
                trial_id=gql.Int_comparison_exp(_eq=self.id),
            )

            order_by = []  # type: List[gql.checkpoints_order_by]
            if uuid is not None:
                where.uuid = gql.uuid_comparison_exp(_eq=uuid)
            elif latest:
                order_by = [gql.checkpoints_order_by(end_time=gql.order_by.desc)]
            elif best:
                where.validation = gql.validations_bool_exp(
                    state=gql.validation_state_comparison_exp(_eq="COMPLETED")
                )
                order_by = [
                    gql.checkpoints_order_by(
                        validation=gql.validations_order_by(
                            metric_values=gql.validation_metrics_order_by(signed=gql.order_by.asc)
                        )
                    )
                ]

            checkpoint_gql = q.op.checkpoints(where=where, order_by=order_by, limit=1)

        checkpoint_gql.state()
        checkpoint_gql.uuid()
        checkpoint_gql.resources()

        validation = checkpoint_gql.validation()
        validation.metrics()
        validation.state()

        step = checkpoint_gql.step()
        step.id()
        step.start_time()
        step.end_time()
        step.trial.experiment.config()

        resp = q.send()

        result = resp.best_checkpoint_by_metric if sort_by is not None else resp.checkpoints

        if not result:
            raise AssertionError("No checkpoint found for trial {}".format(self.id))

        ckpt_gql = result[0]
        batch_number = ckpt_gql.step.trial.experiment.config["batches_per_step"] * ckpt_gql.step.id
        return checkpoint.Checkpoint(
            ckpt_gql.uuid,
            ckpt_gql.step.trial.experiment.config["checkpoint_storage"],
            batch_number,
            ckpt_gql.step.start_time,
            ckpt_gql.step.end_time,
            ckpt_gql.resources,
            ckpt_gql.validation,
        )
