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

    path = f"api/v1/experiments/{trial_info.experiment_id}/model_def"
    resp = None
    try:
        resp = request.get(info.master_url, path=path, cert=cert)
        resp.raise_for_status()
    except Exception:
        # Since this is the very first api call in the entrypoint script, and the call is made
        # before you can debug with a startup hook, we offer an overly-detailed explanation to help
        # sysadmins debug their cluster.
        resp_content = str(resp and resp.content)
        noverify = info.master_cert_file == "noverify"
        cert_content = None if noverify else info.master_cert_file
        if cert_content is not None:
            with open(cert_content) as f:
                cert_content = f.read()
        print(
            "Failed to download model definition from master.  This may be due to an address\n"
            "resolution problem, a certificate problem, a firewall problem, or some other\n"
            "networking error.\n"
            "Debug information:\n"
            f"    master_url: {info.master_url}\n"
            f"    endpoint: {path}\n"
            f"    tls_verify_name: {info.master_cert_name}\n"
            f"    tls_noverify: {noverify}\n"
            f"    tls_cert: {cert_content}\n"
            f"    response code: {resp and resp.status_code}\n"
            f"    response content: {resp_content}\n"
        )
        raise

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
