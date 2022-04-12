import pytest
from tests import experiment as exp
from tests import config as conf
from determined.common.api import authentication, bindings, certs
from determined.common.experimental import session
from determined.common.api.bindings import determinedexperimentv1State
import time


def create_exp_get_trial_id():
    exp_id = exp.create_experiment(conf.cv_examples_path("cifar10_pytorch/const.yaml"),conf.cv_examples_path("cifar10_pytorch"))
    exp.wait_for_experiment_state(exp_id, determinedexperimentv1State.STATE_ACTIVE) 
    session = test_session()
    trials = bindings.get_GetExperimentTrials(session, experimentId=exp_id).trials    
    while(len(trials) <= 0):
        print(f'no of trials: {len(trials)}')
        time.sleep(5 * 60) # wait for 5 minutes to check again whether trials were added 
        trials = bindings.get_GetExperimentTrials(session, experimentId=exp_id).trials    

    assert len(trials) > 0
    trial_example = trials[0]
    trial_id = trial_example.id
    return trial_id

def test_session() -> session.Session:
    murl = conf.make_master_url()
    certs.cli_cert = certs.default_load(murl)
    authentication.cli_auth = authentication.Authentication(murl, try_reauth=True)
    return session.Session(murl, "determined", authentication.cli_auth, certs.cli_cert)




