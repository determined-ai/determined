import time
from typing import List, Optional

import determined_client

from determined_common import api
from determined_common.experimental import checkpoint


class Experiment:
    """
    Helper class that supports querying the set of checkpoints associated with an
    experiment.

    Arguments:
        experiment_id (int): The ID of this experiment.
        master (string, optional): The URL of the Determined master. If this
            class is obtained via :class:`~determined.experimental.Determined`, the
            master URL is automatically passed into this constructor.
    """

    def __init__(self, api_client, experiment_data=None, config=None):
        self.id = None
        self.state = None

        self.config = config
        self.api_client = api_client
        self.metric = config.get('searcher').get('metric')
        self.smaller_is_better = config.get('searcher').get('smaller_is_better')

        for attribute in experiment_data.attribute_map:
            setattr(self, attribute, getattr(experiment_data, attribute))


    @classmethod
    def create_experiment(cls, api_client, config, context_dir, local=False, test=False, master=""):
        print("Creating Experiment")
        experiment_api = determined_client.ExperimentsApi(api_client)
        experiment_api.determined_post_experiment()
        # experiment = api.create_experiment()
        # experiment_id = 1  # really should be experiment.id
        # print(f"Created Experiment {experiment_id}")
        # return cls(api_client, experiment_id)

    @classmethod
    def get_experiment(cls, api_client, experiment_id):
        experiment_api = determined_client.ExperimentsApi(api_client)
        api_response = experiment_api.determined_get_experiment(experiment_id)
        return cls(api_client, api_response.experiment, api_response.config)

    # @property
    # def status(self) -> str:
    #     # status = api.get_experiment_status()
    #     status = "COMPLETED"
    #     return status

    def success(self):
        if self.state == 'STATE_COMPLETED':
            return True

        return False

    def wait_for_completion(self):
        while self.state == "ACTIVE":
            time.sleep(10)

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
