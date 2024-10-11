import shutil
import tempfile

import pytest

from tests import api_utils, detproc
from tests.experiment import noop


@pytest.mark.e2e_cpu
def test_support_bundle() -> None:
    sess = api_utils.user_session()
    exp_ref = noop.create_experiment(sess)
    exp_ref.wait(interval=0.01)

    outdir = tempfile.mkdtemp(suffix="test-support-bundle")
    try:
        trial_id = exp_ref.get_trials()[0].id
        command = ["det", "trial", "support-bundle", str(trial_id), "-o", outdir]
        detproc.check_call(sess, command)
    finally:
        shutil.rmtree(outdir)
