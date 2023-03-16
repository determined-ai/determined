import copy
import logging
import pathlib
import pickle
import random
import uuid
from abc import abstractmethod
from typing import Any, Dict, List, Optional, Set, Tuple, Union

from determined import searcher
from dsat import _defaults, _utils
from tensorflow.python.ops.array_ops import reverse


class DSATTrial:
    """
    Helper class for tracking the results and properties of individual Trials.
    """

    def __init__(
        self,
        hparams: Dict[str, Any],
        is_model_profiling_info_run: bool = False,
        request_id: Optional[uuid.UUID] = None,
        metric: Optional[Any] = None,
        parent: Optional["DSATTrial"] = None,
        children: Optional[Set["DSATTrial"]] = None,
        search_data: Optional[Any] = None,
        oom: bool = False,
    ) -> None:
        self.hparams = hparams
        self.is_model_profiling_info_run = is_model_profiling_info_run
        self.request_id = request_id or uuid.uuid4()
        self.metric = metric

        # Properties for lineage tracking.
        self.parent = parent
        self.children = children or set()

        # Arbitrary attribute for search-specific data tracking.
        self.search_data = search_data

        # Booleans for tracking OOM errors.
        self.oom = oom

    @property
    def ds_config(self):
        return self.hparams["ds_config"]

    @property
    def zero_stage(self):
        try:
            zero_stage = int(self.ds_config["zero_optimization"]["stage"])
        except KeyError:
            zero_stage = 0  # The DS Default. TODO: add to _defaults.py
        return zero_stage

    def add_child(self, trial: "DSATTrial") -> None:
        """Register child-parent relationship in lineage tree."""
        self.children.add(trial)
        trial.parent = self

    @property
    def lineage_root(self) -> "DSATTrial":
        if self.parent is None:
            return self
        else:
            return self.parent.lineage_root

    @property
    def lineage_set(self) -> Set["DSATTrial"]:
        """Computes set of trials in lineage tree."""
        root = self.lineage_root
        trials_set = {root}
        children = root.children
        while children:
            random_child = children.pop()
            trials_set.add(random_child)
            children |= random_child.children
        return trials_set

    @property
    def num_trials_in_lineage(self) -> int:
        """Computes total number of trials in lineage tree."""
        num_trials = len(self.lineage_set)
        return num_trials

    @property
    def oom_in_direct_history(self) -> bool:
        trial = self
        while trial is not None:
            if trial.oom:
                return True
            trial = trial.parent
        return False

    def get_state_dict(self) -> Dict[str, Any]:
        state_dict = self.__dict__
        return state_dict

    @classmethod
    def from_state_dict(cls, state_dict: Dict[str, Any]) -> "DSATTrial":
        return cls(**state_dict)


