# Contributing to Determined

## Reporting Issues and Feature Requests

If you encounter an issue or would like to request a new feature, please create
[an issue on GitHub](https://github.com/determined-ai/determined/issues). You can
also join the [Slack](https://join.slack.com/t/determined-community/shared_invite/zt-cnj7802v-KcVbaUrIzQOwmkmY7gP0Ew)
to get support and talk with other users and developers in real-time.

## Contributing Changes

We welcome outside contributions. If you'd like to make a contribution, please:

1. Tell us about what you'd like to contribute on
   [our Slack](https://join.slack.com/t/determined-community/shared_invite/zt-cnj7802v-KcVbaUrIzQOwmkmY7gP0Ew)
   or [community mailing list](https://groups.google.com/a/determined.ai/forum/#!forum/community).
   We'd hate for you to duplicate efforts that are already in-flight.

1. Apply the linter with `make fmt` and test locally with `make test` before
   submitting your code. Make sure that your code doesn't accidentally include
   cloud credentials. We recommend [using git-secrets to automatically prevent
   this](#secrets).

1. The first time you submit code, you'll need to
   [sign a CLA](https://determined.ai/cla/).

1. Submit a pull request. Someone from the Determined team will review the
   request and provide feedback. Once we agree that the code is in good shape,
   it will be merged into the master branch.

## Installation from Source

### Setting up

Determined can be developed and run on both Linux and macOS (Linux is strongly
recommended for production deployments). Determined has been tested with Ubuntu
16.04 LTS, Ubuntu 18.04 LTS, Arch Linux, CentOS 7, and macOS. Ubuntu is
recommended; on AWS, a good AMI to use is a recent version of "Deep Learning
Base AMI (Ubuntu)".

Start by cloning the Determined repository:

```sh
git clone --recurse-submodules https://github.com/determined-ai/determined.git
```

#### Prerequisites

- Go (>= 1.20)
- Python (>= 3.7.4, <= 3.9), including:
  - python3-venv
  - python3-wheel
  - python3-dev
- Node (>= 20.1.0, < 21)
- NPM (>= 9.5.1)
- Docker (>= 19.03)
- Helm (>= 3.0.0)
- protobuf-compiler (>= 3.15)
- cURL (>= 7)
- jq (>= 1.6)
- socat (>= 1.7)

If you are installing prerequisites from your Linux distribution's package
repository, ensure that they meet the version requirements above, particularly
Python and Node.

#### Install Prerequisites with Homebrew for Linux and macOS

Because the versions of prerequisites from Linux distribution package
repositories can vary widely, we recommend installing the Determined build
prerequisites with [Homebrew](https://brew.sh/).

The following instructions are also applicable for building Determined on macOS.

Install a compiler and build tools:

- macOS: Install the [Command Line Tools for Xcode](https://developer.apple.com/) from Apple
- Debian and Ubuntu: `sudo apt install build-essential`
- Red Hat, CentOS, and Fedora: `sudo yum install gcc make perl-devel`
- SUSE and openSUSE: `sudo zypper install -t pattern devel_basis`

Install Homebrew:

```sh
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

Add Homebrew to your PATH:

```sh
echo 'eval "$($HOME/.linuxbrew/bin/brew shellenv)"' >> $HOME/.profile
eval "$($HOME/.linuxbrew/bin/brew shellenv)"
```

Install the Determined prerequisites:

```sh
brew install go@1.20 python@3.9 node@16 protobuf docker helm curl jq socat
```

Add Python and Node to your PATH:

```sh
echo 'export PATH="$HOME/.linuxbrew/opt/python@3.9/bin:$HOME/.linuxbrew/opt/node@16/bin:$PATH"' >> $HOME/.profile
source $HOME/.profile
```

On Red Hat and CentOS 8 and Ubuntu 16.04, add the compiled GCC 11 libraries to your LD_LIBRARY_PATH:

```sh
echo 'export LD_LIBRARY_PATH=$HOME/.linuxbrew/Cellar/gcc/$(ls $HOME/.linuxbrew/Cellar/gcc/)/lib/gcc/lib64/' >> $HOME/.profile
source $HOME/.profile
```

### Building Determined

```sh
cd determined
python3 -m venv $HOME/.virtualenvs/determined
. $HOME/.virtualenvs/determined/bin/activate
$HOME/.virtualenvs/determined/bin/python3.7 -m pip install --upgrade pip
export PATH=$PATH:$HOME/go/bin
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

For running these services in development, please use [devcluster](https://github.com/determined-ai/devcluster).
It offers an intuitive UI as well as easy rebuild, restart, and configuration
of master and one or more local agents.

### Accessing Determined

After following either set of instructions above, the WebUI will be available at
`http://localhost:8080`. You can also use our command-line tool, `det`, to
interact with Determined. For example, `det slot list` should print out a line
for each GPU on your machine, if you have any, or a line for your CPU, if not.
For more information, see [the reference
documentation](https://docs.determined.ai/latest/reference/cli.html).

## Training a Sample Model

The `tutorials/mnist_pytorch` directory contains code to train a convnet
on [MNIST](http://yann.lecun.com/exdb/mnist/) using PyTorch. To train a model,
run

```sh
det experiment create <config> tutorials/mnist_pytorch/
```

where `<config>` can be

- `tutorials/mnist_pytorch/const.yaml` to train a single model with fixed hyperparameters
- `tutorials/mnist_pytorch/adaptive.yaml` to train multiple models using an [adaptive hyperparameter search algorithm](https://docs.determined.ai/latest/topic-guides/hp-tuning-det/index.html#adaptive-search)

Determined also supports [several other hyperparameter search
methods](https://docs.determined.ai/latest/topic-guides/hp-tuning-det/index.html#other-supported-methods).

After starting a model, you can check on its progress using the WebUI
or the CLI command `det experiment list`.

## Development

### Linting and typechecking

Run `make check`.

### Unit tests

Run `make test`.

### Integration tests

```bash
# Run a Determined cluster
make -C tools run

# Run integration tests locally.
pytest -m "e2e_cpu" e2e_tests/tests
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

### Secrets

To prevent cloud credentials from accidentally being exposed on GitHub, install
and configure the [git-secrets](https://github.com/awslabs/git-secrets) tool.
This sets up git hooks to prevent pushing code that contains secrets (based on regex).

For Mac, the tool can be installed via `brew install git-secrets`. For other
OSes see installation instructions [here](https://github.com/awslabs/git-secrets#installing-git-secrets).

Then navigate to the repository, set up the git hooks, and define the regexes:

```shell
cd /path/to/my/repository

# Set up the git hooks for this repo
git secrets --install

# Add AWS regexes
git secrets --register-aws
# Add GCP regex
git secrets --add '"private_key":\s"-----BEGIN\sPRIVATE\sKEY-----'
```

## Documentation

Visit our [Documentation Guide](https://github.com/determined-ai/determined/blob/main/docs/README.md) to find out how we generate and maintain our docs.
