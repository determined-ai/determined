# DeepSpeed CIFAR Example
This example is adapted from the
[DCGAN example in the DeepSpeedExamples](https://github.com/microsoft/DeepSpeedExamples/tree/master/gan)
repository. It is intended to demonstrate a simple usecase of DeepSpeed with Determined.

## Files
* **model_def.py**: The DCGANTrial definition.
* **gan_model.py**: Network definitions for generator and discriminator.
* **data.py**: Dataset loading/downloading code.

### Configuration Files
* **ds_config.json**: The DeepSpeed config file.
* **mnist.yaml**: Determined config to train the model on mnist.
* **mnist_grad_accum.yaml**: Same as `mnist.yaml`, but with gradient accumulation turned on.
* **cifar10_zero2.yaml**: Determined config to train the model on `cifar10` with ZeRO stage 2 and float16 mxied precision enabled.

## Data
This repo supports the same datasets as the original example: `["imagenet", "lfw", "lsun", "cifar10", "mnist", "fake", "celeba"]`.  The `cifar10` and `mnist` datasets will be downloaded as needed, whereas the rest must be mounted on the agent.  For `lsun`, the `data_config.classes` setting must be set.  The `folder` dataset can be used to load an arbitrary torchvision `ImageFolder` that is mounted on the agent.

## To Run
If you have not yet installed Determined, installation instructions can be found
under `docs/install-admin.html` or at https://docs.determined.ai/latest/index.html

Run the following command:
```
det experiment create mnist.yaml .
```
The other configurations can be run by specifying the appropriate configuration file in place
of `mnist.yaml`.

## Results
Training `mnist` should yield reasonable looking fake digit images on the images tab in TensorBoard after ~5k steps.

Training `cifar10` does not converge as convincingly, but should look image-like after ~10k steps.
