import type { TabsProps } from 'antd';
import _ from 'lodash';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';

import Pivot from 'components/kit/Pivot';
import Spinner from 'components/kit/Spinner';
import Message from 'components/Message';
import ModelRegistry from 'components/ModelRegistry';
import Page from 'components/Page';
import PageNotFound from 'components/PageNotFound';
import TaskList from 'components/TaskList';
import useFeature from 'hooks/useFeature';
import usePermissions from 'hooks/usePermissions';
import usePolling from 'hooks/usePolling';
import ResourcePoolsBound from 'pages/WorkspaceDetails/ResourcePoolsBound';
import WorkspaceMembers from 'pages/WorkspaceDetails/WorkspaceMembers';
import WorkspaceProjects from 'pages/WorkspaceDetails/WorkspaceProjects';
import { useWorkspaceActionMenu } from 'pages/WorkspaceList/WorkspaceActionDropdown';
import { paths } from 'routes/utils';
import { getGroups, getWorkspaceMembers, searchRolesAssignableToScope } from 'services/api';
import { V1Group, V1GroupSearchResult, V1Role, V1RoleWithAssignments } from 'services/api-ts-sdk';
import determinedStore from 'stores/determinedInfo';
import userStore from 'stores/users';
import workspaceStore from 'stores/workspaces';
import { User, ValueOf } from 'types';
import handleError from 'utils/error';
import { Loadable } from 'utils/loadable';
import { useObservable } from 'utils/observable';

type Params = {
  tab: string;
  workspaceId: string;
};

export const WorkspaceDetailsTab = {
  Members: 'members',
  ModelRegistry: 'models',
  Projects: 'projects',
  ResourcePools: 'pools',
  Tasks: 'tasks',
} as const;

export type WorkspaceDetailsTab = ValueOf<typeof WorkspaceDetailsTab>;

const WorkspaceDetails: React.FC = () => {
  const { rbacEnabled } = useObservable(determinedStore.info);
  const rpBindingFlagOn = useFeature().isOn('rp_binding');
  const loadableUsers = useObservable(userStore.getUsers());
  const users = Loadable.getOrElse([], loadableUsers);
  const { tab, workspaceId: workspaceID } = useParams<Params>();
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
  const [canceler] = useState(new AbortController());
  const [tabKey, setTabKey] = useState<WorkspaceDetailsTab>(
    (tab as WorkspaceDetailsTab) || WorkspaceDetailsTab.Projects,
  );
  const pageRef = useRef<HTMLElement>(null);
  const workspaceId = workspaceID ?? '';
  const id = Number(workspaceId);
  const navigate = useNavigate();
  const { canViewWorkspace, canViewModelRegistry, loading: rbacLoading } = usePermissions();

  const loadableWorkspace = useObservable(workspaceStore.getWorkspace(id));
  const workspace = Loadable.getOrElse(undefined, loadableWorkspace);

  const fetchGroups = useCallback(async (): Promise<void> => {
    try {
      const response = await getGroups({ limit: 100 }, { signal: canceler.signal });

      setGroups((prev) => {
        if (_.isEqual(prev, response.groups)) return prev;
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
        if (_.isEqual(prev, response.roles)) return prev;
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
    () => users.filter((user) => !usersAssignedDirectlyIds.has(user.id)),
    [users, usersAssignedDirectlyIds],
  );
  const addableUsersAndGroups = useMemo(
    () => [...addableGroups, ...addableUsers],
    [addableGroups, addableUsers],
  );

  const { contextHolders, menu, onClick } = useWorkspaceActionMenu({
    onComplete: () => workspaceStore.fetch(undefined, true),
    workspace: workspace || undefined,
  });

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

    if (rpBindingFlagOn && canViewWorkspace({ workspace })) {
      items.push({
        children: <ResourcePoolsBound workspace={workspace} />,
        key: WorkspaceDetailsTab.ResourcePools,
        label: 'Resource Pools',
      });
    }

    return items;
  }, [
    addableUsersAndGroups,
    canViewModelRegistry,
    canViewWorkspace,
    fetchGroupsAndUsersAssignedToWorkspace,
    groupsAssignedDirectly,
    id,
    rbacEnabled,
    rolesAssignableToScope,
    usersAssignedDirectly,
    workspace,
    workspaceAssignments,
    rpBindingFlagOn,
  ]);

  const canViewWorkspaceFlag = canViewWorkspace({ workspace: { id } });
  const fetchAll = useCallback(async () => {
    if (!canViewWorkspaceFlag) return;
    await Promise.allSettled([
      fetchGroups(),
      fetchGroupsAndUsersAssignedToWorkspace(),
      fetchRolesAssignableToScope(),
    ]);
  }, [
    canViewWorkspaceFlag,
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

  if (Loadable.isLoading(loadableWorkspace) || Loadable.isLoading(loadableUsers)) {
    return <Spinner spinning tip={`Loading workspace ${workspaceId} details...`} />;
  } else if (isNaN(id)) {
    return <Message title={`Invalid Workspace ID ${workspaceId}`} />;
  } else if ((!rbacLoading && !canViewWorkspaceFlag) || workspace === null) {
    return <PageNotFound />;
  }

  const breadcrumb = [
    {
      breadcrumbName: 'Workspaces',
      path: paths.workspaceList(),
    },
  ];
  if (workspace) {
    breadcrumb.push({
      breadcrumbName: id !== 1 ? workspace.name : 'Uncategorized Experiments',
      path: paths.workspaceDetails(id),
    });
  }

  return (
    <Page
      breadcrumb={breadcrumb}
      containerRef={pageRef}
      id="workspaceDetails"
      key={workspaceId}
      menuItems={menu.length > 0 ? menu : undefined}
      onClickMenu={onClick}>
      <Pivot
        activeKey={tabKey}
        destroyInactiveTabPane
        items={tabItems}
        onChange={handleTabChange}
      />
      {contextHolders}
    </Page>
  );
};

export default WorkspaceDetails;
