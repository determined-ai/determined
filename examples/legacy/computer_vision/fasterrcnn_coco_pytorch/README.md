# PyTorch Faster R-CNN Example

This example shows how to build an object detection model on the Penn-Fudan 
Database using Determined's PyTorch API. This example is adapted from this [PyTorch 
Mask R-CNN tutorial](https://pytorch.org/tutorials/intermediate/torchvision_tutorial.html)

## Files
* **model_def.py**: The core code for the model. This includes building and compiling the model.

### Configuration Files
* **const.yaml**: Train the model with constant hyperparameter values.
* **adaptive.yaml**: Perform a hyperparameter search using Determined's state-of-the-art adaptive hyperparameter tuning algorithm.

## Data
The current implementation uses the pedestrian detection and segmentation 
[Penn-Fudan Database](https://www.cis.upenn.edu/~jshi/ped_html/).

## To Run
If you have not yet installed Determined, installation instructions can be found
under `docs/install-admin.html` or at https://docs.determined.ai/latest/index.html

Run the following command: `det -m <master host:port> experiment create -f 
const.yaml .`. The other configurations can be run by specifying the appropriate 
configuration file in place of `const.yaml`.

## Results
Training the model with the hyperparameter settings in `const.yaml` should yield
an IOU of ~0.42.
