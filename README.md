<p align="center"><img src="determined-logo.png" alt="Determined AI Logo"></p>

# Determined: Deep Learning Training Platform

Determined helps deep learning teams **train models more quickly**, **easily
share GPU resources**, and **effectively collaborate**. Determined allows deep
learning engineers to focus on building and training models at scale, without
needing to worry about DevOps or writing custom code for common tasks like
fault tolerance or experiment tracking.

You can think of Determined as a platform that bridges the gap between tools
like TensorFlow and PyTorch --- which work great for a single researcher with a
single GPU --- to the challenges that arise when doing deep learning at scale,
as teams, clusters, and data sets all increase in size.

## Key Features

  - high-performance distributed training without any additional changes to
    your model code
  - intelligent hyperparameter tuning based on cutting-edge research
  - flexible GPU scheduling, including dynamically resizing training jobs
    on-the-fly, automatic management of cloud resources on AWS and GCP and
    optional support for Kubernetes
  - built-in experiment tracking, metrics visualization, and model registry
  - automatic fault tolerance for DL training jobs
  - integrated support for TensorBoard and GPU-powered Jupyter notebooks

To use Determined, you can continue using popular DL frameworks such as
TensorFlow and PyTorch; you just need to modify your model code to implement
the Determined API.

## Installation

* [Installation Guide](https://docs.determined.ai/latest/how-to/install-main.html)

### Try Now on AWS

[![Try Now](https://s3.amazonaws.com/cloudformation-examples/cloudformation-launch-stack.png)](https://console.aws.amazon.com/cloudformation/home?region=us-west-2#/stacks/create/review?templateURL=https://determined-ai-public.s3-us-west-2.amazonaws.com/simple.yaml)

## Next Steps

For a brief introduction to using Determined, start with the
[Quick Start Guide](https://docs.determined.ai/latest/tutorials/quick-start.html).

To port an existing deep learning model to Determined, follow the
tutorial for your preferred deep learning framework:

* [PyTorch MNIST Tutorial](https://docs.determined.ai/latest/tutorials/pytorch-mnist-tutorial.html)
* [TensorFlow Keras MNIST Tutorial](https://docs.determined.ai/latest/tutorials/tf-mnist-tutorial.html)

## Documentation

The documentation for the latest version of Determined can always be found
[here](https://docs.determined.ai).

## Community

If you need help, want to file a bug report, or just want to keep up-to-date
with the latest news about Determined, please join the Determined community!

* [Slack](https://determined-community.slack.com) is the best place to
  ask questions about Determined and get support. [Click here to join our Slack](
  https://join.slack.com/t/determined-community/shared_invite/zt-cnj7802v-KcVbaUrIzQOwmkmY7gP0Ew).
* You can also join the [community mailing list](https://groups.google.com/a/determined.ai/forum/#!forum/community)
  to ask questions about the project and receive announcements.
* To report a bug, [file an issue](https://github.com/determined-ai/determined/issues) on GitHub.
* To report a security issue, email [`security@determined.ai`](mailto:security@determined.ai).

## Contributing

[Contributor's Guide](CONTRIBUTING.md)

## License

[Apache V2](LICENSE)
