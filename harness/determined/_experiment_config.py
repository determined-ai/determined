from typing import List, cast


class ExperimentConfig(dict):
    def debug_enabled(self) -> bool:
        return bool(self.get("debug", False))

    def horovod_optional_args(self) -> List[str]:
        return cast(List, self.get("data", {}).get("__det_dtrain_args", []))

    def profile_frequency(self) -> int:
        return int(self.get("data", {}).get("__det_profile_frequency", 0))

    def scheduling_unit(self) -> int:
        return int(self.get("scheduling_unit", 100))

    def native_enabled(self) -> bool:
        return "internal" in self and self["internal"] is not None and "native" in self["internal"]

    def native_parallel_enabled(self) -> bool:
        return bool(self["resources"]["native_parallel"])

    # TODO(DET-3262): remove this backward compatibility.
    def mixed_precision_enabled(self) -> bool:
        return bool(self["optimizations"]["mixed_precision"] != "O0")

    def averaging_training_metrics_enabled(self) -> bool:
        return bool(self["optimizations"]["average_training_metrics"])

    # TODO (DET-3798): Remove once customers are migrated.
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

    def get_data_layer_type(self) -> str:
        return cast(str, self["data_layer"]["type"])
