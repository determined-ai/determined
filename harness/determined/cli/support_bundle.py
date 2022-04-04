from argparse import Namespace
from determined.common import api
from determined.common.declarative_argparse import Arg, Cmd
from determined.common.api import authentication, bindings
import tarfile

@authentication.required
def output_logs(args: Namespace): 
    output_dir_tar = tarfile.open(f'{args.output_dir}.tar.gz')

    err = write_trial_logs(args.trial_id, output_dir_tar)
    if err != nil:
        raise.Error("Unable to write trial logs to a tar file")
    
    err = write_master_logs(args.trial_id, output_dir_tar)
     if err != nil:
        raise.Error("Unable to write master logs to a tar file")
   
    err = write_api_logs(args.trial_id, output_dir_tar)
     if err != nil:
        raise.Error("Unable to api logs to a tar file")

def write_trial_logs(trial_id, output_dir):  #difference between this trial_logs and api call? There doesn't seem to be any. 
    trial_logs = api.trial_logs(args.master, trial_id)
     create_file_in_dir(trial_logs,'trial_logs', output_dir)

def write_master_logs(trial_id, output_dir):
    response = api.get(
                args.master, "logs"
            )
    create_file_in_dir(response,'master_logs', output_dir)


def write_api_logs(trial_id, output_dir): 
    #r = bindings.get_GetExperiment(sess, experimentId=exp_id).experiment -> how would I get experiment id associated with trial? From task table in database?
    return None

def create_json_file_in_dir(content, file_name, output_dir): 
    with open('{file_name}.json', 'w') as f: 
        json.dump(content, f)
        
    outpur_dir.add(f'{file_name}.json')
    

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