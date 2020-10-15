import os
import sys

from determined.experimental import Determined


def get_validation_metric(checkpoint):
    config = checkpoint.experiment_config
    searcher = config['searcher']
    smaller_is_better = bool(searcher['smaller_is_better'])
    metric_name = searcher['metric']

    metrics = checkpoint.validation['metrics']
    metric = metrics['validationMetrics'][metric_name]
    return metric, smaller_is_better


def is_better(c1, c2):
    m1, smaller_is_better = get_validation_metric(c1)
    m2, _ = get_validation_metric(c2)
    if smaller_is_better and m1 < m2:
        return True
    return False


d = Determined(master='latest-master.determined.ai:8080')
model_name = 'dummy_model'

config = {
    'this': 'is_my_test_config'
}

context_dir = os.getcwd()

experiment = d.create_experiment({}, context_dir)
experiment.wait_for_completion()

if experiment.status != 'COMPLETED':
    print(f'Experiment {experiment.id} did not complete successfully')
    sys.exit(1)

best_checkpoint = experiment.top_checkpoint()

try:
    model = d.get_model(model_name)

except:  # Model not yet in registry
    print(f'Registering new Model: {model_name}')
    model = d.create_model(model_name)

latest_version = model.get_version()

if not latest_version:
    better = True
else:
    better = is_better(latest_version, best_checkpoint)

if better:
    print(f'Registering new version: {model_name}')
    model.register_version(best_checkpoint.uuid)




