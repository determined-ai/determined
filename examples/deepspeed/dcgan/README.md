# DeepSpeed CIFAR Example
This example is adapted from the
[DCGAN example in the DeepSpeedExamples](https://github.com/microsoft/DeepSpeedExamples/tree/master/training/gan)
repository. It is intended to demonstrate a simple usecase of DeepSpeed with Determined.

## Files
* **model.py**: The DCGANTrial definition.
* **gan_model.py**: Network definitions for generator and discriminator.
* **data.py**: Dataset loading/downloading code.

### Configuration Files
* **ds_config.json**: The DeepSpeed config file.
* **mnist.yaml**: Determined config to train the model on mnist on a cluster.

## Data
This repo supports the same datasets as the original example: `["imagenet", "lfw", "lsun", "cifar10", "mnist", "fake", "celeba"]`.  The `cifar10` and `mnist` datasets will be downloaded as needed, whereas the rest must be mounted on the agent.  For `lsun`, the `data_config.classes` setting must be set.  The `folder` dataset can be used to load an arbitrary torchvision `ImageFolder` that is mounted on the agent.

## To Run Locally

It is recommended to run this from within one of our agent docker images, found at
https://hub.docker.com/r/determinedai/pytorch-ngc/tags

After installing docker and pulling an image, users can launch a container via
`docker run --gpus=all -v ~path/to/repo:/src/proj -it <container name>`

Install necessary dependencies via `pip install determined mpi4py`

Then, run the following command:
```
python trainer.py
```

Any additional configs can be specified in `mnist.yaml` and `ds_config.json` accordingly.

## To Run on Cluster
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
