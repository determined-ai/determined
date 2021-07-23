import base64
import distutils.util
import io
import os
import tarfile

from determined import constants
from determined.common.api import certs, request

if __name__ == "__main__":
    exp_id = os.environ["DET_EXPERIMENT_ID"]
    master_addr = os.environ["DET_MASTER_ADDR"]
    master_port = os.environ["DET_MASTER_PORT"]
    use_tls = distutils.util.strtobool(os.environ.get("DET_USE_TLS", "false"))

    master_url = f"http{'s' if use_tls else ''}://{master_addr}:{master_port}"
    certs.cli_cert = certs.default_load(master_url=master_url)

    resp = request.get(master_url, f"api/v1/experiments/{exp_id}/model_def")
    resp.raise_for_status()

    tgz = base64.b64decode(resp.json()["b64Tgz"])

    with tarfile.open(fileobj=io.BytesIO(tgz), mode="r:gz") as model_def:
        # Ensure all members of the tarball resolve to subdirectories.
        for path in model_def.getnames():
            if os.path.relpath(path).startswith("../"):
                raise ValueError(f"'{path}' in tarball would expand to a parent directory")
        model_def.extractall(path=constants.MANAGED_TRAINING_MODEL_COPY)
        model_def.extractall(path=".")
