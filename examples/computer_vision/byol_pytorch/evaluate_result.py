from argparse import ArgumentParser

from determined.experimental import client

if __name__ == "__main__":
    parser = ArgumentParser(
        description="Start an evaluation run (w/ classifier training) from the top checkpoint of a given experiment."
    )
    parser.add_argument("--experiment-id", type=int, required=True)
    parser.add_argument("--classifier-train-epochs", type=int, default=80)
    args = parser.parse_args()
    exp = client.get_experiment(args.experiment_id)
    config = dict(exp.get_config())
    print(sorted(list(config.keys())))
    config["name"] = config["name"] + "_evaluation"
    config["min_validation_period"] = {"epochs": args.classifier_train_epochs}
    config["searcher"]["max_length"]["epochs"] = args.classifier_train_epochs
    config["hyperparameters"]["training_mode"] = "CLASSIFIER_ONLY"
    config["hyperparameters"]["validate_with_classifier"] = True
    config["searcher"]["source_checkpoint_uuid"] = exp.top_checkpoint().uuid
    config["searcher"]["metric"] = "test_accuracy"
    config["searcher"]["smaller_is_better"] = False
    client.create_experiment(config, ".")
