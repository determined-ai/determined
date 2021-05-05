#import pprint
#from typing import Any, Dict, List, Optional

#from determined.common import api
from determined.common.experimental.determined import Determined
#from determined.common.experimental.checkpoint import Checkpoint
#from determined.common.experimental.experiment import ExperimentReference

def cancel_create_experiment() -> None:
    det = Determined(master='https://gcloud.determined.ai', user='determined')
    exp1 = det.get_experiment(experiment_id=1)    
    print("Cancelling experimet id: {}".format(exp1.id))
    exp1.cancel()

    exp2 = det.create_experiment(config_file='../tutorials/mnist_pytorch/const.yaml',
                                    model_dir='../tutorials/mnist_pytorch')
    exp2.activate()
    exp2.wait_till_complete()
    print("Experiment {} {} is complete".format(exp2.config['description'], exp2.id))

def main():
    cancel_create_experiment()

if __name__ == '__main__':
    main()