class DSATTrialTracker:
    """
    Class for organizing DSATTrial instances and retrieving pertinent info.
    """

    def __init__(
        self,
        all_trials_dict: Optional[Dict[str, DSATTrial]] = None,
        best_autotuning_metric_val: Optional[Any] = None,
        num_trials_since_best_result: int = 0,
        should_early_stop: bool = False,
    ) -> None:
        self.all_trials_dict = all_trials_dict if all_trials_dict is not None else {}

        # Altered after running autotuning trials:
        self.best_autotuning_metric_val = best_autotuning_metric_val
        self.num_trials_since_best_result = num_trials_since_best_result
        self.should_early_stop = should_early_stop

    def __getitem__(self, request_id: uuid.UUID) -> DSATTrial:
        return self.all_trials_dict[request_id]

    def create_trial(
        self,
        hparams: Dict[str, Any],
        is_model_profiling_info_run: bool = False,
        search_data: Optional[Any] = None,
        parent_trial: Optional[DSATTrial] = None,
    ) -> DSATTrial:
        """
        Creates a new `DSATTrial` object, updates lineages as appropriate, and updates the
        searcher's Trial tracking dictionary.
        """
        trial = DSATTrial(hparams=hparams, is_model_profiling_info_run=is_model_profiling_info_run)
        if search_data is not None:
            trial.search_data = search_data
        if parent_trial is not None:
            parent_trial.add_child(trial)
        self.all_trials_dict[trial.request_id] = trial
        return trial

    def get_root_trial_set(self) -> Set[DSATTrial]:
        """
        Returns the set of all non-model-profiling-info DSATTrials which were the root element
        in their lineage.
        """
        root_trial_set = {
            trial
            for trial in self.all_trials_dict.values()
            if trial.parent is None and not trial.is_model_profiling_info_run
        }
        return root_trial_set

    def get_closed_trials_dict(
        self, searcher_state: searcher.SearcherState, zero_stage: Optional[int] = None
    ) -> Dict[str, DSATTrial]:
        closed_request_ids = searcher_state.trials_closed
        closed_trials_dict = self.get_trials_dict_from_request_id_set(
            closed_request_ids, zero_stage
        )
        return closed_trials_dict

    def get_failed_trials_dict(
        self, searcher_state: searcher.SearcherState, zero_stage: Optional[int]
    ) -> Dict[str, DSATTrial]:
        failed_request_ids = searcher_state.failures
        failed_trials_dict = self.get_trials_dict_from_request_id_set(
            failed_request_ids, zero_stage
        )
        return failed_trials_dict

    def get_running_trials_dict(
        self, searcher_state: searcher.SearcherState, zero_stage: Optional[int]
    ) -> Dict[str, DSATTrial]:
        running_request_ids = searcher_state.trials_created - (
            searcher_state.trials_closed | searcher_state.failures
        )
        running_trials_dict = self.get_trials_dict_from_request_id_set(
            running_request_ids, zero_stage
        )
        return running_trials_dict

    def get_trials_dict_from_request_id_set(
        self, request_id_set: Set[int], zero_stage: Optional[int]
    ):
        for r_id in request_id_set:
            trial = self.all_trials_dict[r_id]
            if zero_stage is None or trial.zero_stage == zero_stage:
                request_id_set[r_id] = trial

    def get_ops_list_from_trial(self, trial: DSATTrial, length: int) -> List[searcher.Operation]:
        create_op = searcher.Create(
            request_id=trial.request_id,
            hparams=trial.hparams,
            checkpoint=None,
        )
        validate_after_op = searcher.ValidateAfter(request_id=trial.request_id, length=length)
        ops = [create_op, validate_after_op]
        return ops

    def update_best_trial_info(
        self,
        last_trial: DSATTrial,
        searcher_metric_name: str,
        smaller_is_better: bool,
        metric: Optional[Dict[str, Any]] = None,
    ) -> None:
        if metric is None or last_trial.oom:
            # TODO: Curently not counting errors and ooms against num_trials_since_best_result
            # because otherwise early Trials can just all OOM, early_stopping is triggered, and no
            # non-trivial results are returned. Should discuss, though.
            return
        else:
            searcher_metric_value = metric[searcher_metric_name]
            last_trial_is_best = self.best_autotuning_metric_val is None or (
                searcher_metric_value < self.best_autotuning_metric_val
                if smaller_is_better
                else searcher_metric_value > self.best_autotuning_metric_val
            )
            if last_trial_is_best:
                self.best_autotuning_metric_val = searcher_metric_value
                self.num_trials_since_best_result = 0
            else:
                self.num_trials_since_best_result += 1

    def get_state_dict(self) -> Dict[uuid.UUID, Any]:
        state_dict = self.__dict__
        return state_dict

    @classmethod
    def from_state_dict(cls, state_dict: Dict[uuid.UUID, Any]) -> "DSATTrialTracker":
        trial_tracker = cls(**state_dict)
        return trial_tracker


