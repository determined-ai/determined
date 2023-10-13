# Core API Tutorial
This tutorial shows how to take an existing training script and integrate it
with the Determined platform in a series of steps.  To keep the focus of the
tutorial on the Core API mechanics, this tutorial is based on a script that
increments an integer, rather than training any real model.

## Files
* **0_start.py**: A starting script with no Determined integrations at all.
* **0_start.yaml**: A barebones config to run the `0_start.py` script on
  a Determined cluster.
* **1_metrics.py**: Modify `0_start.py` to report metrics to Determined.
* **1_metrics.yaml**: A config to run `1_metrics.py`
* **2_checkpoints.py**: Modify `1_metrics.py` to save and load to and from
  checkpoint storage.  This also introduces preemption support.
* **2_checkpoints.yaml**: A config to run `2_checkpoints.py`.
* **3_hpsearch.py**: Modify `2_checkpoints.py` to participate in the
  hyperparameter search offered by the Determined master.
* **3_hpsearch.yaml**: Use an adaptive ASHA search to optimize a hyperparameter.
* **4_distributed.py**: Introduce distributed training to our script.
* **4_distributed.yaml**: Run `4_distributed.py` across many nodes.

## To Run
If you have not yet installed Determined, installation instructions can be found
under `docs/install-admin.html` or at https://docs.determined.ai/latest/index.html

Run stage 0 with the following command:

    det -m <master host:port> experiment create -f 0_start.yaml .

The other stages can be run by specifying the appropriate config file.
