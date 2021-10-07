import enum
import time
from typing import Any, Dict, List, Optional, TypeVar, Union

from determined.common import schemas


class DeviceV0(schemas.SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v0/device.json"
    container_path: str
    host_path: str
    mode: Optional[bool] = None

    @schemas.auto_init
    def __init__(
        self,
        container_path: str,
        host_path: str,
        mode: Optional[bool] = None,
    ) -> None:
        pass


class ResourcesConfigV0(schemas.SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v0/resources.json"
    agent_label: Optional[str] = None
    devices: Optional[List[DeviceV0]] = None
    max_slots: Optional[int] = None
    native_parallel: Optional[bool] = None
    priority: Optional[int] = None
    resource_pool: Optional[str] = None
    shm_size: Optional[int] = None
    slots_per_trial: Optional[int] = None
    weight: Optional[float] = None

    @schemas.auto_init
    def __init__(
        self,
        agent_label: Optional[str] = None,
        devices: Optional[List[DeviceV0]] = None,
        max_slots: Optional[int] = None,
        native_parallel: Optional[bool] = None,
        priority: Optional[int] = None,
        resource_pool: Optional[str] = None,
        shm_size: Optional[int] = None,
        slots_per_trial: Optional[int] = None,
        weight: Optional[float] = None,
    ) -> None:
        pass


class OptimizationsConfigV0(schemas.SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v0/optimizations.json"
    aggregation_frequency: Optional[int] = None
    auto_tune_tensor_fusion: Optional[bool] = None
    average_aggregated_gradients: Optional[bool] = None
    average_training_metrics: Optional[bool] = None
    gradient_compression: Optional[bool] = None
    grad_updates_size_file: Optional[str] = None
    mixed_precision: Optional[str] = None
    tensor_fusion_cycle_time: Optional[int] = None
    tensor_fusion_threshold: Optional[int] = None

    @schemas.auto_init
    def __init__(
        self,
        aggregation_frequency: Optional[int] = None,
        auto_tune_tensor_fusion: Optional[bool] = None,
        average_aggregated_gradients: Optional[bool] = None,
        average_training_metrics: Optional[bool] = None,
        gradient_compression: Optional[bool] = None,
        grad_updates_size_file: Optional[str] = None,
        mixed_precision: Optional[str] = None,
        tensor_fusion_cycle_time: Optional[int] = None,
        tensor_fusion_threshold: Optional[int] = None,
    ) -> None:
        pass


class BindMountV0(schemas.SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v0/bind-mount.json"
    container_path: str
    host_path: str
    propagation: Optional[str] = None
    read_only: Optional[bool] = None

    @schemas.auto_init
    def __init__(
        self,
        container_path: str,
        host_path: str,
        propagation: Optional[str] = None,
        read_only: Optional[bool] = None,
    ) -> None:
        pass


class ReproducibilityConfigV0(schemas.SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v0/reproducibility.json"
    experiment_seed: Optional[int] = None

    @schemas.auto_init
    def __init__(
        self,
        experiment_seed: Optional[int] = None,
    ) -> None:
        pass

    def runtime_defaults(self) -> None:
        if self.experiment_seed is None:
            self.experiment_seed = int(time.time())


class ProfilingConfigV0(schemas.SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v0/profiling.json"
    enabled: Optional[bool] = None
    begin_on_batch: Optional[int] = None
    end_after_batch: Optional[int] = None
    sync_timings: Optional[bool] = None

    @schemas.auto_init
    def __init__(
        self,
        enabled: Optional[bool] = None,
        begin_on_batch: Optional[int] = None,
        end_after_batch: Optional[int] = None,
        sync_timings: Optional[bool] = None,
    ) -> None:
        pass


class LengthV0(schemas.SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v0/length.json"
    batches: Optional[int] = None
    epochs: Optional[int] = None
    records: Optional[int] = None

    @schemas.auto_init
    def __init__(
        self,
        batches: Optional[int] = None,
        epochs: Optional[int] = None,
        records: Optional[int] = None,
    ) -> None:
        pass

    def to_dict(self, explicit_nones: bool = False) -> Any:
        if not explicit_nones:
            return super().to_dict(explicit_nones=False)
        if self.batches is not None or self.epochs is not None or self.records is not None:
            return super().to_dict(explicit_nones=False)
        # explicit_nones means we pick any value... never show all three; that's nonsensical.
        return {"batches": None}


class DataLayerConfigV0(schemas.UnionBase):
    _id = "http://determined.ai/schemas/expconf/v0/data-layer.json"
    _union_key = "type"


@DataLayerConfigV0.member("shared_fs")
class SharedFSDataLayerConfigV0(schemas.SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v0/data-layer-shared-fs.json"
    container_storage_path: Optional[str] = None
    host_storage_path: Optional[str] = None

    @schemas.auto_init
    def __init__(
        self,
        container_storage_path: Optional[str] = None,
        host_storage_path: Optional[str] = None,
    ) -> None:
        pass


@DataLayerConfigV0.member("s3")
class S3DataLayerConfigV0(schemas.SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v0/data-layer-s3.json"
    bucket: str
    bucket_directory_path: str
    local_cache_container_path: Optional[str] = None
    local_cache_host_path: Optional[str] = None
    access_key: Optional[str] = None
    secret_key: Optional[str] = None
    endpoint_url: Optional[str] = None

    @schemas.auto_init
    def __init__(
        self,
        bucket: str,
        bucket_directory_path: str,
        local_cache_container_path: Optional[str] = None,
        local_cache_host_path: Optional[str] = None,
        access_key: Optional[str] = None,
        secret_key: Optional[str] = None,
        endpoint_url: Optional[str] = None,
    ) -> None:
        pass


@DataLayerConfigV0.member("gcs")
class GCSDataLayerConfigV0(schemas.SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v0/data-layer-gcs.json"
    bucket: str
    bucket_directory_path: str
    local_cache_container_path: Optional[str] = None
    local_cache_host_path: Optional[str] = None

    @schemas.auto_init
    def __init__(
        self,
        bucket: str,
        bucket_directory_path: str,
        local_cache_container_path: Optional[str] = None,
        local_cache_host_path: Optional[str] = None,
    ) -> None:
        pass


DataLayerConfigV0_Type = Union[SharedFSDataLayerConfigV0, S3DataLayerConfigV0, GCSDataLayerConfigV0]
DataLayerConfigV0.finalize(DataLayerConfigV0_Type)


class EnvironmentImageV0(schemas.SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v0/environment-image.json"
    cpu: Optional[str] = None
    gpu: Optional[str] = None

    @schemas.auto_init
    def __init__(
        self,
        cpu: Optional[str] = None,
        gpu: Optional[str] = None,
    ) -> None:
        pass

    @classmethod
    def from_dict(cls, d: Union[dict, str], prevalidated: bool = False) -> "EnvironmentImageV0":
        # Accept either a string or a map of strings to strings.
        if isinstance(d, str):
            d = {"cpu": d, "gpu": d}
        return super().from_dict(d, prevalidated)

    def runtime_defaults(self) -> None:
        if self.cpu is None:
            self.cpu = (
                "determinedai/environments:py-3.8-pytorch-1.9-lightning-1.3-tf-2.4-cpu-f286b87"
            )
        if self.gpu is None:
            self.gpu = (
                "determinedai/environments:cuda-11.1-pytorch-1.9-lightning-1.3-tf-2.4-gpu-f286b87"
            )


class EnvironmentVariablesV0(schemas.SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v0/environment-variables.json"
    cpu: Optional[List[str]] = None
    gpu: Optional[List[str]] = None

    @schemas.auto_init
    def __init__(
        self,
        cpu: Optional[List[str]] = None,
        gpu: Optional[List[str]] = None,
    ) -> None:
        pass

    @classmethod
    def from_dict(
        cls, d: Union[dict, list, tuple], prevalidated: bool = False
    ) -> "EnvironmentVariablesV0":
        # Accept either a list of strings or a map of strings to lists of strings.
        if isinstance(d, (list, tuple)):
            d = {"cpu": d, "gpu": d}
        return super().from_dict(d, prevalidated)


class RegistryAuthConfigV0(schemas.SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v0/registry-auth.json"
    auth: Optional[str] = None
    email: Optional[str] = None
    identitytoken: Optional[str] = None
    password: Optional[str] = None
    registrytoken: Optional[str] = None
    serveraddress: Optional[str] = None
    username: Optional[str] = None

    @schemas.auto_init
    def __init__(
        self,
        auth: Optional[str] = None,
        email: Optional[str] = None,
        identitytoken: Optional[str] = None,
        password: Optional[str] = None,
        registrytoken: Optional[str] = None,
        serveraddress: Optional[str] = None,
        username: Optional[str] = None,
    ) -> None:
        pass

    def to_dict(self, explicit_nones: bool = False) -> Any:
        # Match go's docker library's omitempty behavior.
        return super().to_dict(explicit_nones=False)


class EnvironmentConfigV0(schemas.SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v0/environment.json"
    add_capabilities: Optional[List[str]] = None
    drop_capabilities: Optional[List[str]] = None
    environment_variables: Optional[EnvironmentVariablesV0] = None
    force_pull_image: Optional[bool] = None
    image: Optional[EnvironmentImageV0] = None
    pod_spec: Optional[Dict[str, Any]] = None
    ports: Optional[Dict[str, int]] = None
    registry_auth: Optional[RegistryAuthConfigV0] = None

    @schemas.auto_init
    def __init__(
        self,
        add_capabilities: Optional[List[str]] = None,
        drop_capabilities: Optional[List[str]] = None,
        environment_variables: Optional[EnvironmentVariablesV0] = None,
        force_pull_image: Optional[bool] = None,
        image: Optional[EnvironmentImageV0] = None,
        pod_spec: Optional[Dict[str, Any]] = None,
        ports: Optional[Dict[str, int]] = None,
        registry_auth: Optional[RegistryAuthConfigV0] = None,
    ) -> None:
        pass


H = TypeVar("H", bound="HyperparameterV0")


class HyperparameterV0(schemas.UnionBase):
    _id = "http://determined.ai/schemas/expconf/v0/hyperparameter.json"
    _union_key = "type"

    @classmethod
    def from_dict(cls, d: Any, prevalidated: bool = False) -> schemas.SchemaBase:  # type: ignore
        if not isinstance(d, dict) or "type" not in d:
            # Implicit const.
            return ConstHyperparameterV0(val=d)

        return super().from_dict(d, prevalidated)


@HyperparameterV0.member("const")
class ConstHyperparameterV0(schemas.SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v0/hyperparameter-const.json"
    val: Any

    @schemas.auto_init
    def __init__(
        self,
        val: Any,
    ) -> None:
        pass


@HyperparameterV0.member("int")
class IntHyperparameterV0(schemas.SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v0/hyperparameter-int.json"
    minval: int
    maxval: int
    count: Optional[int] = None

    @schemas.auto_init
    def __init__(
        self,
        minval: int,
        maxval: int,
        count: Optional[int] = None,
    ) -> None:
        pass


@HyperparameterV0.member("double")
class DoubleHyperparameterV0(schemas.SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v0/hyperparameter-double.json"
    minval: float
    maxval: float
    count: Optional[int] = None

    @schemas.auto_init
    def __init__(
        self,
        minval: float,
        maxval: float,
        count: Optional[int] = None,
    ) -> None:
        pass


@HyperparameterV0.member("log")
class LogHyperparameterV0(schemas.SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v0/hyperparameter-log.json"
    minval: float
    maxval: float
    base: float
    count: Optional[int] = None

    @schemas.auto_init
    def __init__(
        self,
        minval: float,
        maxval: float,
        base: float,
        count: Optional[int] = None,
    ) -> None:
        pass


@HyperparameterV0.member("categorical")
class CategoricalHyperparameterV0(schemas.SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v0/hyperparameter-categorical.json"
    vals: List[Any]

    @schemas.auto_init
    def __init__(
        self,
        vals: List[Any],
    ) -> None:
        pass


HyperparameterV0_Type = Union[
    ConstHyperparameterV0,
    IntHyperparameterV0,
    DoubleHyperparameterV0,
    LogHyperparameterV0,
    CategoricalHyperparameterV0,
]
HyperparameterV0.finalize(HyperparameterV0_Type)


@schemas.register_known_type
class AdaptiveMode(enum.Enum):
    CONSERVATIVE = "conservative"
    STANDARD = "standard"
    AGGRESSIVE = "aggressive"


class SearcherConfigV0(schemas.UnionBase):
    _id = "http://determined.ai/schemas/expconf/v0/searcher.json"
    _union_key = "name"


@SearcherConfigV0.member("single")
class SingleConfigV0(schemas.SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v0/searcher-single.json"
    max_length: LengthV0
    metric: str
    smaller_is_better: Optional[bool] = None
    source_checkpoint_uuid: Optional[str] = None
    source_trial_id: Optional[int] = None

    @schemas.auto_init
    def __init__(
        self,
        max_length: LengthV0,
        metric: str,
        smaller_is_better: Optional[bool] = None,
        source_checkpoint_uuid: Optional[str] = None,
        source_trial_id: Optional[int] = None,
    ) -> None:
        pass


@SearcherConfigV0.member("random")
class RandomConfigV0(schemas.SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v0/searcher-random.json"
    max_length: LengthV0
    max_trials: int
    metric: str
    max_concurrent_trials: Optional[int] = None
    smaller_is_better: Optional[bool] = None
    source_checkpoint_uuid: Optional[str] = None
    source_trial_id: Optional[int] = None

    @schemas.auto_init
    def __init__(
        self,
        max_length: LengthV0,
        max_trials: int,
        metric: str,
        max_concurrent_trials: Optional[int] = None,
        smaller_is_better: Optional[bool] = None,
        source_checkpoint_uuid: Optional[str] = None,
        source_trial_id: Optional[int] = None,
    ) -> None:
        pass


@SearcherConfigV0.member("grid")
class GridConfigV0(schemas.SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v0/searcher-grid.json"
    max_length: LengthV0
    metric: str
    max_concurrent_trials: Optional[int] = None
    smaller_is_better: Optional[bool] = None
    source_checkpoint_uuid: Optional[str] = None
    source_trial_id: Optional[int] = None

    @schemas.auto_init
    def __init__(
        self,
        max_length: LengthV0,
        metric: str,
        max_concurrent_trials: Optional[int] = None,
        smaller_is_better: Optional[bool] = None,
        source_checkpoint_uuid: Optional[str] = None,
        source_trial_id: Optional[int] = None,
    ) -> None:
        pass


@SearcherConfigV0.member("async_halving")
class AsyncHalvingConfigV0(schemas.SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v0/searcher-async-halving.json"
    max_length: LengthV0
    max_trials: int
    metric: str
    num_rungs: int
    divisor: Optional[float] = None
    max_concurrent_trials: Optional[int] = None
    smaller_is_better: Optional[bool] = None
    source_checkpoint_uuid: Optional[str] = None
    source_trial_id: Optional[int] = None
    stop_once: Optional[int] = None

    @schemas.auto_init
    def __init__(
        self,
        max_length: LengthV0,
        max_trials: int,
        metric: str,
        num_rungs: int,
        divisor: Optional[float] = None,
        max_concurrent_trials: Optional[int] = None,
        smaller_is_better: Optional[bool] = None,
        source_checkpoint_uuid: Optional[str] = None,
        source_trial_id: Optional[int] = None,
        stop_once: Optional[int] = None,
    ) -> None:
        pass


@SearcherConfigV0.member("adaptive_asha")
class AdaptiveASHAConfigV0(schemas.SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v0/searcher-adaptive-asha.json"
    max_length: LengthV0
    max_trials: int
    metric: str
    bracket_rungs: Optional[List[int]] = None
    divisor: Optional[float] = None
    max_concurrent_trials: Optional[int] = None
    max_rungs: Optional[int] = None
    mode: Optional[AdaptiveMode] = None
    smaller_is_better: Optional[bool] = None
    source_checkpoint_uuid: Optional[str] = None
    source_trial_id: Optional[int] = None
    stop_once: Optional[int] = None

    @schemas.auto_init
    def __init__(
        self,
        max_length: LengthV0,
        max_trials: int,
        metric: str,
        bracket_rungs: Optional[List[int]] = None,
        divisor: Optional[float] = None,
        max_concurrent_trials: Optional[int] = None,
        max_rungs: Optional[int] = None,
        mode: Optional[AdaptiveMode] = None,
        smaller_is_better: Optional[bool] = None,
        source_checkpoint_uuid: Optional[str] = None,
        source_trial_id: Optional[int] = None,
        stop_once: Optional[int] = None,
    ) -> None:
        pass


class PBTReplaceConfig(schemas.SchemaBase):
    truncate_fraction: float

    @schemas.auto_init
    def __init__(
        self,
        truncate_fraction: float,
    ) -> None:
        pass


class PBTExploreConfig(schemas.SchemaBase):
    resample_probability: float
    perturb_factor: float

    @schemas.auto_init
    def __init__(
        self,
        resample_probability: float,
        perturb_factor: float,
    ) -> None:
        pass


@SearcherConfigV0.member("pbt")
class PBTConfigV0(schemas.SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v0/searcher-pbt.json"
    length_per_round: LengthV0
    metric: str
    num_rounds: int
    population_size: int
    replace_function: PBTReplaceConfig
    explore_function: PBTExploreConfig
    smaller_is_better: Optional[bool] = None
    source_checkpoint_uuid: Optional[str] = None
    source_trial_id: Optional[int] = None

    @schemas.auto_init
    def __init__(
        self,
        length_per_round: LengthV0,
        metric: str,
        num_rounds: int,
        population_size: int,
        replace_function: PBTReplaceConfig,
        explore_function: PBTExploreConfig,
        smaller_is_better: Optional[bool] = None,
        source_checkpoint_uuid: Optional[str] = None,
        source_trial_id: Optional[int] = None,
    ) -> None:
        pass


# This is an EOL searcher, not to be used in new experiments.
@SearcherConfigV0.member("sync_halving")
class SyncHalvingConfigV0(schemas.SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v0/searcher-sync-halving.json"
    budget: LengthV0
    max_length: LengthV0
    metric: str
    num_rungs: int
    divisor: Optional[float] = None
    smaller_is_better: Optional[bool] = None
    source_checkpoint_uuid: Optional[str] = None
    source_trial_id: Optional[int] = None
    train_stragglers: Optional[bool] = None

    @schemas.auto_init
    def __init__(
        self,
        budget: LengthV0,
        max_length: LengthV0,
        metric: str,
        num_rungs: int,
        divisor: Optional[float] = None,
        smaller_is_better: Optional[bool] = None,
        source_checkpoint_uuid: Optional[str] = None,
        source_trial_id: Optional[int] = None,
        train_stragglers: Optional[bool] = None,
    ) -> None:
        pass


# This is an EOL searcher, not to be used in new experiments.
@SearcherConfigV0.member("adaptive")
class AdaptiveConfigV0(schemas.SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v0/searcher-adaptive.json"
    budget: LengthV0
    max_length: LengthV0
    metric: str
    bracket_rungs: Optional[List[int]] = None
    divisor: Optional[float] = None
    max_rungs: Optional[int] = None
    mode: Optional[AdaptiveMode] = None
    smaller_is_better: Optional[bool] = None
    source_checkpoint_uuid: Optional[str] = None
    source_trial_id: Optional[int] = None
    train_stragglers: Optional[bool] = None

    @schemas.auto_init
    def __init__(
        self,
        budget: LengthV0,
        max_length: LengthV0,
        metric: str,
        bracket_rungs: Optional[List[int]] = None,
        divisor: Optional[float] = None,
        max_rungs: Optional[int] = None,
        mode: Optional[AdaptiveMode] = None,
        smaller_is_better: Optional[bool] = None,
        source_checkpoint_uuid: Optional[str] = None,
        source_trial_id: Optional[int] = None,
        train_stragglers: Optional[bool] = None,
    ) -> None:
        pass


# This is an EOL searcher, not to be used in new experiments.
@SearcherConfigV0.member("adaptive_simple")
class AdaptiveSimpleConfigV0(schemas.SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v0/searcher-adaptive-simple.json"
    max_length: LengthV0
    max_trials: int
    metric: str
    divisor: Optional[float] = None
    max_rungs: Optional[int] = None
    mode: Optional[AdaptiveMode] = None
    smaller_is_better: Optional[bool] = None
    source_checkpoint_uuid: Optional[str] = None
    source_trial_id: Optional[int] = None

    @schemas.auto_init
    def __init__(
        self,
        max_length: LengthV0,
        max_trials: int,
        metric: str,
        divisor: Optional[float] = None,
        max_rungs: Optional[int] = None,
        mode: Optional[AdaptiveMode] = None,
        smaller_is_better: Optional[bool] = None,
        source_checkpoint_uuid: Optional[str] = None,
        source_trial_id: Optional[int] = None,
    ) -> None:
        pass


SearcherConfigV0_Type = Union[
    SingleConfigV0,
    RandomConfigV0,
    GridConfigV0,
    AsyncHalvingConfigV0,
    AdaptiveASHAConfigV0,
    PBTConfigV0,
    # EOL searchers:
    SyncHalvingConfigV0,
    AdaptiveConfigV0,
    AdaptiveSimpleConfigV0,
]
SearcherConfigV0.finalize(SearcherConfigV0_Type)


class CheckpointStorageConfigV0(schemas.UnionBase):
    _id = "http://determined.ai/schemas/expconf/v0/checkpoint-storage.json"
    _union_key = "type"


@CheckpointStorageConfigV0.member("shared_fs")
class SharedFSConfigV0(schemas.SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v0/shared-fs.json"
    host_path: str
    checkpoint_path: Optional[str] = None
    container_path: Optional[str] = None
    propagation: Optional[str] = None
    save_experiment_best: Optional[int] = None
    save_trial_best: Optional[int] = None
    save_trial_latest: Optional[int] = None
    storage_path: Optional[str] = None
    tensorboard_path: Optional[str] = None

    @schemas.auto_init
    def __init__(
        self,
        host_path: str,
        checkpoint_path: Optional[str] = None,
        container_path: Optional[str] = None,
        propagation: Optional[str] = None,
        save_experiment_best: Optional[int] = None,
        save_trial_best: Optional[int] = None,
        save_trial_latest: Optional[int] = None,
        storage_path: Optional[str] = None,
        tensorboard_path: Optional[str] = None,
    ) -> None:
        pass


@CheckpointStorageConfigV0.member("hdfs")
class HDFSConfigV0(schemas.SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v0/hdfs.json"
    hdfs_url: str
    hdfs_path: str
    save_experiment_best: Optional[int] = None
    save_trial_best: Optional[int] = None
    save_trial_latest: Optional[int] = None
    user: Optional[str] = None

    @schemas.auto_init
    def __init__(
        self,
        hdfs_url: str,
        hdfs_path: str,
        save_experiment_best: Optional[int] = None,
        save_trial_best: Optional[int] = None,
        save_trial_latest: Optional[int] = None,
        user: Optional[str] = None,
    ) -> None:
        pass


@CheckpointStorageConfigV0.member("s3")
class S3ConfigV0(schemas.SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v0/s3.json"
    bucket: str
    access_key: Optional[str] = None
    endpoint_url: Optional[str] = None
    save_experiment_best: Optional[int] = None
    save_trial_best: Optional[int] = None
    save_trial_latest: Optional[int] = None
    secret_key: Optional[str] = None

    @schemas.auto_init
    def __init__(
        self,
        bucket: str,
        access_key: Optional[str] = None,
        endpoint_url: Optional[str] = None,
        save_experiment_best: Optional[int] = None,
        save_trial_best: Optional[int] = None,
        save_trial_latest: Optional[int] = None,
        secret_key: Optional[str] = None,
    ) -> None:
        pass


@CheckpointStorageConfigV0.member("gcs")
class GCSConfigV0(schemas.SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v0/gcs.json"
    bucket: str
    save_experiment_best: Optional[int] = None
    save_trial_best: Optional[int] = None
    save_trial_latest: Optional[int] = None

    @schemas.auto_init
    def __init__(
        self,
        bucket: str,
        save_experiment_best: Optional[int] = None,
        save_trial_best: Optional[int] = None,
        save_trial_latest: Optional[int] = None,
    ) -> None:
        pass


@CheckpointStorageConfigV0.member("azure")
class AzureConfigV0(schemas.SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v0/azure.json"
    container: str
    connection_string: Optional[str] = None
    account_url: Optional[str] = None
    credential: Optional[str] = None
    save_experiment_best: Optional[int] = None
    save_trial_best: Optional[int] = None
    save_trial_latest: Optional[int] = None

    @schemas.auto_init
    def __init__(
        self,
        container: str,
        connection_string: Optional[str] = None,
        account_url: Optional[str] = None,
        credential: Optional[str] = None,
        save_experiment_best: Optional[int] = None,
        save_trial_best: Optional[int] = None,
        save_trial_latest: Optional[int] = None,
    ) -> None:
        pass


CheckpointStorageConfigV0_Type = Union[
    SharedFSConfigV0, HDFSConfigV0, S3ConfigV0, GCSConfigV0, AzureConfigV0
]
CheckpointStorageConfigV0.finalize(CheckpointStorageConfigV0_Type)


class ExperimentConfigV0(schemas.SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v0/experiment.json"

    # Note that the fields internal, security, and tensorboard_storage are omitted entirely
    # as internal is totally non-user-facing and security and tensorboard_storage are completely
    # ignored byt the system.  These fields are allowed during validation but will be ignored by
    # .from_dict().

    # Fields which must be defined by the user.
    hyperparameters: Dict[str, HyperparameterV0_Type]
    searcher: SearcherConfigV0

    # Fields which can be omitted or defined at the cluster level.
    bind_mounts: Optional[List[BindMountV0]] = None
    checkpoint_policy: Optional[str] = None
    checkpoint_storage: Optional[CheckpointStorageConfigV0_Type] = None
    data_layer: Optional[DataLayerConfigV0_Type] = None
    data: Optional[Dict[str, Any]] = None
    debug: Optional[bool] = None
    description: Optional[str] = None
    entrypoint: Optional[str] = None
    environment: Optional[EnvironmentConfigV0] = None
    # internal: Optional[InternalConfigV0] = None
    labels: Optional[str] = None
    max_restarts: Optional[int] = None
    min_checkpoint_period: Optional[LengthV0] = None
    min_validation_period: Optional[LengthV0] = None
    name: Optional[str] = None
    optimizations: Optional[OptimizationsConfigV0] = None
    perform_initial_validation: Optional[bool] = None
    profiling: Optional[ProfilingConfigV0] = None
    records_per_epoch: Optional[int] = None
    reproducibility: Optional[ReproducibilityConfigV0] = None
    resources: Optional[ResourcesConfigV0] = None
    scheduling_unit: Optional[int] = None
    # security: Optional[SecurityConfigV0] = None
    # tensorboard_storage: Optional[TensorboardStorageConfigV0_Type] = None

    @schemas.auto_init
    def __init__(
        self,
        hyperparameters: Dict[str, HyperparameterV0_Type],
        searcher: SearcherConfigV0,
        bind_mounts: Optional[List[BindMountV0]] = None,
        checkpoint_policy: Optional[str] = None,
        checkpoint_storage: Optional[CheckpointStorageConfigV0_Type] = None,
        data_layer: Optional[DataLayerConfigV0_Type] = None,
        data: Optional[Dict[str, Any]] = None,
        debug: Optional[bool] = None,
        description: Optional[str] = None,
        entrypoint: Optional[str] = None,
        environment: Optional[EnvironmentConfigV0] = None,
        # internal: Optional[InternalConfigV0] = None,
        labels: Optional[str] = None,
        max_restarts: Optional[int] = None,
        min_checkpoint_period: Optional[LengthV0] = None,
        min_validation_period: Optional[LengthV0] = None,
        name: Optional[str] = None,
        optimizations: Optional[OptimizationsConfigV0] = None,
        perform_initial_validation: Optional[bool] = None,
        profiling: Optional[ProfilingConfigV0] = None,
        records_per_epoch: Optional[int] = None,
        reproducibility: Optional[ReproducibilityConfigV0] = None,
        resources: Optional[ResourcesConfigV0] = None,
        scheduling_unit: Optional[int] = None,
        # security: Optional[SecurityConfigV0] = None,
        # tensorboard_storage: Optional[TensorboardStorageConfigV0_Type] = None,
    ) -> None:
        pass

    def runtime_defaults(self) -> None:
        if self.name is None:
            self.name = "Experiment (really-bad-petname)"


# Test Structs Below:


class TestSubV0(schemas.SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v0/test-sub.json"

    val_y: Optional[str] = None

    @schemas.auto_init
    def __init__(
        self,
        val_y: Optional[str] = None,
    ):
        pass


class TestUnionV0(schemas.UnionBase):
    _id = "http://determined.ai/schemas/expconf/v0/test-union.json"
    _union_key = "type"


@TestUnionV0.member("a")
class TestUnionAV0(schemas.SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v0/test-union-a.json"

    val_a: int
    common_val: Optional[str] = None

    @schemas.auto_init
    def __init__(
        self,
        val_a: int,
        common_val: Optional[str] = None,
    ):
        pass


@TestUnionV0.member("b")
class TestUnionBV0(schemas.SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v0/test-union-b.json"
    _union_key = "type"
    _union_id = "b"

    val_b: int
    common_val: Optional[str] = None

    @schemas.auto_init
    def __init__(
        self,
        val_b: int,
        common_val: Optional[str] = None,
    ):
        pass


TestUnionV0_Type = Union[TestUnionAV0, TestUnionBV0]
TestUnionV0.finalize(TestUnionV0_Type)


class TestRootV0(schemas.SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v0/test-root.json"

    val_x: int
    defaulted_array: Optional[List[str]] = None
    nodefault_array: Optional[List[str]] = None
    sub_obj: Optional[TestSubV0] = None
    sub_union: Optional[TestUnionV0_Type] = None
    runtime_defaultable: Optional[int] = None

    @schemas.auto_init
    def __init__(
        self,
        val_x: int,
        defaulted_array: Optional[List[str]] = None,
        nodefault_array: Optional[List[str]] = None,
        sub_obj: Optional[TestSubV0] = None,
        sub_union: Optional[TestUnionV0_Type] = None,
        runtime_defaultable: Optional[int] = None,
    ):
        pass

    def runtime_defaults(self) -> None:
        if self.runtime_defaultable is None:
            self.runtime_defaultable = 10
