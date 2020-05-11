from typing import Any


def get_validation_metric(metric_name: str, validation: Any) -> Any:
    if not validation or not validation["metrics"]:
        return None

    return validation["metrics"].get("validation_metrics", {}).get(metric_name)
