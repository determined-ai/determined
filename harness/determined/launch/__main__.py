import determined as det
import subprocess


def launch():
    info = det.get_cluster_info()
    assert info is not None, "must be run on-cluster"
    assert info.task_type == "TRIAL", f'must be run with task_type="TRIAL", not "{info.task_type}"'

    # Hack: read the full config.  The experiment config is not a stable API!
    experiment_config = info.trial._config

    launch_cmd = experiment_config.get("launch", "python3 -m determined.launch.autohorovod")
    return subprocess.Popen(launch_cmd.split(" ")).wait()


if __name__ == "__main__":
    launch()