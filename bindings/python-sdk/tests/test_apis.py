from pprint import pprint

import python_sdk
from swagger_client.models import Determinedexperimentv1State as Determinedexperimentv1State

def test_get_labels(det: object):
    labels = det.get_experiment_labels()
    print(labels)

def test_get_trial(det: object, trial_id: int):
    trial_resp = det.get_trial(trial_id = trial_id)
    pprint(trial_resp)

def test_get_experiment(det: object, exp_id: int):
    exp_resp = det.get_experiment(exp_id = exp_id)

    if exp_resp is None:
        print("Invalid Experiemnt ID")
    else:
        if (exp_resp.experiment.state == Determinedexperimentv1State.COMPLETED):
            print("Experiment {} is Complete".format(id))
        elif (exp_resp.experiment.state == Determinedexperimentv1State.PAUSED):
            raise ValueError("Experiment {} is not active".format(id))
        
    pprint(exp_resp)

def test_get_experiments(det: object):
    exp_resp = det.get_experiments()
    pprint(exp_resp)

def test_create_experiment_wait(det: object):
    exp_resp = det.create_experiement_from_file('../../../examples/tutorials/mnist_pytorch/const.yaml',
        '../../../examples/tutorials/mnist_pytorch',
        False,
    )

    if exp_resp is None:
        print("Failed to create an experiment")
    else:
        exp_id = exp_resp.experiment.id
        if exp_resp.experiment.id != 0:
            print("Created experiment: {}".format(exp_id))
            ret = det.activate_experiment(id = exp_id)
            if (ret != 0):
                print("Failed to activate experiment")
            else:
                print("Activated experiment: {}".format(exp_id))
    python_sdk._wait_for_experiment_complete(det, exp_id, 2)

def main():
    det = python_sdk.Core('detconfig.yaml')
    print("Creating an experiment")
    test_create_experiment_wait(det)
    print("Getting labels: ")
    test_get_labels(det)
    print("Getting experiments")
    test_get_experiments(det)
    print("Getting experiment: 1")
    test_get_experiment(det, 1)
    print("Getting trial: 1")
    test_get_trial(det, 1)
    
    input("press enter to continue...")
    
if __name__ == '__main__':
    main()
