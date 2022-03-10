import re
import sys
import time
import subprocess
import shutil

from collections import deque


class IdleGpuWatcher:
    def __init__(
        self,
        min_threshhold_percentage: int = 1,
        sample_freq: int = 10,
        num_samples: int = 30,
        delay_samples: int = 30,
        debug: bool = False
    ):
        if not shutil.which("nvidia-smi"):
            raise RuntimeError("Unable to locate 'nvidia-smi'")
        if not isinstance(min_threshhold_percentage, int) or min_threshhold_percentage < 1:
            raise ValueError("'min_threshhold_percentage' must be an int >= 1")
        if not isinstance(num_samples, int) or num_samples < 1:
            raise ValueError("'num_samples' must be an int >= 1")
        if not isinstance(sample_freq, int) or sample_freq < 1:
            raise ValueError("'sample_freq' must be an int >= 1")
        if not isinstance(delay_samples, int) or delay_samples < 1:
            raise ValueError("'sample_freq' must be an int >= 1")

        self._num_samples = num_samples
        self._sample_freq = sample_freq
        self._full_window = sample_freq * num_samples
        self._min_threshhold = min_threshhold_percentage
        self._delay_samples = delay_samples
        self._debug = debug

        header_regstr = {
            "uuid": r"(?P<uuid>[a-zA-Z0-9-]+)",
            "utilization.gpu": r"(?P<gpu_util>[0-9]+) %"
        }
        line_regstr = ", ".join(header_regstr.values())
        query_gpu = ",".join(header_regstr.keys())

        self._nvida_smi_cmd = ["nvidia-smi", "--format=csv,noheader", f"--query-gpu={query_gpu}"]
        self._line_regex = re.compile(line_regstr)
        self._gpu_util_samples = {}  # gpu_uuid: <circular queue of samples>

    def _get_nvidia_smi(self):
        # nvidia-smi --help-query-gpu: "utilization.gpu":
        # Percent of time over the past sample period during which one or more kernels was
        # executing on the GPU. The sample period may be between 1 second and 1/6 second
        # depending on the product.
        output = {}

        csv_output = subprocess.check_output(self._nvida_smi_cmd).decode()
        for line in csv_output.splitlines():
            match = re.match(self._line_regex, line)
            if match is None:
                raise RuntimeError(f"Unexpected output format: {line}")

            data = match.groupdict()
            output[data["uuid"]] = int(data["gpu_util"])
        return output

    def _check_idle(self):
        gpu_utils = self._get_nvidia_smi()

        for gpu_uuid, gpu_util in gpu_utils.items():
            if self._gpu_util_samples.get(gpu_uuid) is None:
                self._gpu_util_samples[gpu_uuid] = deque()

            self._gpu_util_samples[gpu_uuid].append(gpu_util)

            if len(self._gpu_util_samples[gpu_uuid]) < self._num_samples:
                continue

            if all(s < self._min_threshhold for s in self._gpu_util_samples[gpu_uuid]):
                sys.stderr.write(
                    f"IDLE_GPU_WATCHER: {gpu_uuid} usage below threshhold {self._min_threshhold}% "
                    f"for last {self._sample_freq * self._num_samples} seconds. Killing trial.\n")
                sys.exit(1)
            if self._debug:
                print(f"DEBUG: {gpu_uuid} usage {gpu_util}% ")

            self._gpu_util_samples[gpu_uuid].popleft()

    def watch_process(self, args):
        p = subprocess.Popen(args)
        print("IDLE_GPU_WATHCER: "
              f"Starting idle GPU watcher: min_threshhold={self._min_threshhold}%, "
              f"sample_freq={self._sample_freq}, num_samples={self._num_samples}, "
              f"delay_samples={self._delay_samples}, window_size={self._full_window}")
        try:
            while True:
                if p.poll() is None:
                    time.sleep(self._sample_freq)

                    if self._delay_samples > 0:
                        self._delay_samples -= 1
                        continue

                    self._check_idle()
                    continue

                break

        finally:
            if p.poll() is None:
                print("IDLE_GPU_WATCHER: Training process still alive, killing.")
                p.kill()  # SIGKILL because it is likely hung

        return p.wait()


def usage():
    print("""Purpose:
  Launches a process and watches GPU usage.
  Exits if below min_percentage for num_samples after delay_samples.

Usage:
  ./idle_gpu_watcher.py <min_percent> <sample_freq> <num_samples> <delay_samples> <debug> <exec> <arg1> <arg2> ...
""")


if __name__ == "__main__":
    """
    This is for determined 0.17.2, there are easier ways to do this in later version.

    The intention here is to inject this into a startup-hook.sh script like so:

```
# Other startup-hook.sh script commands

#
# Note: this must appear at the end of the file.
#
"$DET_PYTHON_EXECUTABLE" -m determined.exec.prep_container --rendezvous

MIN_PERCENTAGE=1   # Minimum GPU usage which triggers a failure.
SAMPLE_FREQ=5      # How frequently to sample the GPU usage.
NUM_SAMPLES=60     # Number of samples to look at when evaluating GPU usage.
DELAY_SAMPLES=12   # Number of samples to wait before evaluating GPU usage.
IDLE_DEBUG=true    # Print debug statements

IDLE_ARGS="$MIN_PERCENTAGE $SAMPLE_FREQ $NUM_SAMPLES $DELAY_SAMPLES $IDLE_DEBUG"

EXEC_ARGS="$DET_PYTHON_EXECUTABLE -m determined.exec.launch_autohorovod $@"
exec "$DET_PYTHON_EXECUTABLE" -m idle_gpu_watcher $IDLE_ARGS $EXEC_ARGS
```
    """
    if len(sys.argv) < 7:
        usage()
        sys.stderr.write("ERR: We must have at least 6 arguments\n")
        sys.stderr.flush()
        sys.exit(1)

    if any(not x.isdigit() for x in sys.argv[1:5]) or any(int(x) < 1 for x in sys.argv[1:5]):
        usage()
        sys.stderr.write("ERR: Args 1 through 4 must be non-zero positive integers\n")
        sys.stderr.flush()
        sys.exit(1)

    min_threshhold_percentage = int(sys.argv[1])
    sample_freq = int(sys.argv[2])
    num_samples = int(sys.argv[3])
    delay_samples = int(sys.argv[4])
    idle_debug = True if sys.argv[5].lower() == "true" else False
    process_args = sys.argv[6:]

    watcher = IdleGpuWatcher(
        min_threshhold_percentage=min_threshhold_percentage,
        sample_freq=sample_freq,
        num_samples=num_samples,
        delay_samples=delay_samples,
        debug=idle_debug
    )

    sys.exit(watcher.watch_process(process_args))
