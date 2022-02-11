
## Setup

We need to install the Determined CLI and then point the CLI to the master IP address.

```
pip install determined
export DET_MASTER=[CLUSTER ADDRESS]
```

Clone [our fork of gpt-neox](https://github.com/determined-ai/gpt-neox) and checkout the `determined` branch.
```
git clone -b determined https://github.com/determined-ai/gpt-neox
```

## Running Experiments
Go to the `gpt-neox/determined` directory. From here, you can submit experiments to the cluster by running
```
det experiment create [experiment config yaml file] .
```

To create custom HP trials, you can run
```
python custom_search/create_experiments.py
```
after modifying `create_experiments.py` to include the desired HP settings.

### Interpreting HP Search
- Invalid HP trials will show up as "COMPLETED" but have 0 batches trained.
- Errored trials will include OOM settings as well as trials that failed for other reasons.


### Customizing start-up behavior
You can use the `startup-hook.sh` script to setup the container prior to the start of training.
Right now it just makes sure the test dataset is available and directories are setup correctly to match `configs/determined_cluster.yml`.


### Customizing Docker
Modify the Dockerfile in this directory.  Then run from one level up, run 
```
./determined/build_docker.sh [TAG]
```

Be sure to change the experiment config to point to new image name:
```
environment:
  image:
    gpu: [FILL_IN]
```

