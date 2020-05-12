# Contributing to Determined

## Reporting Issues

Please report any issues at https://github.com/determined-ai/determined/issues.

## Contributing Changes

TODO

## Installation from Source

### Setting up

Determined can be developed and run on both Linux and macOS (Linux is strongly
recommended for production deployments). Determined has been tested with Ubuntu
16.04 LTS, Ubuntu 18.04 LTS, Arch Linux, CentOS 7, and macOS. Ubuntu is
recommended; on AWS, a good AMI to use is a recent version of "Deep Learning
Base AMI (Ubuntu)".

Start by cloning the Determined repo:

```sh
git clone git@github.com:determined-ai/determined.git
```

To install OS-level dependencies, run the appropriate one of the scripts below
from within your clone of the repository.

```sh
scripts/setup-env-ubuntu.sh
scripts/setup-env-centos.sh
scripts/setup-env-arch.sh
scripts/setup-env-macos.sh
```

### Building Determined

```sh
python3.6 -m venv ~/.virtualenvs/determined
. ~/.virtualenvs/determined/bin/activate
make all
```

In the future, ensure that you activate the virtualenv (by running the
`activate` command above) whenever you want to interact with Determined. Tools
such as [virtualenvwrapper](https://virtualenvwrapper.readthedocs.io/en/latest/)
or [direnv](https://direnv.net/) may help streamline the process.

## Running Determined

A minimal Determined cluster consists of three services: a
[PostgreSQL](https://www.postgresql.org/) database, a Determined master,
and a Determined agent.

To start the master and agent, along with a transient database, do:

```sh
make -C tools run
```

The database will be destroyed when the cluster is shutdown. To start a
long-running database (running in the background), do:

```sh
make -C tools start-db
```

### Accessing Determined

After following either set of instructions above, the WebUI will be available at
http://localhost:8080. You can also use our command-line tool, `det`, to
interact with Determined. For example, `det slot list` should print out a line
for each GPU on your machine, if you have any, or a line for your CPU, if not.
For more information, see [the reference
documentation](https://docs.determined.ai/latest/reference/cli.html).

## Training a Sample Model

The `examples/official/mnist_pytorch` directory contains code to train a convnet
on [MNIST](http://yann.lecun.com/exdb/mnist/) using PyTorch. To train a model,
run

```sh
det experiment create <config> examples/official/mnist_pytorch/
```

where `<config>` can be

- `examples/official/mnist_pytorch/const.yaml` to train a single model with fixed hyperparameters
- `examples/official/mnist_pytorch/adaptive.yaml` to train multiple models using
  an [adaptive hyperparameter search
  algorithm](https://docs.determined.ai/latest/topic-guides/hp-tuning-det/index.html#adaptive-search)

Determined also supports [several other hyperparameter search
methods](https://docs.determined.ai/latest/topic-guides/hp-tuning-det/index.html#other-supported-methods).

After starting a model, you can check on its progress using the WebUI
or the CLI command `det experiment list`.

## Development

### Linting and typechecking

Run `make check`.

To add a commit message template and a commit-time hook to help you follow our
commit message guidelines, you can also run `scripts/configure-repo.sh` (which
does not need to be done repeatedly).

### Unit tests

Run `make test`.

### Integration tests

#### Prerequisites

For cloud integration tests, AWS and GCP credentials must be configured.

- [AWS Credentials](https://boto3.amazonaws.com/v1/documentation/api/latest/guide/configuration.html)
- [GCP Credentials](https://cloud.google.com/docs/authentication/getting-started)

#### Run the tests

```bash
# Run local integration tests except for cloud-related tests.
make test-integrations
```

#### Customize configuration

By default, the master process is exposed on port 8081 of the host
machine. To change the master port, run

```sh
make test-integrations INTEGRATIONS_HOST_PORT=<PORT>
```

## Debugging

### Connecting to PostgreSQL

To connect directly to the Determined metadata database, run this command from
the Determined master host:

```sh
docker run -it --rm \
  --network determined \
  -e PGPASSWORD=my-postgres-password \
  postgres:10 psql -h determined-db -U postgres -d determined
```

### Get profiling information

```sh
go tool pprof http://master-ip:port  # for CPU samples
go tool pprof http://master-ip:port/debug/pprof/heap  # for heap samples
go tool pprof -http :8081 ~/pprof/sample-file
```

### GPU support

To use Determined with GPUs, the Nvidia drivers (>= 384.81) and
[`nvidia-container-toolkit`](https://docs.determined.ai/latest/how-to/installation/requirements.html#installing-docker)
must be installed.

To verify that your system can run containers that use GPUs and CUDA, run:

```sh
docker run --gpus all --rm nvidia/cuda:10.0-cudnn7-runtime-ubuntu16.04 nvidia-smi
```

If this command displays one or more GPUs, the Determined agent should
automatically detect the system's GPUs and make them available for
running experiments.
