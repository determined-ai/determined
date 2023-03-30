import copy
import logging
import pathlib
import pickle
import random
import uuid
from abc import abstractmethod
from typing import Any, Dict, List, Optional, Set, Tuple, Union

from determined import searcher
from determined.pytorch.deepspeed import get_ds_config_from_hparams
from determined.pytorch.deepspeed.dsat import _defaults, _utils
from determined.util import merge_dicts


class DSATTrial:
    """
    Helper class for tracking the results and properties of individual Trials.
    """

    def __init__(
        self,
        hparams: Dict[str, Any],
        model_dir: str,
        is_model_profiling_info_run: bool = False,
        request_id: Optional[uuid.UUID] = None,
        metric: Optional[Dict[str, Any]] = None,
        parent: Optional["DSATTrial"] = None,
        children: Optional[Set["DSATTrial"]] = None,
        search_data: Optional[Any] = None,
        error: bool = False,
    ) -> None:
        self.hparams = hparams
        self.model_dir = model_dir
        self.is_model_profiling_info_run = is_model_profiling_info_run
        self.request_id = request_id or uuid.uuid4()
        self.metric = metric if metric is not None else {}

        # Properties for lineage tracking.
        self.parent = parent
        self.children = children or set()

        # Arbitrary attribute for search-specific data tracking.
        self.search_data = search_data

        # Booleans for tracking whether the Trial errored.
        self.error = error

    @property
    def ds_config(self):
        return get_ds_config_from_hparams(self.hparams, self.model_dir)

    @property
    def zero_stage(self):
        return int(self.ds_config.get("zero_optimization", {}).get("stage", _defaults.ZERO_STAGE))

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
    def error_in_direct_history(self) -> bool:
        trial = self
        while trial is not None:
            if trial.error:
                return True
            trial = trial.parent
        return False

    @property
    def mbs_in_lineage(self) -> Set["DSATTrial"]:
        """
        Returns the set of all `train_micro_batch_size_per_gpu` (mbs) used in the Trial's lineage.
        """
        lineage_set = self.lineage_set
        mbs_in_lineage = {t.ds_config["train_micro_batch_size_per_gpu"] for t in lineage_set}
        return mbs_in_lineage

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
        slots: int,
        model_dir: str,
        autotuning_config: Dict[str, Any],
        smaller_is_better: bool,
        searcher_metric_name: str,
        all_trials_dict: Optional[Dict[uuid.UUID, DSATTrial]] = None,
        best_autotuning_metric_val: Optional[Any] = None,
        num_trials_since_best_result: int = 0,
        should_stop: bool = False,
    ) -> None:
        self.slots = slots
        self.model_dir = model_dir
        self.autotuning_config = autotuning_config
        self.smaller_is_better = smaller_is_better
        self.searcher_metric_name = searcher_metric_name

        self.all_trials_dict = all_trials_dict if all_trials_dict is not None else {}

        # Altered after running autotuning trials:
        self.best_autotuning_metric_val = best_autotuning_metric_val
        self.num_trials_since_best_result = num_trials_since_best_result
        self.should_stop = should_stop

    @property
    def tuner_num_trials(self) -> int:
        return self.autotuning_config["tuner_num_trials"]

    @property
    def tuner_early_stopping(self) -> int:
        return self.autotuning_config["tuner_early_stopping"]

    @property
    def successful_trials_dict(self) -> Dict[uuid.UUID, DSATTrial]:
        return {u: t for u, t in self.all_trials_dict.items() if t.metric and not t.error}

    @property
    def errored_trials_dict(self) -> Dict[uuid.UUID, DSATTrial]:
        return {u: t for u, t in self.all_trials_dict.items() if t.error}

    @property
    def running_trials_dict(self) -> Dict[uuid.UUID, DSATTrial]:
        # `DSATTrial` instances are initialized with trivial metrics and `errror` False. After
        # either successfully completing or early exiting, at least one of these fields will be
        # non-trivially populated.
        return {u: t for u, t in self.all_trials_dict.items() if not t.metric and not t.error}

    def __len__(self) -> int:
        return len(self.all_trials_dict)

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
        # Create a consistent batch size configuration which obeys the DS constraints.
        ds_config = get_ds_config_from_hparams(hparams, self.model_dir)
        batch_size_config = _utils.get_batch_config_from_mbs_gas_and_slots(
            ds_config, slots=self.slots
        )
        # Key to enable dsat code path for Trial classes.
        hparams[_defaults.USE_DSAT_MODE_KEY] = True
        hparams[_defaults.OVERWRITE_KEY] = {
            **hparams.get(
                _defaults.OVERWRITE_KEY,
            ),
            **batch_size_config,
        }

        trial = DSATTrial(
            hparams=hparams,
            model_dir=self.model_dir,
            is_model_profiling_info_run=is_model_profiling_info_run,
        )
        self.all_trials_dict[trial.request_id] = trial
        self.update_should_stop()
        # TODO: Delete print test.
        logging.info(f"=============Total Trials Created: {len(self)}=============")
        if search_data is not None:
            trial.search_data = search_data
        if parent_trial is not None:
            parent_trial.add_child(trial)
        return trial

    def get_root_trial_set(
        self, include_model_profiling_info_trial: bool = False
    ) -> Set[DSATTrial]:
        """
        Returns the set of all non-model-profiling-info DSATTrials which were the root element
        in their lineage.
        """
        root_trial_set = set()
        for trial in self.all_trials_dict.values():
            if trial.parent is None:
                if trial.is_model_profiling_info_run and not include_model_profiling_info_trial:
                    continue
                root_trial_set.add(trial)
        return root_trial_set

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
    ) -> None:
        searcher_metric_value = last_trial.metric.get(self.searcher_metric_name)
        # TODO: Curently not counting explicit OOMs or other errors (which are sometimes also
        # opaque OOMs) against num_trials_since_best_result
        # because otherwise early Trials can just all OOM, early stopping is triggered, and no
        # non-trivial results are returned. Should discuss, though.
        if searcher_metric_value is not None:
            last_trial_is_best = self.best_autotuning_metric_val is None or (
                searcher_metric_value < self.best_autotuning_metric_val
                if self.smaller_is_better
                else searcher_metric_value > self.best_autotuning_metric_val
            )
            if last_trial_is_best:
                self.best_autotuning_metric_val = searcher_metric_value
                self.num_trials_since_best_result = 0
            else:
                self.num_trials_since_best_result += 1
            self.update_should_stop()

    def update_should_stop(self) -> None:
        if not self.should_stop:
            if self.num_trials_since_best_result == self.tuner_early_stopping:
                logging.info("Early stopping criteria met, searcher will shut down.")
                self.should_stop = True
            # +1 because the model profile info run shouldn't be counted.
            if len(self) == self.tuner_num_trials + 1:
                logging.info("All Trials completed, searcher will shut down.")
                self.should_stop = True

    def get_state_dict(self) -> Dict[str, Any]:
        state_dict = self.__dict__
        return state_dict

    @classmethod
    def from_state_dict(cls, state_dict: Dict[str, Any]) -> "DSATTrialTracker":
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

        # TODO: remove some of these info logs. Some are just for testing.
        logging.info(f"Appprox. max mbs per stage: {self.max_mbs_per_stage}")
        logging.info(f"Approx. GPU memory per stage: {self.mem_per_gpu_per_stage}")
        logging.info(f"Total GPU memory: {self.gpu_mem}")
        logging.info(f"Viable zero stages: {self.viable_zero_stages}")

    @property
    def gpu_mem(self) -> int:
        """
        Returns the available GPU memory in bytes.
        """
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
        # optimizer_mem assumes Adam, following DS. TODO: don't assume this.
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

    def __init__(self, submitted_config_dict: Dict[str, Any], model_dir: str) -> None:
        self.submitted_config_dict = submitted_config_dict
        self.model_dir = model_dir

        self.slots = self.submitted_config_dict["resources"]["slots_per_trial"]
        self.searcher_metric_name = self.submitted_config_dict["searcher"]["metric"]
        self.smaller_is_better = self.submitted_config_dict["searcher"].get(
            "smaller_is_better", _defaults.SMALLER_IS_BETTER
        )
        self.submitted_hps = self.submitted_config_dict["hyperparameters"]
        self.ds_config = get_ds_config_from_hparams(self.submitted_hps, self.model_dir)
        self.fp16 = self.ds_config.get("fp16", {}).get("enabled") or False

        self.autotuning_config = _defaults.AUTOTUNING_DICT  # TODO: let the user configure more.
        self.autotuning_config["metric"] = self.searcher_metric_name
        self.mp_size = self.autotuning_config["mp_size"]
        self.num_tuning_micro_batch_sizes = self.autotuning_config["num_tuning_micro_batch_sizes"]
        self.submitted_hps_with_autotuning = merge_dicts(
            self.submitted_hps, {_defaults.OVERWRITE_KEY: {"autotuning": self.autotuning_config}}
        )

        self.trial_tracker = DSATTrialTracker(
            slots=self.slots,
            model_dir=self.model_dir,
            autotuning_config=self.autotuning_config,
            smaller_is_better=self.smaller_is_better,
            searcher_metric_name=self.searcher_metric_name,
        )

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
        """Generates a list of new operations to run based on the results of the last trial."""
        pass

    def initial_operations(
        self, searcher_state: searcher.SearcherState
    ) -> List[searcher.Operation]:
        """
        Submits the model info profiling run in order to collect model and resources info to
        inform the search.
        """
        model_profile_info_hps = copy.deepcopy(self.submitted_hps)
        model_profile_info_hps[_defaults.OVERWRITE_KEY] = merge_dicts(
            model_profile_info_hps.get(_defaults.OVERWRITE_KEY, {}),
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
        # TODO: Remove print tests.
        logging.info(f"Calling on_validation_completed for {request_id}")
        last_trial = self.trial_tracker[request_id]
        # We catch explicit OOMs and report them in `report_completed`, since that information may
        # be useful to inform future search decisions.
        last_trial.metric = metric

        if last_trial.is_model_profiling_info_run:
            self.model_profile_info = DSATModelProfilingInfo(
                request_id=request_id,
                model_profiling_info_results=metric,
                slots=self.slots,
                fp16=self.fp16,
                mp_size=self.mp_size,
            )
        else:
            self.trial_tracker.update_best_trial_info(
                last_trial=last_trial,
            )

        if self.trial_tracker.should_stop:
            # Shutdown if `should_stop` is True, once all currently-running trials have completed.
            # TODO: Remove print tests.
            logging.info(
                f"******** Running Trials: {self.trial_tracker.running_trials_dict} ********"
            )
            new_ops_list = (
                [searcher.Shutdown()] if not self.trial_tracker.running_trials_dict else []
            )
        else:
            new_ops_list = self.get_new_searcher_ops_list(
                searcher_state=searcher_state,
                request_id=request_id,
                metric=metric,
                last_trial=last_trial,
            )

        return new_ops_list

    def on_trial_closed(
        self, searcher_state: searcher.SearcherState, request_id: uuid.UUID
    ) -> List[searcher.Operation]:
        # GG: This code should only be reached by errored trials, since every successful trial
        # ends with an `exit` rather than a Close operation.
        # TODO: Remove print tests and error raising.
        logging.info(f"Calling on_trial_closed for {request_id}")
        trial = self.trial_tracker[request_id]
        logging.info("********* Checking that the Closed Trial OOMed *********")
        logging.info(trial.metric)
        assert trial.metric.get(
            _defaults.OOM_KEY
        ), "`on_trial_closed` called for non-OOM-ing trial."
        return []

    def on_trial_exited_early(
        self,
        searcher_state: searcher.SearcherState,
        request_id: uuid.UUID,
        exited_reason: searcher.ExitedReason,
    ) -> List[searcher.Operation]:
        # TODO: Remove print tests.
        logging.info(f"Calling on_trial_exited_early for {request_id}")
        last_trial = self.trial_tracker[request_id]
        last_trial.error = True
        self.trial_tracker.update_best_trial_info(
            last_trial=last_trial,
        )

        if last_trial.is_model_profiling_info_run:
            logging.info(
                "**** Shutting down DeepSpeed Autotune: Error in Model Profiling Info Trial ****"
            )
            new_ops_list = [searcher.Shutdown()]
        elif exited_reason == searcher.ExitedReason.ERRORED:
            # Some early exits are due to OOMs which are caught by the `dsat_reporting_context`
            # context manager. Handle these cases differently from other errors.
            # NOTE: some early exits are actually OOMs which take a path other than the standard
            # RuntimeError. Handle theses cases.
            if self.trial_tracker.should_stop:
                # Shutdown if `should_stop` is True, once all currently-running trials have completed.
                logging.info(
                    f"******** Running Trials: {self.trial_tracker.running_trials_dict} ********"
                )
                new_ops_list = (
                    [searcher.Shutdown()] if not self.trial_tracker.running_trials_dict else []
                )
            else:
                new_ops_list = self.get_new_searcher_ops_list(
                    searcher_state=searcher_state,
                    request_id=request_id,
                    metric=None,
                    last_trial=last_trial,
                )
        else:
            # TODO: this code should never be reached, except for user error, so it's essentially
            # here as a test. Remove later.
            logging.info(f"############### SHOULD NOT HAVE BEEN REACHED ##############")
            logging.info(f"**** Shutting down DeepSpeed Autotune due to {exited_reason} ****")
            logging.info(f"############### SHOULD NOT HAVE BEEN REACHED ##############")
            raise RuntimeError(f"Something went wrong: Trial existed with {exited_reason}")
        # After deleting the above tests, we will just shut down in these cases:
        # new_ops_list = [searcher.Shutdown()]

        return new_ops_list

    def progress(self, searcher_state: searcher.SearcherState) -> float:
        progress = (
            len(searcher_state.trials_closed | searcher_state.failures)
            / self.trial_tracker.tuner_num_trials
        )
        return progress

    def save_method_state(self, path: pathlib.Path) -> None:
        if self.model_profile_info is None:  # Just in case of a very early failure.
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
                self.model_profile_info = None
            else:
                self.model_profile_info = DSATModelProfilingInfo.from_state_dict(model_profile_info)


class DSATRandomSearchMethod(DSATSearchMethodBase):
    """
    Base class for all DS AT searchers. Written so that only the `get_new_searcher_ops_list` method
    needs to be written overwritten when subclassing (at a minimum).

    Some unusual behavior to note:
        * Successfully completed DS AT Trials call `exit` explicitly, negating the need to `Close`
        any DS AT trials. (In fact, posting a `Close` operation will lead to periodic errors due to
        race conditions.) Because of the explicit `exit` call, such successful Trials *do not*
        trigger the searcher's `on_trial_closed` method.
        * The `dsat_reporting_context` context manager catches OOMs, calls
        `report_validation_metrics, `op.report_completed`, and the re-raises the OOM error.
        Consequently, all three of `on_trial_closed`, `on_validation_completed`, and
        `on_trial_exited_early` are usually triggered by most OOM-ing Trials. Qualifiers are
        necessary because there appear to be cases where trials OOM but `on_validation_completed` is
        not called (presumably because of race conditions), and some OOMs are not captured by the
        context manager because they raise obscure errors like illegal memory accesses, rather than
        the usual RuntimeError.
    """

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
        if last_trial.is_model_profiling_info_run:
            new_ops_list = self.get_ops_list_after_model_profiling_info_run()
        elif len(searcher_state.trials_created) < self.trial_tracker.tuner_num_trials:
            new_ops_list = self.get_ops_list_after_autotuning_run(last_trial)
        else:
            new_ops_list = []
        return new_ops_list

    def get_ops_list_after_model_profiling_info_run(
        self,
    ) -> List[searcher.Operation]:
        # This isn't actually how native DS AT uses num_tuning_micro_batch_sizes, but it's a good
        # enough placeholder usage until we get other aspects of custom searcher DS AT to work.
        approx_num_lineages = (
            self.trial_tracker.tuner_num_trials // self.num_tuning_micro_batch_sizes
        )
        new_ops_list = []
        for _ in range(approx_num_lineages):
            hparams, search_data = self.get_random_hparams_and_search_data()
            new_trial = self.trial_tracker.create_trial(
                hparams=hparams,
                search_data=search_data,
                parent_trial=None,
            )
            # A +1 is required to align DS step/DET max_length conventions.
            # TODO: DS has a fixed notion of what a step is while Determined does not. Make sure
            # there are no issues in reconciling this fact.
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
            hparams, search_data = self.get_hparams_and_search_data_after_trial(
                last_trial=last_trial,
            )
            parent_trial = last_trial

            # TODO: remove below print tests.
            logging.info("**************** BSZ History ****************")
            bsz_history = []
            print_trial = last_trial
            while print_trial is not None:
                bsz = print_trial.ds_config["train_micro_batch_size_per_gpu"]
                bsz_history.append(bsz)
                print_trial = print_trial.parent
            logging.info(f"History (to-be-submitted last): {str(list(reversed(bsz_history)))}")
            logging.info("**************** BSZ History End ****************")

        else:
            hparams, search_data = self.get_random_hparams_and_search_data()
            parent_trial = None

        if hparams is None:
            # hparams will be None if a previously used configuration was suggested by the searcher.
            new_ops_list = []
        else:
            new_trial = self.trial_tracker.create_trial(
                hparams=hparams,
                search_data=search_data,
                parent_trial=parent_trial,
            )
            # A +1 is required to align DS step/DET max_length conventions.
            end_profile_step = self.autotuning_config["end_profile_step"] + 1
            new_ops_list = self.trial_tracker.get_ops_list_from_trial(
                trial=new_trial, length=end_profile_step
            )
        return new_ops_list

    def get_hparams_and_search_data_after_trial(
        self,
        last_trial: DSATTrial,
    ) -> Tuple[Optional[Dict[str, Any]], Optional[Dict[str, int]]]:
        """
        Get new hparams and search data according to the results of a parent trial.
        Performs a slightly modified binary search on the train_micro_batch_size_per_gpu.
        """
        # TODO: verify we are always quitting when no more non-trivial trials are possible.

        lo, hi = last_trial.search_data["lo"], last_trial.search_data["hi"]
        mid = (lo + hi) // 2
        # TODO: edge cases and +- 1 error checks.
        if last_trial.error:
            hi = mid - 1
        else:
            lo = mid + 1
            hi = (
                hi if last_trial.error_in_direct_history else int(1.05 * hi)
            )  # TODO: let user configure ceiling factor. Current number is just a guess, and maybe
            # what native DS AT does?
        new_mid = (lo + hi) // 2
        if new_mid in last_trial.mbs_in_lineage:
            # Already tried this configuration.
            new_hparams = new_search_data = None
        else:
            new_hparams = copy.deepcopy(last_trial.hparams)
            new_hparams[_defaults.OVERWRITE_KEY]["train_micro_batch_size_per_gpu"] = new_mid
            new_search_data = {"lo": lo, "hi": hi}
        return new_hparams, new_search_data

    def get_random_hparams_and_search_data(self) -> Tuple[Dict[str, Any], Dict[str, Any]]:
        # TODO: verify that we are not repeating a previously attempted config.
        random_zero_stage = random.choice(tuple(self.model_profile_info.viable_zero_stages))
        new_hparams = copy.deepcopy(self.submitted_hps_with_autotuning)
        zero_optim_config = _utils.get_random_zero_optim_dict_for_zero_stage(random_zero_stage)
        new_hparams[_defaults.OVERWRITE_KEY] = merge_dicts(
            new_hparams.get(_defaults.OVERWRITE_KEY, {}),
            {"zero_optimization": zero_optim_config},
        )
        random_zero_stage_max_mbs = self.model_profile_info.max_mbs_per_stage[random_zero_stage]
        lo, hi = 1, 2 * random_zero_stage_max_mbs - 1
        mid = (lo + hi) // 2
        search_data = {
            "lo": lo,
            "hi": hi,
        }
        new_hparams[_defaults.OVERWRITE_KEY]["train_micro_batch_size_per_gpu"] = mid
        return (new_hparams, search_data)
