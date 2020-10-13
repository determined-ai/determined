from typing import Dict, List, Optional, cast


class ExperimentConfig(dict):
    def debug_enabled(self) -> bool:
        return bool(self.get("debug", False))

    def horovod_optional_args(self) -> List[str]:
        return cast(List, self.get("data", {}).get("__det_dtrain_args", []))

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

    def slots_per_trial(self) -> int:
        return int(self["resources"]["slots_per_trial"])

    def experiment_seed(self) -> int:
        return int(self.get("reproducibility", {}).get("experiment_seed", 0))

    def get_data_layer_type(self) -> str:
        return cast(str, self["data_layer"]["type"])

    def get_records_per_epoch(self) -> Optional[int]:
        records_per_epoch = self.get("records_per_epoch")
        return int(records_per_epoch) if records_per_epoch is not None else None

    def get_min_validation_period(self) -> Dict:
        min_validation_period = self.get("min_validation_period", {})
        assert isinstance(min_validation_period, dict)
        return min_validation_period
