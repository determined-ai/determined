# TensorFlow (tf.keras) UNet Example

This folder contains the required files and the example code to use Tensorflow's Image Segmentation via UNet tutorial with Determined.
The file version can be found on [Tensorflow Image Segmentation with UNet](https://www.tensorflow.org/tutorials/images/segmentation)

### Files
* **model_def.py**: The core code for the model. This includes building and compiling the model.
* **const.yaml**: A configuration file that trains the model with constant hyperparameter values. This is also where you can set the flags used in the original script.
* **distributed.yaml**: Same as const.yaml, but instead uses multiple GPUs. 
* **startup-hook.sh**: Extra dependencies that Determined is required to install. This includes downloading the training data.

### Data
The data used for this script was fetched via Tensorflow Datasets as done by the tutorial itself. The original Oxford-IIIT Pet dataset is linked [here](https://www.robots.ox.ac.uk/~vgg/data/pets/). 

### To Run
Installation instructions can be found under `docs/install-admin.html` or at [Determined installation page](https://docs.determined.ai/latest/index.html). 
After configuring the settings in const.yaml, run the following command: `det -m <master host:port> experiment create -f const.yaml . `

### Expected Results
![Single GPU vs. Distributed Training with Determined AI](Cumulative_Batches.png)
![Single GPU vs. Distributed Training Validation Accuracy](Validation_Accuracy.png)
