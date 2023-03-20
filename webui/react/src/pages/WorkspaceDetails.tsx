import type { TabsProps } from 'antd';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';

import Pivot from 'components/kit/Pivot';
import Page from 'components/Page';
import PageNotFound from 'components/PageNotFound';
import TaskList from 'components/TaskList';
import useFeature from 'hooks/useFeature';
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
import usersStore from 'stores/users';
import { User, Workspace } from 'types';
import handleError from 'utils/error';
import { Loadable } from 'utils/loadable';
import { useObservable } from 'utils/observable';

import ModelRegistry from './ModelRegistry';
import WorkspaceDetailsHeader from './WorkspaceDetails/WorkspaceDetailsHeader';
import WorkspaceMembers from './WorkspaceDetails/WorkspaceMembers';
import WorkspaceProjects from './WorkspaceDetails/WorkspaceProjects';
import css from './WorkspaceDetails.module.scss';

type Params = {
  tab: string;
  workspaceId: string;
};

export const WorkspaceDetailsTab = {
  Members: 'members',
  ModelRegistry: 'models',
  Projects: 'projects',
  Tasks: 'tasks',
} as const;

export type WorkspaceDetailsTab = ValueOf<typeof WorkspaceDetailsTab>;

const WorkspaceDetails: React.FC = () => {
  const rbacEnabled = useFeature().isOn('rbac');

  const loadableUsers = useObservable(usersStore.getUsers());
  const users = Loadable.map(loadableUsers, ({ users }) => users);
  const { tab, workspaceId: workspaceID } = useParams<Params>();
  const [workspace, setWorkspace] = useState<Workspace | undefined>();
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
  const id = Number(workspaceId);
  const navigate = useNavigate();
  const { canViewWorkspace, canViewModelRegistry, loading: rbacLoading } = usePermissions();

  const fetchWorkspace = useCallback(async () => {
    try {
      const response = await getWorkspace({ id }, { signal: canceler.signal });
      setWorkspace(response);
    } catch (e) {
      if (!pageError) setPageError(e as Error);
    }
  }, [canceler.signal, id, pageError]);

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
    if (!rbacEnabled) return;

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
  }, [id, nameFilter, rbacEnabled]);

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
      handleError(e, { silent: true });
    }
  }, [canceler.signal, id, rbacEnabled]);

  const handleFilterUpdate = (name: string | undefined) => setNameFilter(name);

  // Users and Groups that are not already a part of the workspace
  const addableGroups: V1Group[] = useMemo(
    () =>
      groups
        ? groups
            .map((groupDetails) => groupDetails.group)
            .filter((group) => group.groupId && !groupsAssignedDirectlyIds.has(group.groupId))
        : [],
    [groups, groupsAssignedDirectlyIds],
  );

  const addableUsers = useMemo(
    () =>
      Loadable.isLoaded(users)
        ? users.data.filter((user) => !usersAssignedDirectlyIds.has(user.id))
        : [],
    [users, usersAssignedDirectlyIds],
  );
  const addableUsersAndGroups = useMemo(
    () => [...addableGroups, ...addableUsers],
    [addableGroups, addableUsers],
  );

  const tabItems: TabsProps['items'] = useMemo(() => {
    if (!workspace) {
      return [];
    }

    const items: TabsProps['items'] = [
      {
        children: <WorkspaceProjects id={id} pageRef={pageRef} workspace={workspace} />,
        key: WorkspaceDetailsTab.Projects,
        label: 'Projects',
      },
      {
        children: <TaskList workspace={workspace} />,
        key: WorkspaceDetailsTab.Tasks,
        label: 'Tasks',
      },
    ];

    if (rbacEnabled) {
      items.push({
        children: (
          <WorkspaceMembers
            addableUsersAndGroups={addableUsersAndGroups}
            assignments={workspaceAssignments}
            fetchMembers={fetchGroupsAndUsersAssignedToWorkspace}
            groupsAssignedDirectly={groupsAssignedDirectly}
            pageRef={pageRef}
            rolesAssignableToScope={rolesAssignableToScope}
            usersAssignedDirectly={usersAssignedDirectly}
            workspace={workspace}
            onFilterUpdate={handleFilterUpdate}
          />
        ),
        key: WorkspaceDetailsTab.Members,
        label: 'Members',
      });
    }

    if (canViewModelRegistry({ workspace })) {
      items.push({
        children: <ModelRegistry workspace={workspace} />,
        key: WorkspaceDetailsTab.ModelRegistry,
        label: 'Model Registry',
      });
    }

    return items;
  }, [
    addableUsersAndGroups,
    canViewModelRegistry,
    fetchGroupsAndUsersAssignedToWorkspace,
    groupsAssignedDirectly,
    id,
    rbacEnabled,
    rolesAssignableToScope,
    usersAssignedDirectly,
    workspace,
    workspaceAssignments,
  ]);

  const canViewWorkspaceFlag = canViewWorkspace({ workspace: { id } });
  const fetchAll = useCallback(async () => {
    if (!canViewWorkspaceFlag) return;
    await Promise.allSettled([
      fetchWorkspace(),
      usersStore.ensureUsersFetched(canceler),
      fetchGroups(),
      fetchGroupsAndUsersAssignedToWorkspace(),
      fetchRolesAssignableToScope(),
    ]);
  }, [
    canViewWorkspaceFlag,
    fetchWorkspace,
    canceler,
    fetchGroups,
    fetchGroupsAndUsersAssignedToWorkspace,
    fetchRolesAssignableToScope,
  ]);

  usePolling(fetchAll, { rerunOnNewFn: true });

  const handleTabChange = useCallback(
    (activeTab: string) => {
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

  useEffect(() => {
    return () => canceler.abort();
  }, [canceler]);

  if (!workspace || Loadable.isLoading(users)) {
    return <Spinner spinning tip={`Loading workspace ${workspaceId} details...`} />;
  } else if (isNaN(id)) {
    return <Message title={`Invalid Workspace ID ${workspaceId}`} />;
  } else if (pageError && !isNotFound(pageError)) {
    const message = `Unable to fetch Workspace ${workspaceId}`;
    return <Message title={message} type={MessageType.Warning} />;
  } else if ((!rbacLoading && !canViewWorkspaceFlag) || (pageError && isNotFound(pageError))) {
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
      id="workspaceDetails"
      key={workspaceId}>
      <Pivot
        activeKey={tabKey}
        destroyInactiveTabPane
        items={tabItems}
        onChange={handleTabChange}
      />
    </Page>
  );
};

export default WorkspaceDetails;
