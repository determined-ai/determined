import enum
import functools
import logging
import pathlib
from typing import Any, Callable, Optional, cast

import tensorflow as tf
import yogadl
from yogadl import storage, tensorflow

import determined as det
from determined import horovod
from determined.horovod import hvd
from determined_common import check


def init_container_storage_path(configured_storage_path: Optional[str]) -> pathlib.Path:
    if configured_storage_path:
        storage_path = pathlib.Path(configured_storage_path)
    else:
        storage_path = pathlib.Path.home().joinpath("data/determined/")

    storage_path.mkdir(exist_ok=True, parents=True)
    return storage_path


class StorageTypes(enum.Enum):
    SHARED_FS = "shared_fs"
    S3 = "s3"
    GCS = "gcs"


class _CacheableDecorator:
    def __init__(
        self,
        env: det.EnvContext,
        hvd_config: horovod.HorovodContext,
        training: bool,
        per_slot_batch_size: int,
    ) -> None:
        self._env = env
        self._hvd_config = hvd_config
        self._training = training
        self._per_slot_batch_size = per_slot_batch_size

        self._offset = 0
        self._shard_rank = 0
        self._num_shards = 1
        self._shuffle_seed = self._env.trial_seed
        self._decorator_used = False

        self._dataset_length = None  # type: Optional[int]

        self._init_offset()
        self._init_shard()

    def _init_offset(self) -> None:
        if not self._training:
            return

        batch_size = self._per_slot_batch_size
        self._offset = self._env.initial_workload.total_batches_processed * batch_size

    def _init_shard(self) -> None:
        if not self._hvd_config.use:
            return

        self._shard_rank = hvd.rank()
        self._num_shards = hvd.size()

    def _configure_storage(self) -> None:
        session_config = None  # type: Optional[tf.compat.v1.ConfigProto]
        if self._hvd_config.use:
            # For multi-GPU training, we map processes to individual GPUs. TF requires
            # that for each instantiation of `tf.Session`, the process is mapped
            # to the same GPU.
            session_config = tf.compat.v1.ConfigProto()
            session_config.gpu_options.visible_device_list = str(hvd.local_rank())

        scheme = "wss" if self._env.use_tls else "ws"
        rw_coordinator_url = (
            f"{scheme}://{self._env.master_addr}:{self._env.master_port}/ws/data-layer/"
        )
        data_layer_type = self._env.experiment_config.get_data_layer_type()

        if data_layer_type == StorageTypes.SHARED_FS.value:
            local_cache_dir_path = self._env.experiment_config["data_layer"].get(
                "container_storage_path"
            )
            local_cache_path = init_container_storage_path(
                configured_storage_path=local_cache_dir_path
            )

            storage_config = storage.LFSConfigurations(storage_dir_path=str(local_cache_path))
            self._storage = storage.LFSStorage(storage_config, tensorflow_config=session_config)

        elif data_layer_type == StorageTypes.S3.value:
            local_cache_dir_path = self._env.experiment_config["data_layer"].get(
                "local_cache_container_path"
            )
            local_cache_path = init_container_storage_path(
                configured_storage_path=local_cache_dir_path
            )

            storage_config = storage.S3Configurations(
                bucket=self._env.experiment_config["data_layer"]["bucket"],
                bucket_directory_path=self._env.experiment_config["data_layer"][
                    "bucket_directory_path"
                ],
                url=rw_coordinator_url,
                local_cache_dir=str(local_cache_path),
                access_key=self._env.experiment_config["data_layer"].get("access_key"),
                secret_key=self._env.experiment_config["data_layer"].get("secret_key"),
                endpoint_url=self._env.experiment_config["data_layer"].get("endpoint_url"),
                coordinator_cert_file=self._env.master_cert_file,
                coordinator_cert_name=self._env.master_cert_name,
            )
            self._storage = storage.S3Storage(storage_config, tensorflow_config=session_config)

        elif data_layer_type == StorageTypes.GCS.value:
            local_cache_dir_path = self._env.experiment_config["data_layer"].get(
                "local_cache_container_path"
            )
            local_cache_path = init_container_storage_path(
                configured_storage_path=local_cache_dir_path
            )
            storage_config = storage.GCSConfigurations(
                bucket=self._env.experiment_config["data_layer"]["bucket"],
                bucket_directory_path=self._env.experiment_config["data_layer"][
                    "bucket_directory_path"
                ],
                url=rw_coordinator_url,
                local_cache_dir=str(local_cache_path),
                coordinator_cert_file=self._env.master_cert_file,
                coordinator_cert_name=self._env.master_cert_name,
            )
            self._storage = storage.GCSStorage(storage_config, tensorflow_config=session_config)

        else:
            raise AssertionError(
                "Please select a supported data_layer type. Supported types include: "
                f"{[i.value for i in StorageTypes]}"
            )

    def is_decorator_used(self) -> bool:
        return self._decorator_used

    def get_dataset_length(self) -> int:
        check.is_not_none(self._dataset_length, "Dataset length not yet initialized.")
        return cast(int, self._dataset_length)

    def cache_dataset(
        self,
        dataset_id: str,
        dataset_version: str,
        shuffle: bool,
        skip_shuffle_at_epoch_end: bool,
    ) -> Callable:

        # Perform lazy initialization of storage so that if users are not
        # using data layer, we are not creating unused directories.
        self._configure_storage()

        if self._training:
            # We only check the training cacheable for re-use, because for EstimatorTrial
            # it's possible that the validation cacheable is called every time validation
            # is performed.
            check.check_false(
                self._decorator_used,
                "Pleas use both `@context.experimental.cache_train_dataset(dataset_name, "
                "dataset_version)` and `@context.experimental.cache_validation_dataset("
                "dataset_name, dataset_version)` exactly once.",
            )
        self._decorator_used = True
        dataset_version += "_train" if self._training else "_val"

        def _wrap(make_dataset_fn: Callable) -> Callable:
            @functools.wraps(make_dataset_fn)
            def _decorated_fn(*args: Any, **kwargs: Any) -> Any:
                @self._storage.cacheable(  # type: ignore
                    dataset_id=dataset_id,
                    dataset_version=dataset_version,
                )
                def make_dataset() -> yogadl.DataRef:
                    return make_dataset_fn(*args, **kwargs)

                logging.info(f"Preparing dataset: {dataset_id}:{dataset_version}.")
                logging.debug(
                    f"Calling make dataset for: {dataset_id}:{dataset_version} "
                    f"with following start_offset: {self._offset}, "
                    f"shuffle: {shuffle} shuffle_seed: {self._shuffle_seed} "
                    f"shard_rank: {self._shard_rank}, world size: {self._num_shards} "
                    f"training: {self._training}."
                )

                stream_from_cache = make_dataset().stream(
                    start_offset=self._offset,
                    shuffle=shuffle,
                    skip_shuffle_at_epoch_end=skip_shuffle_at_epoch_end,
                    shuffle_seed=self._shuffle_seed,
                    shard_rank=self._shard_rank,
                    num_shards=self._num_shards,
                    drop_shard_remainder=True if self._training else False,
                )
                self._dataset_length = len(stream_from_cache)
                logging.info(f"Dataset {dataset_id}:{dataset_version} preparation finished.")

                return tensorflow.make_tf_dataset(stream_from_cache)

            return _decorated_fn

        return _wrap
