from __future__ import annotations

import contextlib
import time
from typing import Iterator

from determined.cli.tunnel import ListenerConfig, http_tunnel_listener
from determined.common.api import Session, authentication, bindings


@contextlib.contextmanager
def _tunnel_task(sess: Session, task_id: str, port_map: dict[int, int]) -> Iterator[None]:
    # Args:
    #   port_map: dict of local port => task port.
    #   task_id: tunneled task_id.

    master_addr = sess._master
    listeners = [
        ListenerConfig(service_id=f"{task_id}:{task_port}", local_port=local_port)
        for local_port, task_port in port_map.items()
    ]
    cert = sess._cert
    cert_file, cert_name = None, None
    if cert is not None:
        cert_file = cert.bundle
        cert_name = cert.name

    token = authentication.must_cli_auth().get_session_token()

    with http_tunnel_listener(master_addr, listeners, cert_file, cert_name, token):
        yield


@contextlib.contextmanager
def _tunnel_trial(sess: Session, trial_id: int, port_map: dict[int, int]) -> Iterator[None]:
    # TODO(DET-9000): perhaps the tunnel should be able to probe master for service status,
    # instead of us explicitly polling for task/trial status.
    while True:
        resp = bindings.get_GetTrial(sess, trialId=trial_id)
        trial = resp.trial

        terminal_states = [
            bindings.experimentv1State.STATE_COMPLETED,
            bindings.experimentv1State.STATE_CANCELED,
            bindings.experimentv1State.STATE_ERROR,
        ]
        if trial.state in terminal_states:
            raise ValueError("Can't tunnel a trial in terminal state")

        task_id = trial.taskId
        if task_id is not None:
            break
        else:
            time.sleep(0.1)

    with _tunnel_task(sess, task_id, port_map):
        yield


@contextlib.contextmanager
def tunnel_experiment(
    sess: Session, experiment_id: int, port_map: dict[int, int]
) -> Iterator[None]:
    while True:
        trials = bindings.get_GetExperimentTrials(sess, experimentId=experiment_id).trials
        if len(trials) > 0:
            break
        else:
            time.sleep(0.1)

    first_trial_id = sorted(t.id for t in trials)[0]

    with _tunnel_trial(sess, first_trial_id, port_map):
        yield


def parse_port_map_flag(publish_arg: list[str]) -> dict[int, int]:
    result = {}  # type: dict[int, int]

    for e in publish_arg:
        try:
            if ":" in e:
                lp, tp = e.split(":")
                local_port, task_port = int(lp), int(tp)
                result[local_port] = task_port
            else:
                port = int(e)
                result[port] = port
        except ValueError as e:
            raise ValueError(f"failed to parse --publish argument: {e}") from e

    return result
