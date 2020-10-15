import os

from determined.experimental import Determined

d = Determined(master='latest-master.determined.ai:8080')

config = {
    'this': 'is_my_test_config'
}

context_dir = os.getcwd()

experiment = d.create_experiment({}, context_dir)
experiment.wait_for_completion()

if experiment.status == 'COMPLETED':
    print(f'Experiment {experiment.id} completed Successfully!')
else:
    print(f'Experiment {experiment.id} did not complete successfully')


