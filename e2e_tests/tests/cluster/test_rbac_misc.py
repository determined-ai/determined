import datetime
from typing import Callable, List, Tuple

import pytest

from determined.common.api import authentication, bindings, errors
from tests import api_utils
from tests.cluster.test_rbac import create_workspaces_with_users, rbac_disabled


@pytest.mark.e2e_cpu_rbac
@pytest.mark.skipif(rbac_disabled(), reason="ee rbac is required for this test")
def test_cluster_admin_only_calls() -> None:
    with create_workspaces_with_users(
        [
            [
                (1, ["Editor"]),
                (2, ["Viewer"]),
                (3, []),
            ],
        ]
    ) as (_, creds):
        u_admin_role = api_utils.create_test_user(
            add_password=True,
            user=bindings.v1User(username=api_utils.get_random_string(), active=True, admin=False),
        )
        session = api_utils.determined_test_session(admin=True)
        api_utils.assign_user_role(
            session=session, user=u_admin_role.username, role="ClusterAdmin", workspace=None
        )

        # normal determined admins without ClusterAdmin role.
        u_det_admin = api_utils.create_test_user(
            add_password=True,
            user=bindings.v1User(username=api_utils.get_random_string(), active=True, admin=True),
        )

        def get_agent_slot_ids(creds: authentication.Credentials) -> Tuple[str, str]:
            session = api_utils.determined_test_session(creds)
            agents = sorted(bindings.get_GetAgents(session).agents, key=lambda a: a.id)
            assert len(agents) > 0
            agent = agents[0]
            assert agent.slots is not None
            key = sorted(agent.slots.keys())[0]
            slot_id = agent.slots[key].id
            assert slot_id is not None
            return agent.id, slot_id

        def enable_agent(creds: authentication.Credentials) -> None:
            session = api_utils.determined_test_session(creds)
            agent_id, _ = get_agent_slot_ids(creds)
            bindings.post_EnableAgent(session, agentId=agent_id)

        def disable_agent(creds: authentication.Credentials) -> None:
            session = api_utils.determined_test_session(creds)
            agent_id, _ = get_agent_slot_ids(creds)
            bindings.post_DisableAgent(
                session, agentId=agent_id, body=bindings.v1DisableAgentRequest(agentId=agent_id)
            )

        def enable_slot(creds: authentication.Credentials) -> None:
            session = api_utils.determined_test_session(creds)
            agent_id, slot_id = get_agent_slot_ids(creds)
            bindings.post_EnableSlot(session, agentId=agent_id, slotId=slot_id)

        def disable_slot(creds: authentication.Credentials) -> None:
            session = api_utils.determined_test_session(creds)
            agent_id, slot_id = get_agent_slot_ids(creds)
            bindings.post_DisableSlot(
                session, agentId=agent_id, slotId=slot_id, body=bindings.v1DisableSlotRequest()
            )

        def get_master_logs(creds: authentication.Credentials) -> None:
            logs = list(bindings.get_MasterLogs(api_utils.determined_test_session(creds), limit=2))
            assert len(logs) == 2

        def get_allocations_raw(creds: authentication.Credentials) -> None:
            EXPECTED_TIME_FMT = "%Y-%m-%dT%H:%M:%S.000Z"
            start = datetime.datetime.now()
            start_str = start.strftime(EXPECTED_TIME_FMT)
            end_str = (start + datetime.timedelta(seconds=1)).strftime(EXPECTED_TIME_FMT)
            entries = bindings.get_ResourceAllocationRaw(
                api_utils.determined_test_session(creds),
                timestampAfter=start_str,
                timestampBefore=end_str,
            ).resourceEntries
            assert isinstance(entries, list)

        def get_allocations_aggregated(creds: authentication.Credentials) -> None:
            EXPECTED_TIME_FMT = "%Y-%m-%d"
            start = datetime.datetime.now()
            end = start + datetime.timedelta(seconds=1)
            entries = bindings.get_ResourceAllocationAggregated(
                api_utils.determined_test_session(creds),
                # fmt: off
                period=bindings.v1ResourceAllocationAggregationPeriod\
                .DAILY,
                # fmt: on
                startDate=start.strftime(EXPECTED_TIME_FMT),
                endDate=end.strftime(EXPECTED_TIME_FMT),
            ).resourceEntries
            assert isinstance(entries, list)

        def get_allocations_raw_echo(creds: authentication.Credentials) -> None:
            EXPECTED_TIME_FMT = "%Y-%m-%dT%H:%M:%S.000Z"
            start = datetime.datetime.now()
            start_str = start.strftime(EXPECTED_TIME_FMT)
            end_str = (start + datetime.timedelta(seconds=1)).strftime(EXPECTED_TIME_FMT)
            url = "/resources/allocation/raw"
            params = {"timestamp_after": start_str, "timestamp_before": end_str}
            session = api_utils.determined_test_session(creds)
            response = session.get(url, params=params)
            assert response.status_code == 200

        # FIXME: these can potentially affect other tests running against the same cluster.
        # the targeted agent_id and slot_id are not guaranteed to be the same across checks.

        checks: List[Callable[[authentication.Credentials], None]] = [
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
