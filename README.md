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
  - intelligent hyperparameter optimization based on cutting-edge research
  - flexible GPU scheduling, including dynamically resizing training jobs
    on-the-fly and automatic management of cloud resources on AWS and GCP
  - built-in experiment tracking, metrics storage, and visualization
  - automatic fault tolerance for DL training jobs
  - integrated support for TensorBoard and GPU-powered Jupyter notebooks

To use Determined, you can continue using popular DL frameworks such as
TensorFlow and PyTorch; you just need to modify your model code to implement
the Determined API.

## Installation

* [Installation on AWS](https://docs.determined.ai/latest/how-to/install-aws.html)
* [Installation on GCP](https://docs.determined.ai/latest/how-to/install-gcp.html)
* [Manual Installation](https://docs.determined.ai/latest/how-to/install-general.html)

## Next Steps

Determined supports models written using TensorFlow or PyTorch. To get started
using Determined, follow the tutorial for your preferred deep learning framework:

* [TensorFlow MNIST Tutorial](https://docs.determined.ai/latest/tutorials/tf-mnist-tutorial.html)
* [PyTorch MNIST Tutorial](https://docs.determined.ai/latest/tutorials/pytorch-mnist-tutorial.html)

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
