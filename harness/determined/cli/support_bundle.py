from argparse import Namespace
from determined.common import api
from determined.common.declarative_argparse import Arg, Cmd
from determined.common.api import authentication, bindings
import tarfile
import json

@authentication.required
def output_logs(args: Namespace): 
    output_dir_tar = tarfile.open(f'{args.output_dir}.tar.gz', 'w:gz')

    trial_logs_file = write_trial_logs(args, args.trial_id, output_dir_tar)
    output_dir_tar.add(f'{trial_logs_file}.json')
    
    master_logs_file = write_master_logs(args.trial_id, output_dir_tar)
    output_dir_tar.add(f'{master_logs_file}.json')

    api_trail_file, api_experiment_file = write_api_call(args.trial_id, output_dir_tar)
    output_dir_tar.add(f'{api_trail_file}.json')
    output_dir_tar.add(f'{api_experiment_file}.json')

    output_dir_tar.close()

    return f'{args.output_dir}.tar.gz'

    

def write_trial_logs(args,trial_id, output_dir):  #difference between this trial_logs and api call? There doesn't seem to be any. 
    trial_logs = api.task_logs(args.master, trial_id)
    # use trial logs to get the experiment id 
    file_name = 'trial_logs'
    trial_logs_list = []
    for log in trial_logs: 
        trial_logs_list.append(log)

    create_json_file_in_dir(trial_logs_list,file_name, output_dir)
    return file_name

def write_master_logs(args, trial_id, output_dir):
    response = api.get(
                args.master, "logs"
            )
    file_name = 'master_logs'
    create_json_file_in_dir(response,file_name, output_dir)
    return file_name

def write_api_call(trial_id, output_dir): 
    #r = bindings.get_GetExperiment(sess, experimentId=exp_id).experiment -> how would I get experiment id associated with trial? From task table in database?
    bindings.get_GetExperiment()
    file_name1 = 'api_experiment_call'
    file_name2 = 'api_trial_call'

    trial_obj = bindings.get_GetTrial(trialId=trial_id)
    experiment_id = trial_obj.experimentId
    exp_obj = bindings.get_GetExperiment(experimentId=experiment_id)

    trial_obj.to_json(), exp_obj.to_json()
    create_json_file_in_dir(exp_obj.to_json(), file_name1, output_dir)
    create_json_file_in_dir(trial_obj.to_json(), file_name2, output_dir)
    return file_name1, file_name2

def create_json_file_in_dir(content, file_name, output_dir): 
    with open('{file_name}.json', 'w') as f: 
        json.dump(content, f)
    
    f.close()

args_description = [
    Cmd(
        "support-bundle",
        output_logs,
        "support bundle",
        [
           Arg("-t", "--trial_id", type=int, help="trial ID"),
           Arg(
                        "-o",
                        "--output-dir",
                        type=str,
                        default=None,
                        help="Desired output directory for the logs",
           ),
        ],
            ),
]