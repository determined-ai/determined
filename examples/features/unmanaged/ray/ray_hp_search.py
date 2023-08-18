import random
import uuid

import ray
from ray import air, tune
from ray.tune.schedulers import AsyncHyperBandScheduler

from determined.experimental import core_v2


def objective(config):
    hp = config["hp"]

    experiment_name = air.session.get_experiment_name()
    trial_name = air.session.get_trial_name()
    print(f"experiment name: {experiment_name} trial name: {trial_name}")

    core_v2.init(
        defaults=core_v2.DefaultConfig(
            name=experiment_name,
            hparams={
                "hp": config["hp"],
            },
            # We need to pass a non-single searcher config to have the WebUI display our experiment
            # as HP search.
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

    try:
        for i in range(100):
            if (i + 1) % 10 == 0:
                loss = hp + random.random()
                print("metrics:", {"loss": loss})
                core_v2.train.report_validation_metrics(steps_completed=i, metrics={"loss": loss})
                air.session.report({"iterations": i, "loss": loss})
    finally:
        # Note: this is not called when ASHA terminates a "bad" trial, so these trials will
        # stay in the RUNNING state until they're marked as errored after a timeout.
        # `tune.Trainable` interface should be implemented to support a proper cleanup,
        # see `ray_hp_search_cleanup.py`.
        core_v2.close()


def main():
    ray.init()
    scheduler = AsyncHyperBandScheduler(grace_period=5, max_t=100)
    stopping_criteria = {"training_iteration": 1000}

    # Note: job ids from `ray.get_runtime_context().get_job_id()` are sequential within a cluster,
    # we need to find a better ray-provided uid.
    tuner = tune.Tuner(
        objective,
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