class DSATModelProfilingInfo:
    """
    Helper class for processing the model profiling info run.
    """

    def __init__(
        self,
        request_id: uuid.UUID,
        model_profiling_info_results: Dict[str, Any],
        slots: int,
        fp16: bool,
        mp_size: int,
    ) -> None:
        self.request_id = request_id
        self.model_profiling_info_results = model_profiling_info_results
        self.slots = slots
        self.fp16 = fp16
        self.mp_size = mp_size

        logging.info(
            f"Appprox. max mbs per stage: {self.max_mbs_per_stage}"
        )  # TODO: remove after testing.
        logging.info(f"Approx. GPU memory per stage: {self.mem_per_gpu_per_stage}")  # TODO: remove.
        logging.info(f"Total GPU memory: {self.gpu_mem}")
        logging.info(f"Viable zero stages: {self.viable_zero_stages}")

    @property
    def gpu_mem(self) -> int:
        return self.model_profiling_info_results["gpu_mem"]

    @property
    def num_params(self) -> int:
        return self.model_profiling_info_results["num_params"]

    @property
    def trainable_num_params(self) -> int:
        return self.model_profiling_info_results["trainable_num_params"]

    @property
    def activation_mem_per_gpu(self) -> int:
        return self.model_profiling_info_results["activation_mem_per_gpu"]

    @property
    def mem_per_gpu_per_stage(self) -> Dict[int, int]:
        """
        Returns the required gpu memory in bytes, per stage.
        """
        params_mem = self.num_params * (2 if self.fp16 else 4)
        gradients_mem = self.trainable_num_params * (2 if self.fp16 else 4)
        # optimizer_mem assumes Adam like DS. TODO: don't assume this.
        optimizer_mem = self.trainable_num_params * (16 if self.fp16 else 8)

        non_activation_mem_per_gpu_per_stage = {
            0: params_mem + gradients_mem + optimizer_mem,
            1: params_mem + gradients_mem + optimizer_mem // self.slots,
            2: params_mem + (gradients_mem + optimizer_mem) // self.slots,
            3: (params_mem + gradients_mem + optimizer_mem) // self.slots,
        }
        if self.mp_size > 1:
            non_activation_mem_per_gpu_per_stage = {
                stage: mem // self.mp_size
                for stage, mem in non_activation_mem_per_gpu_per_stage.items()
            }
        # TODO: Following DS here and not dividing activation memory by mp_size, but seems like
        # you should?
        mem_per_gpu_per_stage = {
            stage: mem + self.activation_mem_per_gpu
            for stage, mem in non_activation_mem_per_gpu_per_stage.items()
        }
        return mem_per_gpu_per_stage

    @property
    def viable_zero_stages(self) -> Set[int]:
        """
        Returns the set of viable zero stages based on a rough computation.
        """
        # TODO: account for model parallelism. Add a fudge factor for a little leeway?
        viable_stages = {
            stage for stage, mem in self.mem_per_gpu_per_stage.items() if mem < self.gpu_mem
        }
        return viable_stages

    @property
    def max_mbs_per_stage(self) -> Dict[int, int]:
        """
        Returns the approximate max train_micro_batch_size_per_gpu (mbs) per stage.
        """
        max_mbs_per_stage = {
            stage: (self.gpu_mem - mem) // self.activation_mem_per_gpu
            for stage, mem in self.mem_per_gpu_per_stage.items()
            if stage in self.viable_zero_stages
        }
        return max_mbs_per_stage

    def get_state_dict(self) -> Dict[str, Any]:
        state_dict = self.__dict__
        return state_dict

    @classmethod
    def from_state_dict(cls, state_dict: Dict[str, Any]) -> "DSATModelProfilingInfo":
        return cls(**state_dict)


