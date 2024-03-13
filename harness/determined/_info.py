import json
import os
from typing import Any, Dict, Iterable, List, Optional, Union

from determined import gpu
from determined.common import api

DEFAULT_RENDEZVOUS_INFO_PATH = "/run/determined/info/rendezvous.json"
DEFAULT_TRIAL_INFO_PATH = "/run/determined/info/trial.json"
DEFAULT_RESOURCES_INFO_PATH = "/run/determined/info/resources.json"
DEFAULT_CLUSTER_INFO_PATH = "/run/determined/info/cluster.json"


def getenv_int(key: str) -> Optional[int]:
    val = os.environ.get(key)
    return val if val is None else int(val)


def as_dict(obj: Any, skip: Iterable[str] = ()) -> Dict[str, Any]:
    """Remove the leading underscore from private variables for a json representation."""
    return {k.lstrip("_"): v for k, v in vars(obj).items() if k not in skip}


class RendezvousInfo:
    """
    RendezvousInfo is machine identity information that is:
     - configured in the container by the rendezvous layer (when on a Determined cluster)
     - independent of the launch layer
     - consumed by the launch layer
    """

    def __init__(
        self,
        container_addrs: List[str],
        container_rank: int,
        container_slot_counts: List[int],
    ):
        self.container_addrs = container_addrs
        self.container_rank = container_rank
        self.container_slot_counts = container_slot_counts

    def _to_file(self, path: str = DEFAULT_RENDEZVOUS_INFO_PATH) -> None:
        """
        to_file writes the RendezvousInfo to a well-known location in a Determined container.  This
        is called automatically early in the lifetime of a container, so user code can always expect
        this file to be written before user code runs.
        """
        with open(path, "w") as f:
            json.dump(vars(self), f)

    @classmethod
    def _from_file(
        cls,
        path: str = DEFAULT_RENDEZVOUS_INFO_PATH,
    ) -> Optional["RendezvousInfo"]:
        if not os.path.exists(path):
            return None
        with open(path) as f:
            return cls(**json.load(f))


class TrialInfo:
    def __init__(
        self,
        trial_id: int,
        experiment_id: int,
        trial_seed: int,
        hparams: Dict[str, Any],
        config: Dict[str, Any],
        steps_completed: int,
        trial_run_id: int,
        debug: bool,
        inter_node_network_interface: Optional[str],
    ):
        """
        TrialInfo contains information about the trial that is currently running.
        """

        #: The Trial ID for the current task.
        self.trial_id = trial_id

        #: The Experiment ID for the current task.
        self.experiment_id = experiment_id

        #: The random seed for the current Trial.
        self.trial_seed = trial_seed

        #: The hyperparameter values selected for the current Trial.
        self.hparams = hparams

        # _config is private because it's not a stable API; as the experiment config version
        # increases, the shape of the experiment config dict might change as the master is upgraded.
        # As a result, we should strongly discourage users from reading the experiment config dict
        # in their training code.  They should focus on the data field and the hparams field, which
        # have always been and will always be user-defined.
        self._config = config

        # rb: These fields are private because I am pretty confident that we need to find better
        # ways to pass them around the system.  But for now, they're passed in as environment
        # variables and for now we have to be able to handle that.
        # TODO: decide if we really want to track steps_completed for users or not.
        self._steps_completed = steps_completed
        # TODO: get rid of trial_run_id in favor of allocation id.
        self._trial_run_id = trial_run_id
        # TODO: decide if the experiment config is the right place for users to set a debug flag.
        self._debug = debug
        # TODO: Get rid of this in favor of launch layer configs?
        self._inter_node_network_interface = inter_node_network_interface

    @classmethod
    def _from_env(cls) -> "TrialInfo":
        assert "DET_TRIAL_ID" in os.environ, "must be run inside a Trial container"
        experiment_config = json.loads(os.environ["DET_EXPERIMENT_CONFIG"])
        return cls(
            trial_id=int(os.environ["DET_TRIAL_ID"]),
            experiment_id=int(os.environ["DET_EXPERIMENT_ID"]),
            trial_seed=int(os.environ["DET_TRIAL_SEED"]),
            hparams=json.loads(os.environ["DET_HPARAMS"]),
            config=experiment_config,
            steps_completed=int(os.environ["DET_STEPS_COMPLETED"]),
            trial_run_id=int(os.environ["DET_TRIAL_RUN_ID"]),
            debug=experiment_config.get("debug", False),
            inter_node_network_interface=os.environ.get("DET_INTER_NODE_NETWORK_INTERFACE"),
        )

    def _to_file(self, path: str = DEFAULT_TRIAL_INFO_PATH) -> None:
        with open(path, "w") as f:
            json.dump(as_dict(self), f)

    @classmethod
    def _from_file(cls, path: str = DEFAULT_TRIAL_INFO_PATH) -> Optional["TrialInfo"]:
        if not os.path.exists(path):
            return None
        with open(path) as f:
            return cls(**json.load(f))


