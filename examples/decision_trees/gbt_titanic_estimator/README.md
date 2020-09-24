# TensorFlow (Estimator API) Boosted Trees Example

This folder contains the required files and the example code to use Tensorflow's Boosted Trees Estimator example with Determined.
The file version can be found on [Tensorflow's Estimator examples page](https://www.tensorflow.org/tutorials/estimator/boosted_trees)

### Files
* **model_def.py**: The core code for the model. This includes building and compiling the model.
* **startup-hook.sh**: Extra dependencies that Determined is required to install.
* **const.yaml**: A configuration file that trains the model with constant hyperparameter values. This is also where you can set the flags used in the original script. 
* **adaptive.yaml**: Uses state-of-the-art ASHA hyperparameter tuning algorithm. 

### To Run
Installation instructions can be found under `docs/install-admin.html` or at [Determined installation page](https://docs.determined.ai/latest/index.html).
Note that due to the nature of the model, this example is meant to run as a single-GPU model or a hyperparameter search; it does NOT support distributed training.
After configuring the settings in const.yaml, run the following command: `det -m <master host:port> experiment create -f const.yaml . `

### Results
Upon completion of the experiment, model should achieve target accuracy of ~83%. 
