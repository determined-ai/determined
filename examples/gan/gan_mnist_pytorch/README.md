# PyTorch MNIST GAN Example

This folder contains the required files and the example code to demonstrate how to train a simple GAN on the MNIST dataset.
The file version can be found on this [PyTorch Lightning GAN example](https://github.com/PyTorchLightning/pytorch-lightning/blob/master/pl_examples/domain_templates/generative_adversarial_net.py)

### Files
* **model_def.py**: The core code for the model. This includes building and compiling the model.
* **data.py**: The data loading and preparation code for the model.
* **const.yaml**: A configuration file that trains the model with constant hyperparameter values. This is also where you can set the flags used in the original script. 
* **distributed.yaml**: Same as const.yaml, but instead uses multiple GPUs. 

### To Run
Installation instructions can be found under `docs/install-admin.html` or at [Determined installation page](https://docs.determined.ai/latest/index.html).
After configuring the settings in const.yaml, run the following command: `det -m <master host:port> experiment create -f const.yaml . `