class ResourcesInfo:
    def __init__(self, gpu_uuids: List[str]) -> None:
        self._gpu_uuids = gpu_uuids

    @property
    def gpu_uuids(self) -> List[str]:
        return self._gpu_uuids

    @classmethod
    def _by_inspection(cls) -> "ResourcesInfo":
        return cls(gpu_uuids=gpu.get_gpu_uuids())

    def _to_file(self, path: str = DEFAULT_RESOURCES_INFO_PATH) -> None:
        with open(path, "w") as f:
            json.dump(as_dict(self), f)

    @classmethod
    def _from_file(cls, path: str = DEFAULT_RESOURCES_INFO_PATH) -> Optional["ResourcesInfo"]:
        if not os.path.exists(path):
            return None
        with open(path) as f:
            return cls(**json.load(f))


class ClusterInfo:
    """
    ClusterInfo exposes various properties that are set for tasks while running on the cluster.

    Examples:

    .. code:: python

        info = det.get_cluster_info()
        assert info is not None, "this code only runs on-cluster!"

        print("master_url", info.master_url)
        print("task_id", info.task_id)
        print("allocation_id", info.allocation_id)
        print("session_token", info.session_token)

        print("container_addrs", info.container_addrs)
        print("container_rank", info.container_rank)

        if info.task_type == "TRIAL":
            print("trial.id", info.trial.trial_id)
            print("trial.hparams", info.trial.hparams)

    .. warning::

       Be careful with this object!  If you depend on a ClusterInfo object during training for
       anything more than e.g.  informational logging, you run the risk of making your training code
       unable to run outside of Determined.  ClusterInfo is meant to be most useful to custom launch
       layers, which likely are not able to run outside of Determined anyway.
    """

    def __init__(
        self,
        master_url: str,
        cluster_id: str,
        agent_id: str,
        slot_ids: List[int],
        task_id: str,
        allocation_id: str,
        session_token: str,
        task_type: str,
        # Optional information from the master:
        master_cert_name: Optional[str] = None,
        master_cert_file: Optional[str] = None,
        latest_checkpoint: Optional[str] = None,
        # Information which is generated within a container at runtime.
        trial_info: Optional[TrialInfo] = None,
        rendezvous_info: Optional[RendezvousInfo] = None,
        resources_info: Optional[ResourcesInfo] = None,
    ):
        #: The url for reaching the master.
        self.master_url = api.canonicalize_master_url(master_url)

        #: The unique identifier for this cluster.
        self.cluster_id = cluster_id

        #: The identifier of the Determined agent this container is running on.
        self.agent_id = agent_id

        #: The slot ids assigned to this container.
        self.slot_ids = slot_ids

        #: The unique identifier for the current task.
        self.task_id = task_id

        #: The unique identifier for the current allocation.
        self.allocation_id = allocation_id

        #: The Determined login session token created for the current task.
        self.session_token = session_token

        #: The type of task.  Currently one of the following string literals:
        #:    - ``"TRIAL"``
        #:    - ``"NOTEBOOK"``
        #:    - ``"SHELL"``
        #:    - ``"COMMAND"``
        #:    - ``"TENSORBOARD"``
        #:    - ``"CHECKPOINT_GC"``
        #:
        #: Additional values may be added in the future.
        self.task_type = task_type

        #: The name on the master certificate, when using TLS.
        self.master_cert_name = master_cert_name

        #: The file location for the master certificate, if present, or "noverify" if it has been
        #: configured not to verify the master cert.
        self.master_cert_file = master_cert_file

        self._latest_checkpoint = latest_checkpoint

        self._trial_info = trial_info
        self._rendezvous_info = rendezvous_info
        self._resources_info = resources_info

    @classmethod
    def _from_env(cls) -> "ClusterInfo":
        required = [
            "DET_MASTER",
            "DET_CLUSTER_ID",
            "DET_AGENT_ID",
            "DET_SLOT_IDS",
            "DET_TASK_ID",
            "DET_ALLOCATION_ID",
            "DET_SESSION_TOKEN",
            "DET_TASK_TYPE",
        ]
        missing = [r for r in required if r not in os.environ]
        if missing:
            raise RuntimeError(
                f'missing environment keys [{", ".join(missing)}], is this running on-cluster?'
            )
        return cls(
            master_url=os.environ["DET_MASTER"],
            cluster_id=os.environ["DET_CLUSTER_ID"],
            agent_id=os.environ["DET_AGENT_ID"],
            slot_ids=json.loads(os.environ["DET_SLOT_IDS"]),
            task_id=os.environ["DET_TASK_ID"],
            allocation_id=os.environ["DET_ALLOCATION_ID"],
            session_token=os.environ["DET_SESSION_TOKEN"],
            task_type=os.environ["DET_TASK_TYPE"],
            # Optional info:
            master_cert_name=os.environ.get("DET_MASTER_CERT_NAME"),
            master_cert_file=os.environ.get("DET_MASTER_CERT_FILE"),
            latest_checkpoint=os.environ.get("DET_LATEST_CHECKPOINT"),
            # Separate info objects:
            trial_info=TrialInfo._from_file(),
            rendezvous_info=RendezvousInfo._from_file(),
            resources_info=ResourcesInfo._from_file(),
        )

    def _to_file(self, path: str = DEFAULT_CLUSTER_INFO_PATH) -> None:
        skip = ("_trial_info", "_rendezvous_info", "_resources_info")
        with open(path, "w") as f:
            json.dump(as_dict(self, skip), f)

    @classmethod
    def _from_file(cls, path: str = DEFAULT_CLUSTER_INFO_PATH) -> Optional["ClusterInfo"]:
        if not os.path.exists(path):
            return None
        with open(path) as f:
            return cls(
                trial_info=TrialInfo._from_file(),
                rendezvous_info=RendezvousInfo._from_file(),
                resources_info=ResourcesInfo._from_file(),
                **json.load(f),
            )

    @property
    def latest_checkpoint(self) -> Optional[str]:
        """
        The checkpoint ID of the most recent checkpoint that should be loaded.

        Since non-trial-type tasks cannot currently save checkpoints, ``.latest_checkpoint`` is
        currently always None for non-trial-type tasks.
        """
        if self.task_type != "TRIAL":
            return None
        return self._latest_checkpoint

    @property
    def user_data(self) -> Dict[str, Any]:
        """
        The content of the ``data`` field of the experiment configuration.

        Since other types of configuration files don't allow a ``data`` field, accessing
        ``user_data`` from non-trial-type tasks will always return an empty dictionary.
        """
        if self.task_type != "TRIAL":
            return {}
        return self.trial._config.get("data", {})

    @property
    def trial(self) -> TrialInfo:
        """
        The :class:`~determined.TrialInfo` sub-info object for the current trial task.

        Attempting to read ``.trial`` in a non-trial task type will raise a RuntimeError.
        """
        if self.task_type != "TRIAL":
            raise RuntimeError(
                f'you cannot use the .trial property when .task_type ("{self.task_type}") != '
                '"TRIAL"'
            )
        assert self._trial_info is not None
        return self._trial_info

    @property
    def container_addrs(self) -> List[str]:
        """A list of addresses for all containers in the allocation, ordered by rank."""
        if self.task_type != "TRIAL":
            # Presently, only trials are allowed to use the rendezvous API.
            # But also, only trials are scheduled across multiple nodes, so we can cheat here.
            return ["127.0.0.1"]
        assert self._rendezvous_info is not None
        return self._rendezvous_info.container_addrs

    @property
    def container_slot_counts(self) -> List[int]:
        """A list of slots for all containers in the allocation, ordered by rank."""
        if self.task_type != "TRIAL":
            # Presently, only trials are allowed to use the rendezvous API.
            # But also, only trials are scheduled across multiple nodes, so we can cheat here.
            return [len(self.slot_ids)]
        assert self._rendezvous_info is not None
        return self._rendezvous_info.container_slot_counts

    @property
    def container_rank(self) -> int:
        """
        The rank assigned to this container.

        When using a distributed training framework, the framework may choose a different rank for
        this container.
        """
        if self.task_type != "TRIAL":
            # Presently, only trials are allowed to use the rendezvous API.
            # But also, only trials are scheduled across multiple nodes, so we can cheat here.
            return 0
        assert self._rendezvous_info is not None
        return self._rendezvous_info.container_rank

    @property
    def gpu_uuids(self) -> List[str]:
        """The UUIDs to the gpus assigned to this container."""
        assert self._resources_info is not None
        return self._resources_info.gpu_uuids


_info = "unloaded"  # type: Union[ClusterInfo, str, None]


def get_cluster_info() -> Optional[ClusterInfo]:
    """
    Returns either the :class:`~determined.ClusterInfo` object for the current task, or ``None`` if
    not running in a task.
    """

    global _info
    if isinstance(_info, str):
        _info = ClusterInfo._from_file()
    return _info
