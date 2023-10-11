from typing import Any, Dict, List, Optional, Tuple, Union


class ExperimentConfig(dict):
    def debug_enabled(self) -> bool:
        return bool(self.get("debug", False))

    def scheduling_unit(self) -> int:
        return int(self.get("scheduling_unit", 100))

    def native_parallel_enabled(self) -> bool:
        return bool(self["resources"]["native_parallel"])

    def average_training_metrics_enabled(self) -> bool:
        return bool(self["optimizations"]["average_training_metrics"])

    def slots_per_trial(self) -> int:
        return int(self["resources"]["slots_per_trial"])

    def experiment_seed(self) -> int:
        return int(self.get("reproducibility", {}).get("experiment_seed", 0))

    def profiling_enabled(self) -> bool:
        return bool(self.get("profiling", {}).get("enabled", False))

    def profiling_interval(self) -> Tuple[int, Optional[int]]:
        if not self.profiling_enabled():
            return 0, 0

        return self["profiling"]["begin_on_batch"], self["profiling"].get("end_after_batch", None)

    def profiling_sync_timings(self) -> bool:
        return bool(self.get("profiling", {}).get("sync_timings", True))

    def get_records_per_epoch(self) -> Optional[int]:
        records_per_epoch = self.get("records_per_epoch")
        return int(records_per_epoch) if records_per_epoch is not None else None

    def get_min_validation_period(self) -> Dict:
        min_validation_period = self.get("min_validation_period", {})
        assert isinstance(min_validation_period, dict)
        return min_validation_period

    def get_searcher_metric(self) -> str:
        searcher_metric = self.get("searcher", {}).get("metric")
        assert isinstance(
            searcher_metric, str
        ), f"searcher metric ({searcher_metric}) is not a string"

        return searcher_metric

    def get_min_checkpoint_period(self) -> Dict:
        min_checkpoint_period = self.get("min_checkpoint_period", {})
        assert isinstance(min_checkpoint_period, dict)
        return min_checkpoint_period

    def get_optimizations_config(self) -> Dict[str, Any]:
        """
        Return the optimizations configuration.
        """
        return self.get("optimizations", {})

    def get_checkpoint_storage(self) -> Dict[str, Any]:
        return self.get("checkpoint_storage", {})

    def get_entrypoint(self) -> Union[str, List[str]]:
        entrypoint = self["entrypoint"]
        if not isinstance(entrypoint, (str, list)):
            raise ValueError("invalid entrypoint in experiment config: {entrypoint}")
        if isinstance(entrypoint, list) and any(not isinstance(e, str) for e in entrypoint):
            raise ValueError("invalid entrypoint in experiment config: {entrypoint}")
        return entrypoint
