import { Tabs } from 'antd';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';

import Page from 'components/Page';
import PageNotFound from 'components/PageNotFound';
import { useStore } from 'contexts/Store';
import useFeature from 'hooks/useFeature';
import { useFetchUsers } from 'hooks/useFetch';
import usePermissions from 'hooks/usePermissions';
import { paths } from 'routes/utils';
import {
  getGroups,
  getWorkspace,
  getWorkspaceMembers,
  searchRolesAssignableToScope,
} from 'services/api';
import { V1Group, V1GroupSearchResult, V1Role, V1RoleWithAssignments } from 'services/api-ts-sdk';
import Message, { MessageType } from 'shared/components/Message';
import Spinner from 'shared/components/Spinner';
import usePolling from 'shared/hooks/usePolling';
import { ValueOf } from 'shared/types';
import { isEqual } from 'shared/utils/data';
import { isNotFound } from 'shared/utils/service';
import { User, Workspace } from 'types';
import handleError from 'utils/error';

import css from './WorkspaceDetails.module.scss';
import WorkspaceDetailsHeader from './WorkspaceDetails/WorkspaceDetailsHeader';
import WorkspaceMembers from './WorkspaceDetails/WorkspaceMembers';
import WorkspaceProjects from './WorkspaceDetails/WorkspaceProjects';

type Params = {
  tab: string;
  workspaceId: string;
};

export const WorkspaceDetailsTab = {
  Members: 'members',
  Projects: 'projects',
} as const;

export type WorkspaceDetailsTab = ValueOf<typeof WorkspaceDetailsTab>;

