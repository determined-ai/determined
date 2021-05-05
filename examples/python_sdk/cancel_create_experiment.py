from determined.common.experimental.determined import Determined

def cancel_experiment() -> None:
    det = Determined(master='https://gcloud.determined.ai', user='determined')
    exp1 = det.get_experiment(experiment_id=1)    
    print("Cancelling experimet id: {}".format(exp1.id))
    exp1.cancel()

def create_experiment_wait() -> None:
    exp2 = det.create_experiment(exp_config='../tutorials/mnist_pytorch/const.yaml',
                                    model_dir='../tutorials/mnist_pytorch')
    exp2.activate()
    exp2.wait_till_complete()
    print("Experiment {} {} is complete".format(exp2.config['description'], exp2.id))

def main():
    cancel_experiment()
    create_experiment_wait()

if __name__ == '__main__':
    main()

