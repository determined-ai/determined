# PyTorch HP Search Constraints (MNIST) 
This tutorial shows how to use Determined's HP Search Constraints with 
PyTorch. In this example, the constraints are defined in Lines 56-57 of 
the `__init__` function in `model_def.py` based on the model hyperparameters
via the `det.InvalidHP` exception API (see the `HP Search Constraints` topic 
guide under https://docs.determined.ai/latest/topic-guides/index.html 

Constraints can also be defined in `train_batch` and `evaluate_batch`, 
where an InvalidHP exception can be raised based on 
training and validation metrics respectively.

This example is based on Determined's `mnist_pytorch` tutorial, with the
addition of the HP search constraint as the only modification.

## Files
* **model_def.py**: Where the HP Search constraint is defined and used.
* All other files are identical to the `mnist_pytorch` tutorial code. 

## To Run
If you have not yet installed Determined, installation instructions can be found
under `docs/install-admin.html` or at https://docs.determined.ai/latest/index.html

Run the following command: `det -m <master host:port> experiment create -f 
adaptive.yaml .`.

## Results
Training the model with the hyperparameter settings in `adaptive.yaml` should yield
a validation accuracy of ~97%. 
