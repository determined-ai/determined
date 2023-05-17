import random
import uuid
from typing import Dict, Union

import ray
from ray import air, tune
from ray.tune.schedulers import AsyncHyperBandScheduler

from determined.experimental import core_v2


class Trainable(tune.Trainable):
    def setup(self, config):
        self.hp = config["hp"]
        print(self._trial_info)
        experiment_name = self._trial_info.experiment_name
        trial_name = self.trial_name

        print(f"experiment name: {experiment_name} trial name: {trial_name}")

        core_v2.init(
            defaults=core_v2.DefaultConfig(
                name=experiment_name,
                hparams={
                    "hp": config["hp"],
                },
                searcher={
                    "name": "custom",
                    "metric": "loss",
                    "smaller_is_better": True,
                },
            ),
            unmanaged=core_v2.UnmanagedConfig(
                external_experiment_id=experiment_name,
                external_trial_id=trial_name,
            ),
        )

    def step(self):
        loss = self.hp + random.random()
        i = self.iteration
        print(f"metrics at step {i}:", {"loss": loss})
        core_v2.train.report_validation_metrics(steps_completed=i, metrics={"loss": loss})
        return {"loss": loss}

    def cleanup(self):
        core_v2.close()

    def save_checkpoint(self, checkpoint_dir: str):
        pass

    def load_checkpoint(self, checkpoint: Union[Dict, str]):
        pass


def main():
    ray.init()
    scheduler = AsyncHyperBandScheduler(grace_period=5, max_t=100)
    stopping_criteria = {"training_iteration": 1000}

    # Note: job ids from `ray.get_runtime_context().get_job_id()` are sequential within a cluster,
    # we need to find a better ray-provided uid.
    tuner = tune.Tuner(
        Trainable,
        run_config=air.RunConfig(
            name=f"ray-det-asha-test-{uuid.uuid4()}",
            stop=stopping_criteria,
            verbose=1,
        ),
        tune_config=tune.TuneConfig(
            metric="loss",
            mode="min",
            scheduler=scheduler,
            num_samples=20,
        ),
        param_space={
            "hp": tune.uniform(0, 2),
        },
    )
    results = tuner.fit()
    print("Best hyperparameters found were: ", results.get_best_result().config)


if __name__ == "__main__":
    main()
