# DeepSpeed CIFAR Example
This example is adapted from the 
[CIFAR example in the DeepSpeedExamples](https://github.com/microsoft/DeepSpeedExamples/tree/master/cifar) 
repository. It is intended to demonstrate a simple usecase of DeepSpeed with Determined.

## Files
* **model_def.py**: The core code for the model. This includes building and compiling the model.

### Configuration Files
* **ds_config.json**: The DeepSpeed config file.
* **moe.yaml**: Determined config to train the model with Mixture of Experts enabled.
* **zero_stages.yaml**: Same as `moe.yaml`, but trains the model with ZeRO stage 2 optimizer.

## Data
The CIFAR-10 dataset is downloaded from https://www.cs.toronto.edu/~kriz/cifar.html.

## To Run
If you have not yet installed Determined, installation instructions can be found
under `docs/install-admin.html` or at https://docs.determined.ai/latest/index.html

Run the following command: 
```
det experiment create moe.yaml .
``` 
The other configuration can be run by specifying the appropriate configuration file in place 
of `moe.yaml`.

## Results
Training the model with the hyperparameter settings in `moe.yaml` should yield
a validation accuracy of ~45% after 2 epochs.
