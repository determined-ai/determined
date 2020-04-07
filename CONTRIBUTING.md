# Contributing to Determined

## Reporting Issues

## Contributing Changes

## Installation from Source

These instructions describe how to build Determined from source.
Determined works on both Linux and Mac OS X (Linux is strongly
recommended for production deployments). Start by cloning the Determined
repo:


```sh
git clone git@github.com:determined-ai/determined.git
```

### Linux Prerequisites

Determined has been tested with Ubuntu 16.04 LTS, Ubuntu 18.04 LTS, and CentOS
7. Ubuntu is recommended; on AWS, a good AMI to use is a recent version
of "Deep Learning Base AMI (Ubuntu)".

To install OS-level dependencies on Ubuntu 16.04 LTS,  CentOS 7, or Arch
Linux respectively:

```sh
./scripts/setup-env-ubuntu.sh

./scripts/setup-env-centos.sh

./scripts/setup-env-arch.sh
```

Then, logout and start a new terminal session.

### Mac OS X Prerequisites

1. Download and install [Docker for Mac](https://www.docker.com/docker-mac).

2. Install Homebrew, if you haven't done so yet.

3. Install the other prerequites via Homebrew:

```sh
brew install \
  go \
  libomp \
  node \
  python3 \
  yarn
```

### Building Determined

```sh
mkvirtualenv -a $PWD/determined --python=`which python3.6` --no-site-packages det
make get-deps all
```

In the future, ensure you activate the virtualenv (`workon det`)
whenever you want to interact with Determined.

### Starting the Master

Note: If you are just interested in setting up a local development cluster,
skip this section and [Starting the Agent](#starting-the-agent) and instead
go straight to [Local Deployment](#local-deployment).

After the common steps above, next do:

1. Ensure that TCP port 8080 is Internet-accessible (e.g., on EC2, this
   requires adding tcp/8080 to the instance's security group).

2. `docker network create determined`

3. `./dist/bin/determined-db-start`

4. `./dist/bin/determined-master-start`

5. Visit 'host:8080', where `host` is the public DNS name of the host.

The master stores experiment metadata in a Postgres database; the
database is stored in a persistent Docker volume. To reset the database
and discard all of its content, use `dist/bin/determined-db-reset`.

### Starting the Agent

After the common steps above, start an agent by doing:

```sh
./dist/bin/determined-agent-start $ADDR
```

where `$ADDR` is the IP address of the master node (on EC2, use the
master's private DNS name). Note that using "localhost" or "127.0.0.1"
seem to run into a bug in the WebSockets library we're using (typical
symptom is "Cannot assign requested address"), so use a routeable IP
address or host name instead.

## Local Deployment

```sh
# Setup local docker compose cluster
python deploy/local/cluster_cmd.py fixture-up

# Watch stdout/stderr
python deploy/local/cluster_cmd.py logs

# Edit code
...
# Update docker images.
#
# Can be replaced with specific make targets based on component being updated
make

# Repeat cluster setup, etc.

# Teardown docker compose cluster
python deploy/local/cluster_cmd.py fixture-down
```

### Web Development
If you are working on UI development, you can instruct the master container to
bind mount the `./webui` directory into itself by running `python
deploy/local/cluster_cmd.py fixture-up --webui-root /PATH/TO/DET_DIR/master/build/webui`
or setting the `DET_WEBUI_ROOT` environment variable to an absolute path to the
directory with the WebUI static build files for `docs`, `elm` and `react`.

## Running the MNIST example

The `dist/examples/mnist_tf` directory contains code to train a convnet on
MNIST (from http://yann.lecun.com/exdb/mnist/) using TensorFlow.  They
can be executed via

`det experiment create <config> dist/examples/mnist_tf/`

`<config>` can be one of

1. `dist/examples/mnist_tf/const.yaml`: single trial, fixed hyperparameter settings
2. `dist/examples/mnist_tf/random.yaml`: single trial, random search
3. `dist/examples/mnist_tf/adaptive.yaml`: multiple trials, adaptive

Use the WebUI to watch progress.

## Development

### Linting and Typechecking

Run `make check`.

To add a commit message template and a commit-time hook to help you follow our
commit message guidelines, you can also run `scripts/configure-repo.sh` (which
does not need to be done repeatedly).

### Unit Tests

Run `make test`.

### Integration Tests

**Prerequisites**

- The Determined CLI is installed in your environment. If you
have not yet installed it, you can do so with `pip install -e .`
- For cloud integration tests:
    - AWS and GCP credentials are configured to run cloud-related integration tests.
        - [AWS Credentials](https://boto3.amazonaws.com/v1/documentation/api/latest/guide/configuration.html)
        - [GCP Credentials](https://cloud.google.com/docs/authentication/getting-started)

**Run integration tests**

```bash
# Run local integration tests except for cloud-related tests
make test-integrations

# Run cloud integration tests
make test-cloud-integrations
```

**Customize configuration**

By default, the master process is exposed on port 8081 of the host
machine. To change the master port:

```sh
make test-integrations INTEGRATIONS_HOST_PORT=<PORT>
```

If you want to run the integration tests with GPU support enabled,
change the default Docker container runtime to be `nvidia`:

https://github.com/NVIDIA/nvidia-docker/wiki/Advanced-topics#default-runtime

## Dependency management

This project uses [pip-compile](https://github.com/jazzband/pip-tools) for
pinning dependencies versions.

To add a dependency, edit `setup.py` or an appropriate `.in` file and then run
`make pin-deps`. The `pip-compile` tool will then generate the appropriate
pinned dependencies in `requirements.txt` files throughout the repo.

To update all dependencies, run `make upgrade-deps`.

## Cutting Releases

See [Releases](RELEASE.md) for cutting new releases.

## Debugging

### Connecting to Postgres

To connect directly to the Determined metadata database, run this command from
the Determined master host:

```sh
docker run -it --rm \
    --network determined \
    -e PGPASSWORD=my-postgres-password \
    postgres:10.8 psql -h determined-db -U postgres determined
```

### Get profiling information

```sh
go tool pprof http://master-ip:port  # for CPU samples
go tool pprof http://master-ip:port/debug/pprof/heap  # for heap samples
go tool pprof -http :8081 ~/pprof/sample-file
```

There is also a corresponding command for local deployments:

```sh
python deploy/local/cluster_cmd.py pprof-cpu
python deploy/local/cluster_cmd.py pprof-heap
```

### GPU Support

To use Determined with GPUs, the Nvidia CUDA drivers (>= 384.81) must be
installed. The instructions above install the nvidia-docker2 package; to
verify that your system can run containers that use GPUs, try:

```sh
docker run --runtime=nvidia --rm nvidia/cuda:10.0-cudnn7-runtime-ubuntu16.04 nvidia-smi
```

If this command displays one or more GPUs, the Determined agent should
automatically detect the system's GPUs and make them available for
running experiments.
