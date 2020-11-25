import time
from typing import List, Optional

import determined_client
from determined_client.models.determinedexperimentv1_state import (
    Determinedexperimentv1State as States,
)

from determined_common import api, context
from determined_common.experimental import checkpoint
from determined_common.experimental.trial import TrialReference


class ExperimentReference:
    """
    Helper class that supports querying the set of checkpoints associated with an
    experiment.

    Arguments:
        experiment_id (int): The ID of this experiment.
        master (string, optional): The URL of the Determined master. If this
            class is obtained via :class:`~determined.experimental.Determined`, the
            master URL is automatically passed into this constructor.
    """

    def __init__(self, api_client, master, config=None, experiment_data=None):
        self.master = master
        self.api_client = api_client
        self.experiments_api = determined_client.ExperimentsApi(self.api_client)

        self.id = None
        self.config = config

        for attribute in experiment_data:
            if attribute != "state":
                setattr(self, attribute, experiment_data[attribute])

    def get_status(self):
        experiment = self.experiments_api.determined_get_experiment(self.id)
        return experiment.experiment.state

    def is_active(self):
        if self.get_status() == States.ACTIVE:
            return True

        return False

    def wait(self):
        while self.get_status() == States.ACTIVE:
            time.sleep(10)

    def activate(self):
        experiment_api = determined_client.ExperimentsApi(self.api_client)
        experiment_api.determined_activate_experiment(self.id)

        while self.get_status() == States.PAUSED:
            time.sleep(1)

    def get_trials(self):
        experiment_api = determined_client.ExperimentsApi(self.api_client)
        trials = []
        trials_response = experiment_api.determined_get_experiment_trials(self.id)
        for trial in trials_response.trials:
            trials.append(TrialReference.from_spec(self.api_client, trial))

        return trials

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
        checkpoint_response = self.experiments_api.determined_get_experiment_checkpoints(self.id)
        checkpoints = checkpoint_response.checkpoints

        if not checkpoints:
            raise AssertionError("No checkpoint found for experiment {}".format(self.id))

        if not sort_by:
            sort_by = checkpoints[0].experiment_config["searcher"]["metric"]
            smaller_is_better = checkpoints[0].experiment_config["searcher"]["smaller_is_better"]

        checkpoints.sort(
            reverse=not smaller_is_better, key=lambda x: x.metrics.validation_metrics[sort_by]
        )

        # Ensure returned checkpoints are from distinct trials.
        t_ids = set()
        checkpoint_refs = []
        for ckpt in checkpoints:
            if ckpt.trial_id not in t_ids:
                checkpoint_refs.append(checkpoint.Checkpoint.from_spec(self.api_client, ckpt))
                t_ids.add(ckpt.trial_id)

        return checkpoint_refs[:limit]

    def __repr__(self) -> str:
        return "Experiment(id={})".format(self.id)

    @staticmethod
    def path_to_files(path):
        files = []
        for item in context.read_context(path)[0]:
            content = item["content"].decode("ascii")
            file = determined_client.models.V1File(
                path=item["path"],
                type=item["type"],
                content=content,
                mtime=item["mtime"],
                uid=item["uid"],
                gid=item["gid"],
                mode=item["mode"],
            )
            files.append(file)
        return files
