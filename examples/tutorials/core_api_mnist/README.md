# Core API MNIST Tutorial

This tutorial integrates the [PyTorch MNIST training example](https://github.com/pytorch/examples/blob/main/mnist/main.py) into Determined.

## Files

### Initial Step

* **model_def.py**: The starting script (same as [main.py](https://github.com/pytorch/examples/blob/main/mnist/main.py) without the Determined integrations.
* **const.yaml**: A bare-bones configuration file to run the `model_def.py` script on
  a Determined cluster.

### Metric Reporting

* **model_def_metrics.py**: Modify `model_def.py` to report metrics to Determined.
* **metrics.yaml**: A configuration file to run `model_def_metrics.py`

### Checkpointing

* **model_def_checkpoints.py**: Modify `model_def_metrics.py` to save and load to and from
  checkpoint storage.  This also introduces preemption support.
* **checkpoints.yaml**: A configuration file to run `model_def_checkpoints.py`.

### Hyperparameter Search

* **model_def_adaptive.py**: Modify `model_def_checkpoints.py` to participate in the
  hyperparameter search offered by the Determined master.
* **adaptive.yaml**: Use an adaptive ASHA search to optimize a hyperparameter.

### Distributed Training

* **model_def_distributed.py**: Introduce distributed training to our script.
* **distributed.yaml**: Run `model_def_distributed.py` across many nodes.

## To Run

If you have not yet installed Determined, visit `docs/install-admin.html` or https://docs.determined.ai/latest/index.html for installation instructions.

Run the initial step with the following command:

    det -m <master host:port> experiment create -f const.yaml .

To run the other steps, specifying the appropriate config file.
