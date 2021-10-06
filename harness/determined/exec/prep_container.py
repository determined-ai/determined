import argparse
import base64
import io
import os
import tarfile

import determined as det
from determined import constants
from determined.common.api import certs, request


def trial_prep(info: det.ClusterInfo, cert: certs.Cert) -> None:
    trial_info = det.TrialInfo._from_env()
    trial_info._to_file()

    resp = request.get(
        info.master_url, path=f"api/v1/experiments/{trial_info.experiment_id}/model_def", cert=cert
    )
    resp.raise_for_status()

    tgz = base64.b64decode(resp.json()["b64Tgz"])

    with tarfile.open(fileobj=io.BytesIO(tgz), mode="r:gz") as model_def:
        # Ensure all members of the tarball resolve to subdirectories.
        for path in model_def.getnames():
            if os.path.relpath(path).startswith("../"):
                raise ValueError(f"'{path}' in tarball would expand to a parent directory")
        model_def.extractall(path=constants.MANAGED_TRAINING_MODEL_COPY)
        model_def.extractall(path=".")


def do_rendezvous(info: det.ClusterInfo, cert: certs.Cert) -> None:
    r = request.get(
        info.master_url,
        path=f"/api/v1/allocations/{info.allocation_id}/rendezvous_info/{info.container_id}",
        cert=cert,
    )

    jri = r.json()["rendezvousInfo"]
    rendezvous_info = det.RendezvousInfo(
        container_addrs=jri["addresses"], container_rank=jri["rank"]
    )
    rendezvous_info._to_file()


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--trial", action="store_true")
    parser.add_argument("--resources", action="store_true")
    parser.add_argument("--rendezvous", action="store_true")
    args = parser.parse_args()

    # Avoid reading det.get_cluster_info(), which might (wrongly) set a singleton to None.
    info = det.ClusterInfo._from_file()
    if info is None:
        info = det.ClusterInfo._from_env()
        info._to_file()

    cert = certs.default_load(info.master_url)

    if args.trial:
        trial_prep(info, cert)

    if args.resources:
        det.ResourcesInfo._by_inspection()._to_file()

    if args.rendezvous:
        do_rendezvous(info, cert)
