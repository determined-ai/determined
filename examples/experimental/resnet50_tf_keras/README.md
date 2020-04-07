## ResNet50 with Determined

This folder contains the example code to use Tensorflow's ResNet ImageNet training scripts with Determined. This directory offers two examples for training this model:

(1) A notebook implementation that can be used to launch an experiment via the Determined Native API.
(2) A standard model definition that can be used to launch an experiment via the Determined command line interface.

### Notebook Implementation

To run this notebook on a Determined cluster, use the following command:

```
det notebook start --context .
```

### Standard Model Definition

For this implementation, we removed the GPU, Workers and other distribution utils functionality since Determined will be managing this information. The core functionality of the original script remains unchanged.

#### Folders and Files:
   **model_def.py**: Contains the core code for the model. This includes building and compiling the model.
   **data.py**: Is where the data loaded and prepared for the model.
   **const.yaml**: Contains the configuration for the experiment. This is also where you can set the flags used in the original script.
   **const_multiGPU.yaml**: Same as const.yaml, but instead uses multi-GPUs
   **tensorflow_files**: Is a folder containing the required original scripts used in the Tensorflow example. These files have been extracted from  https://github.com/tensorflow/models/blob/master/official/:5a3b762

#### Data:
   The current implementation uses synthetic data that is in the shape of images.

#### To Run:
   *Preliminary*:
     Follow the installation instruction found under `docs/install-admin.html` or at https://docs.determined.ai/latest/index.html to install Determined

   After configuring the settings in const.yaml. Run the following command:
      `det experiment create -f const.yaml .`

   If Determined is not running locally use
     `det -m <master host:port> experiment create -f const.yaml . `


This example is based on: https://github.com/tensorflow/models/blob/master/official/vision/image_classification/resnet_imagenet_main.py

