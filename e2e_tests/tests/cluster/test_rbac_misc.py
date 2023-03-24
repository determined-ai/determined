from typing import Callable, List, Tuple

import pytest

from determined import cli
from determined.common.api import authentication, bindings, errors
from tests import utils
from tests.api_utils import (
    configure_token_store,
    create_test_user,
    determined_test_session,
    get_random_string,
)
from tests.cluster.test_rbac import create_workspaces_with_users, rbac_disabled

from .test_users import ADMIN_CREDENTIALS


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
        u_admin_role = create_test_user(
            add_password=True,
            user=bindings.v1User(username=get_random_string(), active=True, admin=False),
        )
        configure_token_store(ADMIN_CREDENTIALS)
        cli.rbac.assign_role(
            utils.CliArgsMock(
                username_to_assign=u_admin_role.username,
                role_name="ClusterAdmin",
            )
        )

        # normal determined admins without ClusterAdmin role.
        u_det_admin = create_test_user(
            add_password=True,
            user=bindings.v1User(username=get_random_string(), active=True, admin=True),
        )

        def get_agent_slot_ids(creds: authentication.Credentials) -> Tuple[str, str]:
            session = determined_test_session(creds)
            agents = sorted(bindings.get_GetAgents(session).agents, key=lambda a: a.id)
            assert len(agents) > 0
            agent = agents[0]
            assert agent.slots is not None
            key = sorted(agent.slots.keys())[0]
            slot_id = agent.slots[key].id
            assert slot_id is not None
            return agent.id, slot_id

        def enable_agent(creds: authentication.Credentials) -> None:
            session = determined_test_session(creds)
            agent_id = bindings.get_GetAgents(session).agents[0].id
            bindings.post_EnableAgent(session, agentId=agent_id)

        def disable_agent(creds: authentication.Credentials) -> None:
            session = determined_test_session(creds)
            agent_id = bindings.get_GetAgents(session).agents[0].id
            bindings.post_DisableAgent(
                session, agentId=agent_id, body=bindings.v1DisableAgentRequest(agentId=agent_id)
            )

        def enable_slot(creds: authentication.Credentials) -> None:
            session = determined_test_session(creds)
            agent_id, slot_id = get_agent_slot_ids(creds)
            bindings.post_EnableSlot(session, agentId=agent_id, slotId=slot_id)

        def disable_slot(creds: authentication.Credentials) -> None:
            session = determined_test_session(creds)
            agent_id, slot_id = get_agent_slot_ids(creds)
            bindings.post_DisableSlot(
                session, agentId=agent_id, slotId=slot_id, body=bindings.v1DisableSlotRequest()
            )

        # FIXME: these can potentially affect other tests running against the same cluster.
        # the targeted agent_id and slot_id are not guaranteed to be the same across checks.
        checks: List[Callable[[authentication.Credentials], None]] = [
            disable_agent,
            enable_agent,
            disable_slot,
            enable_slot,
        ]

        # TODO use pytest features.
        for check in checks:
            print(f"testing {check.__name__} with ClusterAdmin")
            check(u_admin_role)
            for user in [creds[1], creds[2], creds[3], u_det_admin]:
                print(f"testing {check.__name__} with {user.username}")
                with pytest.raises(errors.ForbiddenException):
                    check(user)
