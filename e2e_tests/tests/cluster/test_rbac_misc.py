import datetime
from typing import Callable, List, Tuple

import pytest

from determined.common import api
from determined.common.api import bindings, errors
from tests import api_utils
from tests.cluster import test_rbac


@pytest.mark.e2e_cpu_rbac
@api_utils.skipif_rbac_not_enabled()
def test_cluster_admin_only_calls() -> None:
    admin = api_utils.admin_session()
    with test_rbac.create_workspaces_with_users(
        [
            [
                (1, ["Editor"]),
                (2, ["Viewer"]),
                (3, []),
            ],
        ]
    ) as (_, creds):
        u_admin_role, _ = api_utils.create_test_user(
            user=bindings.v1User(username=api_utils.get_random_string(), active=True, admin=False),
        )
        api_utils.assign_user_role(
            session=admin, user=u_admin_role.username, role="ClusterAdmin", workspace=None
        )

        # normal determined admins without ClusterAdmin role.
        u_det_admin, _ = api_utils.create_test_user(
            user=bindings.v1User(username=api_utils.get_random_string(), active=True, admin=True),
        )

        def get_agent_slot_ids(sess: api.Session) -> Tuple[str, str]:
            agents = sorted(bindings.get_GetAgents(sess).agents, key=lambda a: a.id)
            assert len(agents) > 0
            agent = agents[0]
            assert agent.slots is not None
            key = sorted(agent.slots.keys())[0]
            slot_id = agent.slots[key].id
            assert slot_id is not None
            return agent.id, slot_id

        def enable_agent(sess: api.Session) -> None:
            agent_id, _ = get_agent_slot_ids(sess)
            bindings.post_EnableAgent(sess, agentId=agent_id)

        def disable_agent(sess: api.Session) -> None:
            agent_id, _ = get_agent_slot_ids(sess)
            bindings.post_DisableAgent(
                sess, agentId=agent_id, body=bindings.v1DisableAgentRequest(agentId=agent_id)
            )

        def enable_slot(sess: api.Session) -> None:
            agent_id, slot_id = get_agent_slot_ids(sess)
            bindings.post_EnableSlot(sess, agentId=agent_id, slotId=slot_id)

        def disable_slot(sess: api.Session) -> None:
            agent_id, slot_id = get_agent_slot_ids(sess)
            bindings.post_DisableSlot(
                sess, agentId=agent_id, slotId=slot_id, body=bindings.v1DisableSlotRequest()
            )

        def get_master_logs(sess: api.Session) -> None:
            logs = list(bindings.get_MasterLogs(sess, limit=2))
            assert len(logs) == 2

        def get_allocations_raw(sess: api.Session) -> None:
            EXPECTED_TIME_FMT = "%Y-%m-%dT%H:%M:%S.000Z"
            start = datetime.datetime.now()
            start_str = start.strftime(EXPECTED_TIME_FMT)
            end_str = (start + datetime.timedelta(seconds=1)).strftime(EXPECTED_TIME_FMT)
            entries = bindings.get_ResourceAllocationRaw(
                sess,
                timestampAfter=start_str,
                timestampBefore=end_str,
            ).resourceEntries
            assert isinstance(entries, list)

        def get_allocations_aggregated(sess: api.Session) -> None:
            EXPECTED_TIME_FMT = "%Y-%m-%d"
            start = datetime.datetime.now()
            end = start + datetime.timedelta(seconds=1)
            entries = bindings.get_ResourceAllocationAggregated(
                sess,
                period=bindings.v1ResourceAllocationAggregationPeriod.DAILY,
                startDate=start.strftime(EXPECTED_TIME_FMT),
                endDate=end.strftime(EXPECTED_TIME_FMT),
            ).resourceEntries
            assert isinstance(entries, list)

        def get_allocations_raw_echo(sess: api.Session) -> None:
            EXPECTED_TIME_FMT = "%Y-%m-%dT%H:%M:%S.000Z"
            start = datetime.datetime.now()
            start_str = start.strftime(EXPECTED_TIME_FMT)
            end_str = (start + datetime.timedelta(seconds=1)).strftime(EXPECTED_TIME_FMT)
            url = "/resources/allocation/raw"
            params = {"timestamp_after": start_str, "timestamp_before": end_str}
            response = sess.get(url, params=params)
            assert response.status_code == 200

        # FIXME: these can potentially affect other tests running against the same cluster.
        # the targeted agent_id and slot_id are not guaranteed to be the same across checks.

        checks: List[Callable[[api.Session], None]] = [
            get_master_logs,
            get_allocations_raw,
            get_allocations_aggregated,
            get_allocations_raw_echo,
            disable_agent,
            enable_agent,
            disable_slot,
            enable_slot,
        ]

        for check in checks:
            allowed_users = [(u_admin_role, "ClusterAdmin")]
            disallowed_users = zip(
                [creds[1], creds[2], creds[3], u_det_admin],
                ["Editor", "Viewer", "", "u.Admin"],
            )
            for user, role in allowed_users:
                print(f"testing {check.__name__} as ({role})")
                check(user)

            for user, role in disallowed_users:
                print(f"testing {check.__name__} as ({role})")
                with pytest.raises(errors.ForbiddenException):
                    check(user)
