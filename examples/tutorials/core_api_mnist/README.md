Core API MNIST Tutorial

This tutorial integrates the [PyTorch MNIST training example](https://github.com/pytorch/examples/blob/main/mnist/main.py) into Determined.

## Files
* **model_def.py**: The starting script (same as [main.py](https://github.com/pytorch/examples/blob/main/mnist/main.py) with no Determined integrations at all.
* **const.yaml**: A barebones config to run the `model_def.py` script on
  a Determined cluster.
* **model_def_metrics.py**: Modify `model_def.py` to report metrics to Determined.
* **metrics.yaml**: A config to run `model_def_metrics.py`
* **model_def_checkpoints.py**: Modify `model_def_metrics.py` to save and load to and from
  checkpoint storage.  This also introduces preemption support.
* **checkpoints.yaml**: A config to run `model_def_checkpoints.py`.
* **model_def_adaptive.py**: Modify `model_def_checkpoints.py` to participate in the
  hyperparameter search offered by the Determined master.
* **adaptive.yaml**: Use an adaptive ASHA search to optimize a hyperparameter.
* **model_def_distributed.py**: Introduce distributed training to our script.
* **distributed.yaml**: Run `model_def_distributed.py` across many nodes.

## To Run
If you have not yet installed Determined, installation instructions can be found
under `docs/install-admin.html` or at https://docs.determined.ai/latest/index.html

Run stage 0 with the following command:

    det -m <master host:port> experiment create -f const.yaml .

The other stages can be run by specifying the appropriate config file.
