from typing import Optional

from determined.common import check
from determined.common.experimental import checkpoint, session


class TrialReference:
    """
    A TrialReference object is usually obtained from
    ``determined.experimental.client.get_trial()``.

    Trial reference class used for querying relevant
    :class:`~determined.experimental.Checkpoint` instances.
    """

    def __init__(self, trial_id: int, session: session.Session):
        self.id = trial_id
        self._session = session

    def kill(self) -> None:
        self._session.post(f"/api/v1/trials/{self.id}/kill")

    def top_checkpoint(
        self,
        sort_by: Optional[str] = None,
        smaller_is_better: Optional[bool] = None,
    ) -> checkpoint.Checkpoint:
        """
        Return the :class:`~determined.experimental.Checkpoint` instance with the best
        validation metric as defined by the ``sort_by`` and ``smaller_is_better``
        arguments.

        Arguments:
            sort_by (string, optional): The name of the validation metric to
                order checkpoints by. If this parameter is unset the metric defined
                in the related experiment configuration searcher field will be
                used.

            smaller_is_better (bool, optional): Whether to sort the
                metric above in ascending or descending order. If ``sort_by`` is unset,
                this parameter is ignored. By default, the value of ``smaller_is_better``
                from the experiment's configuration is used.
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
        Return the :class:`~determined.experimental.Checkpoint` instance with the best
        validation metric as defined by the ``sort_by`` and ``smaller_is_better``
        arguments.

        Exactly one of the ``best``, ``latest``, or ``uuid`` parameters must be set.

        Arguments:
            latest (bool, optional): Return the most recent checkpoint.

            best (bool, optional): Return the checkpoint with the best validation
                metric as defined by the ``sort_by`` and ``smaller_is_better``
                arguments. If ``sort_by`` and ``smaller_is_better`` are not
                specified, the values from the associated experiment
                configuration will be used.

            uuid (string, optional): Return the checkpoint for the specified UUID.

            sort_by (string, optional): The name of the validation metric to
                order checkpoints by. If this parameter is unset the metric defined
                in the related experiment configuration searcher field will be
                used.

            smaller_is_better (bool, optional): Whether to sort the
                metric above in ascending or descending order. If ``sort_by`` is unset,
                this parameter is ignored. By default, the value of ``smaller_is_better``
                from the experiment's configuration is used.
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
                "`sort_by` and `smaller_is_better` parameters can only be used with `best`"
            )

        if uuid:
            resp = self._session.get("/api/v1/checkpoints/{}".format(uuid))
            return checkpoint.Checkpoint.from_json(resp.json()["checkpoint"], self._session)

        r = self._session.get(
            "/api/v1/trials/{}/checkpoints".format(self.id),
            # The default sort order from the API is by batch number. The order
            # by parameter indicates descending order.
            params={"order_by": 2},
        ).json()
        checkpoints = r["checkpoints"]

        if not checkpoints:
            raise AssertionError("No checkpoint found for trial {}".format(self.id))

        if latest:
            return checkpoint.Checkpoint.from_json(checkpoints[0], self._session)

        if not sort_by:
            sort_by = checkpoints[0]["experimentConfig"]["searcher"]["metric"]
            smaller_is_better = checkpoints[0]["experimentConfig"]["searcher"]["smaller_is_better"]

        best_checkpoint_func = min if smaller_is_better else max
        return checkpoint.Checkpoint.from_json(
            best_checkpoint_func(
                [c for c in checkpoints if c["metrics"] is not None],
                key=lambda x: x["metrics"]["validationMetrics"][sort_by],
            ),
            self._session,
        )

    def __repr__(self) -> str:
        return "Trial(id={})".format(self.id)
