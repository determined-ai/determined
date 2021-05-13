from determined.common import yaml
from determined.common.experimental.determined import Determined


def cancel_experiment(D: Determined) -> None:
    exp = D.get_experiment(experiment_id=1)
    print("Cancelling experimet id: {}".format(exp.id))
    exp.cancel()


def create_experiment_wait(D: Determined) -> None:
    exp = D.create_experiment(
        exp_config="../tutorials/mnist_pytorch/const.yaml",
        model_dir="../tutorials/mnist_pytorch",
    )
    exp.activate()
    exp.wait_till_complete()
    print("Experiment {} {} is complete".format(exp.config["description"], exp.id))


def create_experiment_custom_config(D: Determined) -> None:
    config_file = open("../tutorials/mnist_pytorch/const.yaml", "r")
    myconfig = yaml.safe_load(config_file.read())
    config_file.close()

    myconfig["hyperparameters"]["leraning_rate"] = 0.5

    exp = D.create_experiment(
        exp_config=myconfig, model_dir="../tutorials/mnist_pytorch"
    )
    exp.activate()
    print("Created experimet id: {}".format(exp.id))


def main():
    D = Determined(master="http://localhost:8080", user="determined")
    cancel_experiment(D)
    create_experiment_wait(D)
    create_experiment_custom_config(D)


if __name__ == "__main__":
    main()
