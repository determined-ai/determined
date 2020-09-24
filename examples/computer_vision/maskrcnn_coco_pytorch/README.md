# PyTorch Mask R-CNN Example

This folder contains the example code to run an object detection model with Determined's PyTorch API.
The file version can be found on this [PyTorch Mask R-CNN tutorial](https://pytorch.org/tutorials/intermediate/torchvision_tutorial.html)

### Files
* **model_def.py**: The core code for the model. This includes building and compiling the model.
* **const.yaml**: A configuration file that trains the model with constant hyperparameter values. This is also where you can set the flags used in the original script.
* **distributed.yaml**: Same as const.yaml, but instead uses multiple GPUs.
* **adaptive.yaml**: Uses state-of-the-art ASHA hyperparameter tuning algorithm.

### Data:
The current implementation uses the pedestrian detection and segmentation [Penn-Fudan Database](https://www.cis.upenn.edu/~jshi/ped_html/).

### To Run:
Installation instructions can be found under `docs/install-admin.html` or at [Determined installation page](https://docs.determined.ai/latest/index.html). 
After configuring the settings in  const.yaml, run the following command: `det -m <master host:port> experiment create -f const.yaml . `

### Results:
Upon completion of the experiment, model should achieve a target IOU of ~0.42.
