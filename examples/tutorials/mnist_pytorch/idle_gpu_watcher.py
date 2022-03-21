import argparse
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
        if args is None or len(args) == 0:
            raise ValueError("args cannot be emtpy")

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


class AParser(argparse.ArgumentParser):
    def error(self, message):
        description = """
To use with Determined >= 0.17.7, add the following to your experiment config:

entrypoint:
  - python3
  - -m
  - idle_gpu_watcher
  - --min-percentage
  - "5"
  - --sample-freq
  - "5"
  - --sample-window
  - "10"
  - --startup-delay-samples
  - "6"
  - --debug
  - python3
  - -m
  - determined.launch.autohorovod
  - model_def:Trial"""
        self.print_help()
        print(description)
        sys.stderr.write(f"\nerror: {message}\n")
        sys.exit(2)


if __name__ == "__main__":

    parser = AParser(description=
                     "Launch a process and record GPU usage every sample_freq seconds. "
                     "Exit if at least num_samples are below min_percentage after delay_samples has"
                     " passed.")

    parser.add_argument("--min-percentage", type=int, default=5,
                        help="Minimum GPU usage which triggers a failure")
    parser.add_argument("--sample-freq", type=int, default=5,
                        help="How frequently to sample each GPU usage")
    parser.add_argument("--sample-window", type=int, default=12,
                        help="Number of samples to look at when evaluating GPU usage")
    parser.add_argument("--startup-delay-samples", type=int, default=12,
                        help="Number of samples to wait before evaluating GPU usage")
    parser.add_argument("--debug", action="store_true", help="Display debug information")
    parser.add_argument("cmd", nargs=argparse.REMAINDER, help="Command to run")
    args = parser.parse_args()

    watcher = IdleGpuWatcher(
        min_threshhold_percentage=args.min_percentage,
        sample_freq=args.sample_freq,
        num_samples=args.sample_window,
        delay_samples=args.startup_delay_samples,
        debug=args.debug
    )

    sys.exit(watcher.watch_process(args.cmd))
