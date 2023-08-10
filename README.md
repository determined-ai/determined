<p align="center"><img src="determined-logo.png" alt="Determined AI Logo"></p>

Determined is an all-in-one deep learning platform, compatible with PyTorch and Tensorflow.

It takes care of:

- **Distributed training**, for faster results.
- **Hyperparameter tuning**, for obtaining the best models.
- **Resource management**, for cutting cloud GPU costs.
- **Experiment tracking**, for analysis and reproducibility.


# How Determined Works

The main components of Determined are the Python library, the command line interface (CLI), and the Web UI.

## Python Library
Use the Python library to make your existing PyTorch or Tensorflow code compatible with Determined. 

You can do this by organizing your code into one of the class-based APIs:

```python
from determined.pytorch import PyTorchTrial

class YourExperiment(PyTorchTrial):
  def __init__(self, context):
    ...
```

Or by using just the functions you want, via the Core API:

```python
import determined as det

with det.core.init() as core_context:
    ...
```

## Command Line Interface (CLI)

Use the CLI to start the Determined cluster locally, or on your favorite cloud service: 

```bash
det deploy aws up
```

Then train your models:
```bash
det experiment create gpt.yaml .
```

And use yaml files to configure everything from distributed training to hyperparameter tuning:

```yaml
resources:
  slots_per_trial: 8
  priority: 1
hyperparameters:
  learning_rate: 1.0
  dropout: 0.25
searcher:
  metric: validation_loss
  smaller_is_better: true
```


## Web UI

Use the Web UI to view loss curves, hyperparameter plots, code and configuration snapshots, model registries, cluster utilization, debugging logs, performance profiling reports, and more.

![Web UI](docs/assets/readme_images/webui.png)


# Installation

Install the CLI:
```bash
pip install determined
```

Then use `det deploy` to start the Determined cluster on AWS, GCP, or locally.

See the following guides for all the installation details:

- [Local (on-prem)](https://docs.determined.ai/latest/setup-cluster/deploy-cluster/on-prem/overview.html)
- [AWS](https://docs.determined.ai/latest/setup-cluster/deploy-cluster/aws/overview.html)
- [GCP](https://docs.determined.ai/latest/setup-cluster/deploy-cluster/gcp/overview.html)
- [Kubernetes](https://docs.determined.ai/latest/setup-cluster/deploy-cluster/k8s/overview.html)
- [Slurm/PBS](https://docs.determined.ai/latest/setup-cluster/deploy-cluster/slurm/overview.html)


## Try out Determined Locally

Follow [these instructions](https://docs.determined.ai/latest/how-to/installation/requirements.html#install-docker) to install and set up docker.

Then run the following script:

 ```bash
# Start a Determined cluster locally.
python3.7 -m venv ~/.virtualenvs/test
. ~/.virtualenvs/test/bin/activate
pip install determined
# To start a cluster with GPUs, remove `no-gpu` flag.
det deploy local cluster-up --no-gpu
# Access web UI at localhost:8080. By default, "determined" user accepts a blank password.

# Navigate to a Determined example.
git clone --recurse-submodules https://github.com/determined-ai/determined
cd determined/examples/computer_vision/cifar10_pytorch

# Submit job to train a single model on a single node.
det experiment create const.yaml .
 ```


## Try Now on AWS

[![Try Now](https://s3.amazonaws.com/cloudformation-examples/cloudformation-launch-stack.png)](https://console.aws.amazon.com/cloudformation/home?region=us-west-2#/stacks/create/review?templateURL=https://determined-ai-public.s3-us-west-2.amazonaws.com/simple.yaml)


# Documentation

* [Documentation](https://docs.determined.ai)
* [Quick Start Guide](https://docs.determined.ai/latest/getting-started.html)
* Tutorials:
  * [PyTorch MNIST Tutorial](https://docs.determined.ai/latest/tutorials/pytorch-mnist-tutorial.html)
  * [TensorFlow Keras MNIST Tutorial](https://docs.determined.ai/latest/tutorials/tf-mnist-tutorial.html)


# Community

If you need help, want to file a bug report, or just want to keep up-to-date
with the latest news about Determined, please join the Determined community!

* [Slack](https://determined-community.slack.com) is the best place to
  ask questions about Determined and get support. [Click here to join our Slack](
  https://join.slack.com/t/determined-community/shared_invite/zt-cnj7802v-KcVbaUrIzQOwmkmY7gP0Ew).
* You can also join the [community mailing list](https://groups.google.com/a/determined.ai/forum/#!forum/community)
  to ask questions about the project and receive announcements.
* To report a bug, [file an issue](https://github.com/determined-ai/determined/issues) on GitHub.
* To report a security issue, email [`security@determined.ai`](mailto:security@determined.ai).

# Contributing

[Contributor's Guide](CONTRIBUTING.md)

# License

[Apache V2](LICENSE)
