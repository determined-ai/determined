from typing import List, cast


class ExperimentConfig(dict):
    def debug_enabled(self) -> bool:
        return bool(self.get("debug", False))

    def horovod_optional_args(self) -> List[str]:
        return cast(List, self.get("data", {}).get("__det_dtrain_args", []))

    def profile_frequency(self) -> int:
        return int(self.get("data", {}).get("__det_profile_frequency", 0))

    def batches_per_step(self) -> int:
        return int(self.get("batches_per_step", 100))

    def validation_freq(self) -> int:
        return int(self.get("min_validation_period", 100))

    def native_enabled(self) -> bool:
        return "internal" in self and self["internal"] is not None and "native" in self["internal"]

    def native_parallel_enabled(self) -> bool:
        return bool(self["resources"]["native_parallel"])

    def mixed_precision_enabled(self) -> bool:
        return bool(self["optimizations"]["mixed_precision"] != "O0")

    def input_from_dataflow(self) -> bool:
        # When using tensorpack dataflows as input, it's inefficient
        # to apply sharding, so we only apply sharding to the test set.
        # To have each worker process unique data, we set different random
        # seeds in every train process, and require users to shuffle
        # their train data, but not their test data.
        return bool(self.get("data", {}).get("dataflow_to_tf_dataset", False))

    def slots_per_trial(self) -> int:
        return int(self["resources"]["slots_per_trial"])

    def experiment_seed(self) -> int:
        return int(self.get("reproducibility", {}).get("experiment_seed", 0))
