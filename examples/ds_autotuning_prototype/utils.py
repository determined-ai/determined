import json
import os
import pathlib
import re
from typing import Any, Dict, List

import constants


def upper_case_dict_key(d: Dict[str, Any], key: str) -> Dict[str, Any]:
    upper_d = {}
    for k, v in d.items():
        new_k = k.upper() if key == k else k
        if isinstance(v, dict):
            upper_d[new_k] = upper_case_dict_key(v, key)
        else:
            upper_d[new_k] = v
    return upper_d


def lower_case_dict_key(d: Dict[str, Any], key: str) -> Dict[str, Any]:
    lower_d = {}
    for k, v in d.items():
        new_k = k.lower() if key == k else k
        if isinstance(v, dict):
            lower_d[new_k] = lower_case_dict_key(v, key)
        else:
            lower_d[new_k] = v
    return lower_d


def get_non_decimal_number_in_line(line: str) -> float:
    num_str = re.search(r"\b\d+\b", line).group()
    num = float(num_str)
    return num


def get_decimal_number_in_line(line: str) -> float:
    num_str = re.search(r"\b\d*\.\d+\b", line).group()
    num = float(num_str)
    return num


class DSProfilerResults:
    """Class for extracting results from DS profiler output."""

    def __init__(self, path: pathlib.Path) -> None:
        self.path = path

    def get_results_dict_from_path(self) -> Dict[str, float]:
        metrics_with_units = {"iter latency", "FLOPS per GPU", "params per gpu"}
        metrics_without_units = {
            "samples/second",
            "world size",
            "data parallel size",
            "model parallel size",
            "batch size per GPU",
        }
        # The FLOPS and latency computations are reported with units.  We convert everything to
        # FLOPS and seconds.
        units_map = {
            "TFLOPS": 1e12,
            "GFLOPS": 1e9,
            "MFLOPS": 1e6,
            "KFLOPS": 1e3,
            "M": 1e6,
            "K": 1e3,
            "k": 1e3,
            "s": 1,
            "ms": 1e-3,
            "us": 1e-6,
        }
        results_dict = {}
        with open(self.path, "r") as output:
            for line in output:
                line = line.strip()
                for metric in metrics_with_units:
                    if line.startswith(metric + ":"):
                        units_factor = units_map[line.split()[-1]]
                        results_dict[metric] = get_decimal_number_in_line(line) * units_factor
                for metric in metrics_without_units:
                    if line.startswith(metric + ":"):
                        results_dict[metric] = get_non_decimal_number_in_line(line)
        return results_dict

    def get_config(
        self,
        workspace_name: str,
        project_name: str,
        exp_name: str,
        entrypoint: str,
        append_to_name: str = "_results",
    ) -> Dict[str, Any]:
        config = {
            "entrypoint": entrypoint,
            "max_restarts": 5,
            "resources": {"slots_per_trial": 0},
            "searcher": {
                "name": "single",
                "max_length": 0,
                "metric": "none",
            },
            "hyperparameters": None,
        }
        if workspace_name:
            config["workspace"] = workspace_name
        if project_name:
            config["project"] = project_name
        if exp_name:
            config["name"] = exp_name + append_to_name

        results_dict = self.get_results_dict_from_path()

        config["hyperparameters"] = {"results": results_dict, "profiled": True}

        return config


def get_flattened_dict(d: dict, concat_str: str = "_") -> Dict[str, Any]:
    """Flattens a nested dict into a single level dict with concatenated keys."""
    flat_dict = {}

    def flatten(d: dict, parent_key: str = "") -> None:
        for key, val in d.items():
            if parent_key:
                key = parent_key + concat_str + key
            if not isinstance(val, dict):
                assert key not in flat_dict, f'Key "{key}" already exists in dict!!!'
                flat_dict[key] = val
            else:
                flatten(val, key)

    flatten(d)
    return flat_dict
