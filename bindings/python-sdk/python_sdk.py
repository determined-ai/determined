import io
import time
from typing import Any, Dict, List, Optional, Set, Tuple

from pathlib import Path
from pprint import pprint
from determined.common import api, constants, context, yaml

import swagger_client
from swagger_client.api.authentication_api import AuthenticationApi  # noqa: E501
from swagger_client.rest import ApiException
from swagger_client import models
from swagger_client.models import V1CreateExperimentRequest as CreateExperimentRequest
from swagger_client.models import V1CreateExperimentResponse as CreateExperimentResponse
from swagger_client.models import V1File as V1File
from swagger_client.models import V1GetExperimentResponse as GetExperimentResponse
from swagger_client.models import V1GetExperimentsResponse as GetExperimentsResponse
from swagger_client.models import V1GetExperimentTrialsResponse as GetExperimentTrialsResponse
from swagger_client.models import V1GetTrialResponse as GetTrialResponse
from swagger_client.models import V1GetExperimentLabelsResponse as GetExperimentLabelsResponse
from swagger_client.models import Determinedexperimentv1State as Determinedexperimentv1State

def _print_exception(e: ApiException):
    print("{} {} {}".format(e.status, e.reason, e.body))

def _parse_config_file(config_file: io.FileIO) -> Dict:
    experiment_config = yaml.safe_load(config_file.read())
    config_file.close()
    return experiment_config

def _wait_for_experiment_complete(det: object, exp_id: int, sleep_interval: int = 1) -> GetExperimentResponse:    
    while True:
        exp_resp = det.get_experiment(exp_id = exp_id)
        if exp_resp is None:
            print("Invalid Experiemnt ID")
        else:
            if (exp_resp.experiment.state == Determinedexperimentv1State.COMPLETED or
                exp_resp.experiment.state == Determinedexperimentv1State.CANCELED or
                exp_resp.experiment.state == Determinedexperimentv1State.DELETED or
                exp_resp.experiment.state == Determinedexperimentv1State.ERROR):
                print("Experiment {} is in a terminal state".format(exp_id))
                break
            elif exp_resp.experiment.state == Determinedexperimentv1State.PAUSED:
                raise ValueError("Experiment {} is in paused state".format(exp_id))
            else:
                # ACTIVE, STOPPING_COMPLETED, etc.
                print("Waiting for Experiment {} to complete".format(exp_id))
                time.sleep(sleep_interval)

    return exp_resp

def path_to_files(path):
    files = []
    for item in context.read_context(path)[0]:      
        content = item["content"].decode('utf-8')
        file = V1File(
            path = item["path"],
            type = item["type"],
            content = content,
            mtime = item["mtime"],
            uid = item["uid"],
            gid = item["gid"],
            mode = item["mode"],
        )
        files.append(file)
    return files

