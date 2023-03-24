# Running a Ray Cluster Within a Determined Task

## Launching a Cluster

To launch a single worker cluster, and expose Ray dashboard port 8265 on `localhost:8265`:

    det e create ray_launcher.yaml . -f -p 8265 --config resources.slots_per_trial=1

Note: this experiment will run forever, it must be explicitly terminated either in WebUI or using CLI ``det e kill EXP_ID`` when you no longer need it.

Running a test job:

    pip install -U "ray[air]"
    export RAY_ADDRESS="http://localhost:8265"; ray job submit --working-dir . -- python ray_job.py


## Advanced Usage

### Multi-Worker Cluster

To launch a multi-worker cluster with 4 total workers:

    det e create ray_launcher.yaml . -f -p 8265 --config resources.slots_per_trial=4

### Using Different Local Port for Ray Dashboard Proxy

By default, this example binds local port 8265 and proxies it to ray dashboard running within a Determined task. To use a different port e.g. 8266:

    det e create ray_launcher.yaml . -f -p 8266:8265 --config resources.slots_per_trial=1
    export RAY_ADDRESS="http://localhost:8266"; ray job submit --working-dir . -- python ray_job.py