const WorkspaceDetails: React.FC = () => {
  const rbacEnabled = useFeature().isOn('rbac');
  const mockWorkspaceMembers = useFeature().isOn('mock_workspace_members');

  const { users } = useStore();
  const { tab, workspaceId: workspaceID } = useParams<Params>();
  const [workspace, setWorkspace] = useState<Workspace>();
  const [groups, setGroups] = useState<V1GroupSearchResult[]>();
  const [usersAssignedDirectly, setUsersAssignedDirectly] = useState<User[]>([]);
  const [groupsAssignedDirectly, setGroupsAssignedDirectly] = useState<V1Group[]>([]);
  const [usersAssignedDirectlyIds, setUsersAssignedDirectlyIds] = useState<Set<number>>(
    new Set<number>(),
  );
  const [groupsAssignedDirectlyIds, setGroupsAssignedDirectlyIds] = useState<Set<number>>(
    new Set<number>(),
  );
  const [rolesAssignableToScope, setRolesAssignableToScope] = useState<V1Role[]>([]);
  /* eslint-disable-next-line @typescript-eslint/no-unused-vars */
  const [nameFilter, setNameFilter] = useState<string>();
  const [workspaceAssignments, setWorkspaceAssignments] = useState<V1RoleWithAssignments[]>([]);
  const [pageError, setPageError] = useState<Error>();
  const [canceler] = useState(new AbortController());
  const [tabKey, setTabKey] = useState<WorkspaceDetailsTab>(
    (tab as WorkspaceDetailsTab) || WorkspaceDetailsTab.Projects,
  );
  const pageRef = useRef<HTMLElement>(null);
  const workspaceId = workspaceID ?? '';
  const id = parseInt(workspaceId);
  const navigate = useNavigate();
  const { canViewWorkspace } = usePermissions();

  const fetchWorkspace = useCallback(async () => {
    try {
      const response = await getWorkspace({ id }, { signal: canceler.signal });
      setWorkspace(response);
    } catch (e) {
      if (!pageError) setPageError(e as Error);
    }
  }, [canceler.signal, id, pageError]);

  const fetchUsers = useFetchUsers(canceler);

  const fetchGroups = useCallback(async (): Promise<void> => {
    try {
      const response = await getGroups({ limit: 100 }, { signal: canceler.signal });

      setGroups((prev) => {
        if (isEqual(prev, response.groups)) return prev;
        return response.groups || [];
      });
    } catch (e) {
      handleError(e);
    }
  }, [canceler.signal]);

  const fetchGroupsAndUsersAssignedToWorkspace = useCallback(async () => {
    if (!rbacEnabled || mockWorkspaceMembers) {
      return;
    }

    const response = await getWorkspaceMembers({ nameFilter, workspaceId: id });
    const newGroupIds = new Set<number>();
    setUsersAssignedDirectly(response.usersAssignedDirectly);
    setUsersAssignedDirectlyIds(new Set(response.usersAssignedDirectly.map((user) => user.id)));
    setGroupsAssignedDirectly(response.groups);
    response.groups.forEach((group) => {
      if (group.groupId) {
        newGroupIds.add(group.groupId);
      }
    });
    setGroupsAssignedDirectlyIds(newGroupIds);
    setWorkspaceAssignments(response.assignments);
  }, [id, mockWorkspaceMembers, nameFilter, rbacEnabled]);

  const fetchRolesAssignableToScope = useCallback(async (): Promise<void> => {
    // Only fetch roles if rbac is enabled.
    if (!rbacEnabled) return;
    try {
      const response = await searchRolesAssignableToScope(
        { workspaceId: id },
        { signal: canceler.signal },
      );

      setRolesAssignableToScope((prev) => {
        if (isEqual(prev, response.roles)) return prev;
        return response.roles || [];
      });
    } catch (e) {
      handleError(e);
    }
  }, [canceler.signal, id]);

  const handleFilterUpdate = (name: string | undefined) => setNameFilter(name);

  const fetchAll = useCallback(async () => {
    await Promise.allSettled([
      fetchWorkspace(),
      fetchUsers(),
      fetchGroups(),
      fetchGroupsAndUsersAssignedToWorkspace(),
      fetchRolesAssignableToScope(),
    ]);
  }, [
    fetchWorkspace,
    fetchGroups,
    fetchUsers,
    fetchGroupsAndUsersAssignedToWorkspace,
    fetchRolesAssignableToScope,
  ]);

  usePolling(fetchAll, { rerunOnNewFn: true });

  const handleTabChange = useCallback(
    (activeTab) => {
      const tab = activeTab as WorkspaceDetailsTab;
      navigate(paths.workspaceDetails(workspaceId, tab), { replace: true });
      setTabKey(tab);
    },
    [workspaceId, navigate],
  );

  useEffect(() => {
    // Set the correct pathname to ensure
    // that user settings will save.
    navigate(paths.workspaceDetails(workspaceId, tab), { replace: true });
    tab && setTabKey(tab as WorkspaceDetailsTab);
  }, [workspaceId, navigate, tab]);

  // Users and Groups that are not already a part of the workspace
  const addableGroups: V1Group[] = groups
    ? groups
        .map((groupDetails) => groupDetails.group)
        .filter((group) => group.groupId && !groupsAssignedDirectlyIds.has(group.groupId))
    : [];
  const addableUsers = users.filter((user) => !usersAssignedDirectlyIds.has(user.id));
  const addableUsersAndGroups = [...addableGroups, ...addableUsers];

  useEffect(() => {
    return () => canceler.abort();
  }, [canceler]);

  if (isNaN(id)) {
    return <Message title={`Invalid Workspace ID ${workspaceId}`} />;
  } else if (pageError) {
    if (isNotFound(pageError)) return <PageNotFound />;
    const message = `Unable to fetch Workspace ${workspaceId}`;
    return <Message title={message} type={MessageType.Warning} />;
  } else if (!workspace) {
    return <Spinner tip={`Loading workspace ${workspaceId} details...`} />;
  }

  if (!canViewWorkspace({ workspace: { id } })) {
    return <PageNotFound />;
  }

  return (
    <Page
      className={css.base}
      containerRef={pageRef}
      headerComponent={
        <WorkspaceDetailsHeader
          addableUsersAndGroups={addableUsersAndGroups}
          fetchWorkspace={fetchAll}
          rolesAssignableToScope={rolesAssignableToScope}
          workspace={workspace}
        />
      }
      id="workspaceDetails">
      {rbacEnabled ? (
        <Tabs activeKey={tabKey} destroyInactiveTabPane onChange={handleTabChange}>
          <Tabs.TabPane destroyInactiveTabPane key={WorkspaceDetailsTab.Projects} tab="Projects">
            <WorkspaceProjects id={id} pageRef={pageRef} workspace={workspace} />
          </Tabs.TabPane>
          <Tabs.TabPane destroyInactiveTabPane key={WorkspaceDetailsTab.Members} tab="Members">
            <WorkspaceMembers
              assignments={workspaceAssignments}
              groupsAssignedDirectly={groupsAssignedDirectly}
              pageRef={pageRef}
              rolesAssignableToScope={rolesAssignableToScope}
              usersAssignedDirectly={usersAssignedDirectly}
              workspace={workspace}
              onFilterUpdate={handleFilterUpdate}
            />
          </Tabs.TabPane>
        </Tabs>
      ) : (
        <WorkspaceProjects id={id} pageRef={pageRef} workspace={workspace} />
      )}
    </Page>
  );
};

export default WorkspaceDetails;
