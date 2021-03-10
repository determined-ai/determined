"""
High-level Sprinkle API for training with PyTorchTrial.
"""

# ExperimentConfig settings:
#
#   Backwards compatibility, where you only submit a Trial.
#       #launch_layer: python3 -m determined.launch.auto_horovod
#       #entrypoint_script: python3 -m determined.exec.harness
#       entrypoint: model_def:MyTrial
#
#   Basic usage:
#       #launch_layer: python3 -m determined.launch.auto_horovod
#       entrypoint_script: python3 train.py

### sprinkle api, training, PyTorchTrial:

context = det.pytorch.init()  # type: PyTorchContext

my_trial = MyPyTorchTrial(context.make_trial_context())

# returns metrics, but also report them to master when on cluster
metrics = context.train(
    my_trial,
    my_training_data,
    my_validation_data,

    # used when training locally but ignored by cluster training:
    max_len=Epochs(10)
)

######

### sprinkle api, inference, PyTorchTrial:

context = det.pytorch.init()

my_trial = ... # load via checkpoint export API

predictions = context.predict(
    my_trial,
    my_predict_data,

    # by default, predict_batch_fn would be my_trial.predict_batch
    predict_batch_fn=my_trial.predict_batch,
)