class Core:
    """
    Initialize Swagger-instance and set API endpoints

    user_config_file is a .yaml file with the following three fields -
    host: "URL of master"
    username: "user-name"
    password: "pwd-of-user"
    """

    def __init__(self, user_config_file: str):
        self.user_config_file = user_config_file
        self.configuration = None
        self.auth = None
        self.checkpoints = None
        self.cluster = None
        self.commands = None
        self.experiments = None
        self.internal = None
        self.models = None
        self.notebooks = None
        self.shells = None
        self.templates = None
        self.tensorboards = None
        self.trials = None
        self.users = None

    def initialize(self):
        try:
            f = open(self.user_config_file)
        except OSError:
            print("Error in opening user config file: {}".format(user_config_file))
            sys.exit(1)

        user_config = _parse_config_file(f)

        self.configuration = swagger_client.Configuration()
        self.configuration.host = user_config['host']
        self.configuration.username = user_config['username']
        self.configuration.password = user_config['password']
        self.configuration.api_key_prefix['Authorization'] = 'Bearer'

        self.auth = swagger_client.AuthenticationApi(swagger_client.ApiClient(self.configuration))
        api_response = self.auth.determined_login(models.V1LoginRequest(self.configuration.username, self.configuration.password))        
        self.configuration.api_key['Authorization'] = api_response.token
        
        # Initialize API objects with the configurations

        self.checkpoints = swagger_client.CheckpointsApi(swagger_client.ApiClient(self.configuration))
        self.cluster = swagger_client.ClusterApi(swagger_client.ApiClient(self.configuration))
        self.commands = swagger_client.CommandsApi(swagger_client.ApiClient(self.configuration))
        self.experiments = swagger_client.ExperimentsApi(swagger_client.ApiClient(self.configuration))
        self.internal = swagger_client.InternalApi(swagger_client.ApiClient(self.configuration))
        self.models = swagger_client.ModelsApi(swagger_client.ApiClient(self.configuration))
        self.notebooks = swagger_client.NotebooksApi(swagger_client.ApiClient(self.configuration))
        self.shells = swagger_client.ShellsApi(swagger_client.ApiClient(self.configuration))
        self.templates = swagger_client.TemplatesApi(swagger_client.ApiClient(self.configuration))
        self.tensorboards = swagger_client.TensorboardsApi(swagger_client.ApiClient(self.configuration))
        self.trials = swagger_client.TrialsApi(swagger_client.ApiClient(self.configuration))
        self.users = swagger_client.UsersApi(swagger_client.ApiClient(self.configuration))

    def activate_experiment(self, exp_id: int) -> int:
        if self.experiments is None:
            self.initialize()

        ret_val = 0
        try:
            response = self.experiments.determined_activate_experiment(id = exp_id)
        except ApiException as e:
            ret_val = 1
            _print_exception(e)

        return ret_val

    def archive_experiment(self, exp_id: int) -> int:
        if self.experiments is None:
            self.initialize()

        ret_val = 0
        try:
            response = self.experiments.determined_archive_experiment(id = exp_id)
        except ApiException as e:
            ret_val = 1
            _print_exception(e)

        return ret_val

    def cancel_experiment(self, exp_id: int) -> int:
        if self.experiments is None:
            self.initialize()

        ret_val = 0
        try:
            response = self.experiments.determined_cancel_experiment(id = exp_id)
        except ApiException as e:
            ret_val = 1
            _print_exception(e)

        return ret_val

    def delete_experiment(self, exp_id: int) -> int:
        if self.experiments is None:
            self.initialize()

        ret_val = 0
        try:
            response = self.experiments.determined_delete_experiment(id = exp_id)
        except ApiException as e:
            ret_val = 1
            _print_exception(e)

        return ret_val

    def get_experiment(self, exp_id: int) -> GetExperimentResponse:
        if self.experiments is None:
            self.initialize()

        experiment_response = None
        try:
            experiment_response = self.experiments.determined_get_experiment(experiment_id = exp_id)
        except ApiException as e:
            _print_exception(e)

        return experiment_response

    def get_experiments(
            self,
            sort_by = "SORT_BY_ID",
            order_by = "ORDER_BY_DESC",
            offset = 0,
            limit = 0,
            description = "",
            labels = [],
            #archived = True,
            #states = Determinedexperimentv1State.COMPLETED,
            users = []) -> GetExperimentsResponse:

        if self.experiments is None:
            self.initialize()

        experiment_response = None
        try:
            experiment_response = self.experiments.determined_get_experiments(sort_by = sort_by,
                                    order_by = order_by,
                                    offset = offset,
                                    limit = limit,
                                    description = description,
                                    labels = labels,
                                    #archived = archived,
                                    #states = states,
                                    users = users)
        except ApiException as e:
            _print_exception(e)

        return experiment_response

    def get_experiment_labels(self) -> GetExperimentLabelsResponse:        
        if self.experiments is None:
            self.initialize()

        labels = None
        try:
            labels = self.experiments.determined_get_experiment_labels()
        except ApiException as e:            
            _print_exception(e)

        return labels

    def get_experiment_trials(self, exp_id: int) -> GetExperimentTrialsResponse:        
        if self.experiments is None:
            self.initialize()

        trials = None
        try:
            trials = self.trials.determined_get_experiment_trials(experiment_id = exp_id)
        except ApiException as e:            
            _print_exception(e)

        return trials

    def get_trial(self, trial_id: int) -> GetTrialResponse:        
        if self.experiments is None:
            self.initialize()

        trial = None
        try:
            trial = self.trials.determined_get_trial(trial_id = trial_id)
        except ApiException as e:            
            _print_exception(e)

        return trial

    def create_experiement_from_config(self, experiment_config: Dict, model_dir: str, validate_only: bool) -> CreateExperimentResponse:
        if self.internal is None:
            self.initialize()

        model_context = path_to_files(Path(model_dir))

        experiment_request = CreateExperimentRequest(
            model_definition = model_context,
            config = yaml.safe_dump(experiment_config),
            validate_only = validate_only,
        )

        experiment_response = None
        try:
            experiment_response = self.internal.determined_create_experiment(experiment_request)
        except ApiException as e:
            _print_exception(e)
            
        return experiment_response

    def create_experiement_from_file(self, config_file: str, model_dir: str, validate_only: bool) -> CreateExperimentResponse:
        try:
            f = open(config_file)
        except OSError:
            print("Error in opening experiment config file: {}".format(config_file))
            sys.exit(1)
        
        experiment_config = _parse_config_file(f)
            
        return self.create_experiement_from_config(experiment_config = experiment_config, model_dir = model_dir, validate_only = validate_only)

    def kill_experiment(self, exp_id: int) -> int:
        if self.experiments is None:
            self.initialize()

        ret_val = 0
        try:
            response = self.experiments.determined_kill_experiment(exp_id = exp_id)
        except ApiException as e:
            ret_val = 1
            _print_exception(e)

        return ret_val
    
    def kill_trial(self, trial_id: int) -> int:
        if self.trials is None:
            self.initialize()

        ret_val = 0
        try:
            response = self.trials.determined_kill_trial(trial_id = trial_id)
        except ApiException as e:
            ret_val = 1
            _print_exception(e)

        return ret_val

    def pause_experiment(self, exp_id: int) -> int:
        if self.experiments is None:
            self.initialize()

        ret_val = 0
        try:
            response = self.experiments.determined_pause_experiment(id = exp_id)
        except ApiException as e:
            ret_val = 1
            _print_exception(e)

        return ret_val

    def unarchive_experiment(self, exp_id: int) -> int:
        if self.experiments is None:
            self.initialize()

        ret_val = 0
        try:
            response = self.experiments.determined_unarchive_experiment(id = exp_id)
        except ApiException as e:
            ret_val = 1
            _print_exception(e)

        return ret_val
