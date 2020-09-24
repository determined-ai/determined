# PyTorch multi-prediction MNIST CNN Example

This folder contains the required files to show how to build a multi-prediction MNIST network using Determined's PyTorch API.

### Files
* **model_def.py**: The core code for the model. This includes building and compiling the model.
* **data.py**: The data loading and preparation code for the model.
* **layers.py**: Defines the convolutional layers that the model uses. 
* **const.yaml**: A configuration file that trains the model with constant hyperparameter values. This is also where you can set the flags used in the original script. 
* **adaptive.yaml**: Uses state-of-the-art ASHA hyperparameter tuning algorithm. 

### To Run
Installation instructions can be found under `docs/install-admin.html` or at [Determined installation page](https://docs.determined.ai/latest/index.html). 
After configuring the settings in const.yaml, run the following command: `det -m <master host:port> experiment create -f const.yaml . `

### Results
Upon completion of the experiment, model should achieve target accuracy of ~97%. 
