import re
import subprocess
import traceback
from datetime import datetime, timedelta, timezone
from typing import Any, Dict, Tuple

from dateutil import parser

from determined.common import api
from determined.common.api import authentication, bindings, certs
from tests import config as conf

ADD_KEY = "adding"
REMOVE_KEY = "removing"


def iso_date_to_epoch(iso_date: str) -> int:
    return int(parser.parse(iso_date).timestamp())


def parse_log_for_gpu_stats(log_path: str) -> Tuple[int, str, str]:
    date_parsing_re = re.compile(r"(\d{4}-\d{2}-\d{2}\S+).*")
    line_parsing_re = re.compile(r"(\S+).*(adding|removing) agent: (\S+).*")
    # agent_parsing_re = re.compile("det-agent-argo-dai-dev-[a-z]*-[a-z]*")

    agent_event_mapping: Dict[str, Any] = {}
    # min_ts = 1596240000 # Override if logs start significantly later than start of day
    # max_ts = 1597622399 # Override if logs end significantly earlier than end of day
    min_ts = -1  # will infer start time based on earliest log timestamp
    max_ts = -1  # will infer end time based on latest log timestamp

    with open(log_path, "r") as f:
        for _, line in enumerate(f):
            match_date = date_parsing_re.match(line)
            if match_date:
                try:
                    ts = iso_date_to_epoch(match_date.group(1))
                except parser.ParserError:
                    print("Skip unrecognized date time format ", match_date.group(1))
                    continue
                max_ts = ts if max_ts == -1 or ts > max_ts else max_ts
                min_ts = ts if min_ts == -1 or ts < min_ts else min_ts
            match_line = line_parsing_re.match(line)
            if match_line:
                ts = iso_date_to_epoch(match_line.group(1))
                event = match_line.group(2)
                agent_id = match_line.group(3)
                if agent_id not in agent_event_mapping:
                    agent_event_mapping[agent_id] = {}
                agent_event_mapping[agent_id][event] = ts

    total_agent_uptime_sec = 0

    for agent_id in agent_event_mapping:
        times = agent_event_mapping[agent_id]
        if ADD_KEY not in times:
            print(f"Warning: {agent_id} has no start time logged, assuming {min_ts}")
            agent_event_mapping[agent_id][ADD_KEY] = min_ts
        if REMOVE_KEY not in times:
            print(f"Warning: {agent_id} has no end time logged, assuming {max_ts}")
            agent_event_mapping[agent_id][REMOVE_KEY] = max_ts
        start = times[ADD_KEY] if ADD_KEY in times else min_ts
        end = times[REMOVE_KEY] if REMOVE_KEY in times else max_ts
        total_agent_uptime_sec += end - start
        agent_uptime_hours = (end - start) / 3600
        print(f"{agent_id}: {agent_uptime_hours} hours")

    global_start = datetime.fromtimestamp(min_ts, tz=timezone(timedelta(hours=0))).strftime(
        "%Y-%m-%dT%H:%M:%S.000Z"
    )
    global_end = datetime.fromtimestamp(max_ts, tz=timezone(timedelta(hours=0))).strftime(
        "%Y-%m-%dT%H:%M:%S.000Z"
    )
    print(f"\nMaster log time period: {global_start} to {global_end} \n")
    print(f"Total agent up seconds: {total_agent_uptime_sec} ")
    return total_agent_uptime_sec, global_start, global_end


log_path = "/tmp/det-master.log"


def fetch_master_log() -> bool:
    command = ["det", "-m", conf.make_master_url(), "master", "logs"]
    try:
        output = subprocess.check_output(command)
    except Exception:
        traceback.print_exc()
        return False
    with open(log_path, "wb") as log:
        log.write(output)
    return True


def create_test_session() -> api.Session:
    murl = conf.make_master_url()
    certs.cli_cert = certs.default_load(murl)
    authentication.cli_auth = authentication.Authentication(murl, try_reauth=True)
    return api.Session(murl, "determined", authentication.cli_auth, certs.cli_cert)


def compare_stats() -> None:
    if not fetch_master_log():
        print("Skip compare stats because error at fetch master")
        return
    gpu_from_log, global_start, global_end = parse_log_for_gpu_stats(log_path)
    res = bindings.get_ResourceAllocationRaw(
        create_test_session(), timestampAfter=global_start, timestampBefore=global_end
    )
    gpu_from_api = 0
    gpu_from_api_map = {}
    instance_from_api = 0
    instance_from_api_map = {}
    for r in (res.to_json())["resourceEntries"]:
        if r["kind"] == "agent" and r["seconds"] > 0:
            gpu_from_api += r["seconds"]
            if r["username"] not in gpu_from_api_map:
                gpu_from_api_map[r["username"]] = 0
            gpu_from_api_map[r["username"]] += r["seconds"]
        if r["kind"] == "instance" and r["seconds"] > 0:
            instance_from_api += r["seconds"]
            if r["username"] not in instance_from_api_map:
                instance_from_api_map[r["username"]] = 0
            instance_from_api_map[r["username"]] += r["seconds"]
    for ins in instance_from_api_map:
        # make sure instance initialization time is less than 5 mins
        if ins in gpu_from_api_map:
            assert instance_from_api_map[ins] - gpu_from_api_map[ins] < 60 * 5

    print(f"Agent time: logs={gpu_from_log}, api={gpu_from_api}")
    # make sure agent stats get from script is less than 5% difference with those get from api
    assert abs(gpu_from_log - gpu_from_api) <= max(gpu_from_api, gpu_from_log) * 0.05
