# Training PyTorch implementation of the Deep SAD method on Determined

This example demonstrates the [PyTorch implementation of the Deep SAD method](https://github.com/lukasruff/Deep-SAD-PyTorch)
adapted to Determined `PyTorchTrial` API and using MNIST dataset.

## Files
* **model_def.py**: The method involved pretraining an autoencoder, and then training the main model.
    This file contains the code for the model training, adapted from original `AETrainer` and `DeepSADTrainer` code.
* **run.sh**: Entrypoint script which orchestrates training both autoencoder and the main model in succession.
* **startup-hook.sh**: This script is run at the initialization of training containers to install
    one extra dependency, `scikit-learn` package.
* **base/**, **datasets/**, **networks/**: Reused utility files from the original code,
    trimmed down to support only the `MNIST_LeNet` model and dataset.

### Configuration Files
* **const_ae.yaml**: Configuration file for the autoencoder model.
* **const_main.yaml**: Same as `const_ae.yaml`, but for the main model.

### Autoencoder Model Data
* **ae_state_dict.pth**: Main experiment (`DeepSADMainTrial` / `const_main.yaml`) expects autoencoder model weights
    to be in this file. `run.sh` will create this file automatically. When training models by hand,
    you can download the autoencoder experiment checkpoint, and copy `state_dict.pth` here.

## Data
This example uses MNIFST dataset downloaded via `torchvision`.

## To Run
If you have not yet installed Determined, installation instructions can be found
under `docs/install-admin.html` or at https://docs.determined.ai/latest/index.html

You can run the entire example using `./run.sh`. To go more in-depth, you can
run the individual autoencoder model using  `det e create const_ae.yaml . -f`,
then download its checkpoint using `det e download <EXPERIMENT_ID>`,
copy `state_dict.pth` from the downloaded checkpoint to `./ae_state_dict.pth`,
and run the main model via `det e create const_main.yaml . -f`.

## Results
Both models should converge. Main model should have the final `test_auc` metric at ~97%.