class DSATSearchMethodBase(searcher.SearchMethod):
    """
    Base searcher class implementing common methods.
    """

    def __init__(
        self,
        submitted_config_dict: Dict[str, Any],
    ) -> None:
        self.submitted_config_dict = submitted_config_dict
        self.searcher_metric_name = self.submitted_config_dict["searcher"]["metric"]
        self.smaller_is_better = self.submitted_config_dict["searcher"].get(
            "smaller_is_better", _defaults.SMALLER_IS_BETTER
        )
        self.submitted_hps = self.submitted_config_dict["hyperparameters"]
        self.ds_config = self.submitted_hps["ds_config"]
        # Merge the submitted autotuning section with the DS _defaults.
        self.autotuning_config = {**_defaults.AUTUTONING_DICT, **self.ds_config["autotuning"]}
        self.tuner_num_trials = self.autotuning_config["tuner_num_trials"]
        self.num_tuning_micro_batch_sizes = self.autotuning_config["num_tuning_micro_batch_sizes"]
        self.tuner_early_stopping = self.autotuning_config["num_tuning_micro_batch_sizes"]

        self.trial_tracker = DSATTrialTracker()

        # Non-trivial values instantiated after model profiling run
        self.model_profile_info = None

    @abstractmethod
    def get_new_searcher_ops_list(
        self,
        searcher_state: searcher.SearcherState,
        request_id: uuid.UUID,
        last_trial: DSATTrial,
        metric: Optional[Union[float, Dict[str, Any]]] = None,
    ) -> List[searcher.Operation]:
        """Generates a list of new operations to run."""
        pass

    def initial_operations(
        self, searcher_state: searcher.SearcherState
    ) -> List[searcher.Operation]:
        """
        Submits the model info profiling run in order to collect model and resources info to
        inform the search.
        """
        model_profile_info_hps = copy.deepcopy(self.submitted_hps)
        _utils.replace_dict_in_place(
            model_profile_info_hps["ds_config"],
            _defaults.MODEL_INFO_PROFILING_DS_CONFIG,
        )
        model_profile_info_trial = self.trial_tracker.create_trial(
            hparams=model_profile_info_hps, is_model_profiling_info_run=True
        )
        # Only a single step is required for the model profiling run.
        ops = self.trial_tracker.get_ops_list_from_trial(trial=model_profile_info_trial, length=1)
        return ops

    def on_trial_created(
        self, searcher_state: searcher.SearcherState, request_id: uuid.UUID
    ) -> List[searcher.Operation]:
        return []

    def on_validation_completed(
        self,
        searcher_state: searcher.SearcherState,
        request_id: uuid.UUID,
        metric: Union[float, Dict[str, Any]],
        train_length: int,
    ) -> List[searcher.Operation]:
        last_trial = self.trial_tracker[request_id]
        last_trial.oom = "OOM" in metric

        if last_trial.is_model_profiling_info_run:
            slots = self.submitted_config_dict["resources"]["slots_per_trial"]
            if "fp16" in self.ds_config:
                fp16 = self.ds_config["fp16"]["enabled"]
            else:
                fp16 = False
            mp_size = self.autotuning_config["mp_size"]
            self.model_profile_info = DSATModelProfilingInfo(
                request_id=request_id,
                model_profiling_info_results=metric,
                slots=slots,
                fp16=fp16,
                mp_size=mp_size,
            )
        else:
            last_trial.metric = metric
            self.trial_tracker.update_best_trial_info(
                last_trial=last_trial,
                searcher_metric_name=self.searcher_metric_name,
                smaller_is_better=self.smaller_is_better,
                metric=metric,
            )

        # All DS AT Trials should be closed upon completion.
        ops = [searcher.Close(request_id=request_id)]

        self.update_should_early_stop()
        new_ops_list = self.get_new_searcher_ops_list(
            searcher_state=searcher_state,
            request_id=request_id,
            metric=metric,
            last_trial=last_trial,
        )
        ops.extend(new_ops_list)
        return ops

    def on_trial_closed(
        self, searcher_state: searcher.SearcherState, request_id: uuid.UUID
    ) -> List[searcher.Operation]:
        # Using searcher_state.trials_created led to intermittent errors where the MIP trial seemed
        # to get registered as closed before its follow on trials were created.
        running_trial_ids = (
            set(self.trial_tracker.all_trials_dict)
            - searcher_state.trials_closed
            - searcher_state.failures
        )
        if not running_trial_ids:
            logging.info("**** Shutting down DeepSpeed Autotune: No Remaining Trials ****")
            return [searcher.Shutdown()]
        return []

    def on_trial_exited_early(
        self,
        searcher_state: searcher.SearcherState,
        request_id: uuid.UUID,
        exited_reason: searcher.ExitedReason,
    ) -> List[searcher.Operation]:
        # TODO: some early exits are actually OOMs which take a path other than the standard
        # RuntimeError. Handle theses cases.
        last_trial = self.trial_tracker[request_id]

        if last_trial.is_model_profiling_info_run:
            logging.info(
                "**** Shutting down DeepSpeed Autotune: Error in Model Profiling Info Trial ****"
            )
            return [searcher.Shutdown()]
        if exited_reason == searcher.ExitedReason.ERRORED:
            # Assuming Trial errored due to uncaught OOM, e.g. from model initialization.
            # TODO: can we be less ham-fisted?
            last_trial.oom = True
            self.trial_tracker.update_best_trial_info(
                last_trial=last_trial,
                searcher_metric_name=self.searcher_metric_name,
                smaller_is_better=self.smaller_is_better,
                metric=None,
            )

            self.update_should_early_stop()
            new_ops_list = self.get_new_searcher_ops_list(
                searcher_state=searcher_state,
                request_id=request_id,
                metric=None,
                last_trial=last_trial,
            )

            return new_ops_list

    def progress(self, searcher_state: searcher.SearcherState) -> float:
        progress = len(searcher_state.trials_closed) / self.tuner_num_trials
        return progress

    def save_method_state(self, path: pathlib.Path) -> None:
        if self.model_profile_info is None:
            model_profile_info = None
        else:
            model_profile_info = self.model_profile_info.get_state_dict()
        state_dict = {
            "trial_tracker": self.trial_tracker.get_state_dict(),
            "model_profile_info": model_profile_info,
        }
        checkpoint_path = path.joinpath("state_dict.pkl")
        with checkpoint_path.open("wb") as f:
            pickle.dump(state_dict, f)

    def load_method_state(self, path: pathlib.Path) -> None:
        logging.info(f"Restoring searcher state from checkpoint.")
        checkpoint_path = path.joinpath("state_dict.pkl")
        with checkpoint_path.open("rb") as f:
            state_dict = pickle.load(f)
            self.trial_tracker = DSATTrialTracker.from_state_dict(state_dict["trial_tracker"])
            model_profile_info = state_dict["model_profile_info"]
            if model_profile_info is None:
                self.model_profile_info is None
            else:
                self.model_profile_info = DSATModelProfilingInfo.from_state_dict(model_profile_info)

    def update_should_early_stop(self) -> None:
        """Updates the DSATTrialTracker's should_early_stop attribute."""
        self.trial_tracker.should_early_stop = (
            self.trial_tracker.should_early_stop
            or self.trial_tracker.num_trials_since_best_result == self.tuner_early_stopping
        )


