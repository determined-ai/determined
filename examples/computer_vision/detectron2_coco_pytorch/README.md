# Detectron2 Example

This example is an Detectron2 port using Determined's PyTorchTrial API. The original example can be found on
 [Facebook Research's Detectron2 Github](https://github.com/facebookresearch/detectron2/blob/v0.1.2/tools/plain_train_net.py). This example is based on version 1.2 to compare training performance. More information on the original benchmarks can be found at the benchmark [documentation] (https://detectron2.readthedocs.io/en/latest/notes/benchmarks.html) page.

## Files
* **model_def.py**: The core code for the model. This includes building and compiling the model.
* **startup-hook.sh**: This script will automatically be run by Determined during startup of every container launched for this experiment. This script exports the approprate dataset location .
* **detectron2_files/**: This folder includes original Detectron2 files that have been slightly altered to work with Determined.


### Configuration Files
* **const.yaml**: Train the model with constant hyperparameter values.
* **distributed.yaml**: Same as `const.yaml`, but trains the model with multiple GPUs (distributed training).


## Data
The coco dataset is the primary dataset for this port. Information on how to download the dataset can be found [here](https://cocodataset.org/#home). Once downloaded, copy the data on all the Determined agents. Then configure the bind mount in the configuration files to the appropriate location. More information on bind mounts can be found [here](https://docs.determined.ai/latest/tutorials/data-access.html#distributed-file-system).

## To Run
If you have not yet installed Determined, installation instructions can be found
under `docs/install-admin.html` or at https://docs.determined.ai/latest/index.html

Run the following command: `det -m <master host:port> experiment create -f 
const.yaml .`. The other configurations can be run by specifying the appropriate 
configuration file in place of `const.yaml`.

## Results
The default configurations are based on [the original Detectron benchmarks](https://github.com/facebookresearch/detectron2/blob/v0.1.2/configs/Detectron1-Comparisons/faster_rcnn_R_50_FPN_noaug_1x.yaml). At the end of training, the boxAP and and segmAP should be over 34.00.