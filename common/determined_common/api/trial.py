from typing import List, Optional

from determined_common import api, check, util
from determined_common.api import Checkpoint
from determined_common.api import authentication as auth
from determined_common.api import gql


class Trial(object):
    def __init__(
        self,
        trial_id: int,
        user: Optional[str] = None,
        master: Optional[str] = None,
        attempt_auth: bool = True,
    ):
        self.id = trial_id

        if not master:
            master = util.get_default_master_address()

        self._master = master
        if attempt_auth:
            auth.initialize_session(self._master, user, try_reauth=True)

    def top_checkpoint(
        self, sort_by: Optional[str] = None, smaller_is_better: Optional[bool] = None,
    ) -> Checkpoint:
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
    ) -> Checkpoint:
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

        if (sort_by is not None or smaller_is_better is not None) and not best:
            raise AssertionError(
                "sort_by and smaller_is_better parameters must be used with the --best option set"
            )

        if not self._master:
            self._master = util.get_default_master_address()

        auth.initialize_session(self._master, None, try_reauth=True)

        q = api.GraphQLQuery(self._master)

        if sort_by is not None:
            checkpoint = q.op.best_checkpoint_by_metric(
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

            checkpoint = q.op.checkpoints(where=where, order_by=order_by, limit=1)

        checkpoint.state()
        checkpoint.uuid()
        checkpoint.resources()

        validation = checkpoint.validation()
        validation.metrics()
        validation.state()

        step = checkpoint.step()
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
        return Checkpoint(
            ckpt_gql.uuid,
            ckpt_gql.step.trial.experiment.config["checkpoint_storage"],
            batch_number,
            ckpt_gql.step.start_time,
            ckpt_gql.step.end_time,
            ckpt_gql.resources,
            ckpt_gql.validation,
        )