class DSATRandomSearchMethod(DSATSearchMethodBase):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        # TODO: get desired zero stages from config. Currently just running all viable.

    def get_new_searcher_ops_list(
        self,
        searcher_state: searcher.SearcherState,
        request_id: uuid.UUID,
        last_trial: DSATTrial,
        metric: Optional[Union[float, Dict[str, Any]]] = None,
    ) -> List[searcher.Operation]:
        if self.trial_tracker.should_early_stop:
            new_ops_list = []
            logging.info("Early stopping criteria met, no new Trials will be submitted.")
        elif last_trial.is_model_profiling_info_run:
            new_ops_list = self.get_ops_list_after_model_profiling_info_run()
        elif len(searcher_state.trials_created) < self.tuner_num_trials:
            new_ops_list = self.get_ops_list_after_autotuning_run(last_trial)
        else:
            new_ops_list = []
        return new_ops_list

    def get_ops_list_after_model_profiling_info_run(
        self,
    ) -> List[searcher.Operation]:
        approx_num_lineages = self.tuner_num_trials // self.num_tuning_micro_batch_sizes
        new_ops_list = []
        for _ in range(approx_num_lineages):
            hparams, search_data = self.get_random_hparams_and_search_data()
            new_trial = self.trial_tracker.create_trial(
                hparams=hparams, search_data=search_data, parent_trial=None
            )
            # A +1 is required to align DS step/DET max_length conventions.
            end_profile_step = self.autotuning_config["end_profile_step"] + 1
            new_ops = self.trial_tracker.get_ops_list_from_trial(
                trial=new_trial, length=end_profile_step
            )
            new_ops_list.extend(new_ops)
        return new_ops_list

    def get_ops_list_after_autotuning_run(
        self,
        last_trial: DSATTrial,
    ) -> List[searcher.Operation]:
        if last_trial.num_trials_in_lineage < self.num_tuning_micro_batch_sizes:
            hparams, search_data = self.get_hparams_and_search_data_from_last_trial(
                last_trial=last_trial,
            )

            # TODO: remove print tests.
            logging.info("**************** BSZ History ****************")
            bsz_history = [hparams["ds_config"]["train_micro_batch_size_per_gpu"]]
            print_trial = last_trial
            while print_trial is not None:
                bsz = print_trial.ds_config["train_micro_batch_size_per_gpu"]
                bsz_history.append(bsz)
                print_trial = print_trial.parent
            logging.info(f"History: {str(list(reversed(bsz_history)))}")
            logging.info(f'ds_config for lineage: {hparams["ds_config"]}')
            logging.info("**************** BSZ History End ****************")

            parent_trial = last_trial
        else:
            hparams, search_data = self.get_random_hparams_and_search_data()
            parent_trial = None
        if hparams is None:
            new_ops_list = []
        else:
            new_trial = self.trial_tracker.create_trial(
                hparams=hparams, search_data=search_data, parent_trial=parent_trial
            )
            # A +1 is required to align DS step/DET max_length conventions.
            end_profile_step = self.autotuning_config["end_profile_step"] + 1
            new_ops_list = self.trial_tracker.get_ops_list_from_trial(
                trial=new_trial, length=end_profile_step
            )
        return new_ops_list

    def get_hparams_and_search_data_from_last_trial(
        self,
        last_trial: DSATTrial,
    ) -> Tuple[Optional[Dict[str, Any]], Dict[str, Any]]:
        """
        Perform a slightly modified binary search on the train_micro_batch_size_per_gpu.
        """
        # TODO: verify we are always quitting when no more non-trivial trials are possible.

        lo, hi = last_trial.search_data["lo"], last_trial.search_data["hi"]
        mid = (lo + hi) // 2
        # TODO: edge cases and +- 1 error checks.
        if last_trial.oom:
            hi = mid - 1
        else:
            lo = mid + 1
            hi = (
                hi if last_trial.oom_in_direct_history else int(1.05 * hi)
            )  # TODO: let user configure ceiling factor. Current number is just a guess, and maybe
            # what native DS AT does?
        new_mid = (lo + hi) // 2
        if new_mid == lo:
            new_hparams = None
        else:
            new_hparams = copy.deepcopy(last_trial.hparams)
            new_hparams["ds_config"]["train_micro_batch_size_per_gpu"] = new_mid
        return new_hparams, {"lo": lo, "hi": hi}

    def get_random_hparams_and_search_data(self) -> Tuple[Dict[str, Any], Dict[str, Any]]:
        # TODO: verify that we are not repeating a previously attempted config.
        random_zero_stage = random.choice(tuple(self.model_profile_info.viable_zero_stages))
        new_hparams = copy.deepcopy(self.submitted_hps)
        zero_optim_config = _utils.get_random_zero_optim_dict_for_zero_stage(random_zero_stage)
        _utils.replace_dict_in_place(
            new_hparams["ds_config"], {"zero_optimization": zero_optim_config}
        )
        random_zero_stage_max_mbs = self.model_profile_info.max_mbs_per_stage[random_zero_stage]
        lo, hi = 1, 2 * random_zero_stage_max_mbs - 1
        mid = (lo + hi) // 2
        search_data = {
            "lo": lo,
            "hi": hi,
        }
        new_hparams["ds_config"]["train_micro_batch_size_per_gpu"] = mid
        return (new_hparams, search_data)
