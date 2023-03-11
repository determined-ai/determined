import { Dropdown } from 'antd';
import type { DropDownProps, MenuProps } from 'antd';
import { FilterDropdownProps } from 'antd/lib/table/interface';
import React, { useCallback, useEffect, useMemo } from 'react';

import GroupAvatar from 'components/GroupAvatar';
import Button from 'components/kit/Button';
import UserBadge from 'components/kit/UserBadge';
import InteractiveTable, { ColumnDef } from 'components/Table/InteractiveTable';
import SkeletonTable from 'components/Table/SkeletonTable';
import { getFullPaginationConfig } from 'components/Table/Table';
import TableFilterSearch from 'components/Table/TableFilterSearch';
import useFeature from 'hooks/useFeature';
import useModalWorkspaceAddMember from 'hooks/useModal/Workspace/useModalWorkspaceAddMember';
import useModalWorkspaceRemoveMember from 'hooks/useModal/Workspace/useModalWorkspaceRemoveMember';
import usePermissions from 'hooks/usePermissions';
import { UpdateSettings, useSettings } from 'hooks/useSettings';
import { V1Group, V1Role, V1RoleWithAssignments } from 'services/api-ts-sdk';
import Icon from 'shared/components/Icon/Icon';
import { ValueOf } from 'shared/types';
import { alphaNumericSorter } from 'shared/utils/sort';
import {
  GroupWithRoleInfo,
  User,
  UserOrGroup,
  UserOrGroupWithRoleInfo,
  UserWithRoleInfo,
  Workspace,
} from 'types';
import { isUserWithRoleInfo } from 'utils/user';

import RoleRenderer from './RoleRenderer';
import css from './WorkspaceMembers.module.scss';
import {
  configForWorkspace,
  DEFAULT_COLUMN_WIDTHS,
  WorkspaceMembersSettings,
} from './WorkspaceMembers.settings';

interface Props {
  addableUsersAndGroups: UserOrGroup[];
  assignments: V1RoleWithAssignments[];
  fetchMembers: () => void;
  groupsAssignedDirectly: V1Group[];
  onFilterUpdate: (name: string | undefined) => void;
  pageRef: React.RefObject<HTMLElement>;
  rolesAssignableToScope: V1Role[];
  usersAssignedDirectly: User[];
  workspace: Workspace;
}

interface GroupOrMemberActionDropdownProps {
  fetchMembers: () => void;
  name: string;
  roleIds: number[];
  userOrGroup: UserOrGroupWithRoleInfo;
  workspace: Workspace;
}

const GroupOrMemberActionDropdown: React.FC<GroupOrMemberActionDropdownProps> = ({
  name,
  roleIds,
  userOrGroup,
  workspace,
  fetchMembers,
}) => {
  const {
    modalOpen: openWorkspaceRemoveMemberModal,
    contextHolder: openWorkspaceRemoveMemberContextHolder,
  } = useModalWorkspaceRemoveMember({
    name,
    onClose: fetchMembers,
    roleIds,
    scopeWorkspaceId: workspace.id,
    userOrGroup,
    userOrGroupId: isUserWithRoleInfo(userOrGroup) ? userOrGroup.userId : userOrGroup.groupId ?? 0,
  });

  const menuItems: DropDownProps['menu'] = useMemo(() => {
    const MenuKey = {
      Remove: 'remove',
    } as const;

    const funcs = {
      [MenuKey.Remove]: () => {
        openWorkspaceRemoveMemberModal();
      },
    };

    const onItemClick: MenuProps['onClick'] = (e) => {
      funcs[e.key as ValueOf<typeof MenuKey>]();
    };

    return {
      items: [{ danger: true, key: MenuKey.Remove, label: 'Remove' }],
      onClick: onItemClick,
    };
  }, [openWorkspaceRemoveMemberModal]);

  return (
    <div className={css.dropdown}>
      <Dropdown menu={menuItems} placement="bottomRight" trigger={['click']}>
        <Button icon={<Icon name="overflow-vertical" />} type="text" />
      </Dropdown>
      {openWorkspaceRemoveMemberContextHolder}
    </div>
  );
};

const WorkspaceMembers: React.FC<Props> = ({
  addableUsersAndGroups,
  assignments,
  onFilterUpdate,
  usersAssignedDirectly,
  groupsAssignedDirectly,
  pageRef,
  rolesAssignableToScope,
  workspace,
  fetchMembers,
}: Props) => {
  const { canAssignRoles } = usePermissions();
  const config = useMemo(() => configForWorkspace(workspace.id), [workspace.id]);
  const { settings, updateSettings } = useSettings<WorkspaceMembersSettings>(config);
  const userCanAssignRoles = canAssignRoles({ workspace });

  const UserOrGroupWithRoles: UserOrGroupWithRoleInfo[] = ((): UserOrGroupWithRoleInfo[] => {
    const groupsAndUsers: [
      V1RoleWithAssignments['groupRoleAssignments'],
      V1RoleWithAssignments['userRoleAssignments'],
    ][] = assignments.map((assignment: V1RoleWithAssignments) => {
      return [assignment.groupRoleAssignments, assignment.userRoleAssignments];
    });
    const groups: GroupWithRoleInfo[] = groupsAndUsers
      .flatMap((data) => data?.[0] ?? [])
      .map((d) => {
        const groupnfo = groupsAssignedDirectly.find((g) => g.groupId === d?.groupId) as V1Group;
        const groupWithRole: GroupWithRoleInfo = {
          groupId: groupnfo.groupId,
          groupName: groupnfo.name,
          roleAssignment: d.roleAssignment,
        };
        return groupWithRole;
      });
    const users: UserWithRoleInfo[] = groupsAndUsers
      .flatMap((data) => data?.[1] ?? [])
      .map((d) => {
        const userInfo = usersAssignedDirectly.find((u) => u.id === d?.userId) as User;
        const groupWithRole: UserWithRoleInfo = {
          displayName: userInfo.displayName,
          roleAssignment: d.roleAssignment,
          userId: userInfo.id,
          username: userInfo.username,
        };
        return groupWithRole;
      });
    return [...groups, ...users];
  })();

  const { contextHolder: workspaceAddMemberContextHolder, modalOpen: openWorkspaceAddMember } =
    useModalWorkspaceAddMember({
      addableUsersAndGroups,
      onClose: fetchMembers,
      rolesAssignableToScope,
      workspaceId: workspace.id,
    });

  const rbacEnabled = useFeature().isOn('rbac');

  useEffect(() => {
    onFilterUpdate(settings.name);
  }, [onFilterUpdate, settings.name]);

  const handleAddMembersClick = useCallback(() => {
    openWorkspaceAddMember();
  }, [openWorkspaceAddMember]);

  const handleNameSearchApply = useCallback(
    (newSearch: string) => {
      updateSettings({ name: newSearch || undefined });
    },
    [updateSettings],
  );

  const handleNameSearchReset = useCallback(() => {
    updateSettings({ name: undefined });
  }, [updateSettings]);

  const nameFilterSearch = useCallback(
    (filterProps: FilterDropdownProps) => (
      <TableFilterSearch
        {...filterProps}
        value={settings.name || ''}
        onReset={handleNameSearchReset}
        onSearch={handleNameSearchApply}
      />
    ),
    [handleNameSearchApply, handleNameSearchReset, settings.name],
  );

  const tableSearchIcon = useCallback(() => <Icon name="search" size="tiny" />, []);

  const generateTableKey = useCallback((record: UserOrGroupWithRoleInfo) => {
    const roleId = record.roleAssignment.role.roleId;
    return isUserWithRoleInfo(record)
      ? `user-${record.userId}-${roleId}`
      : `group-${record.groupId}-${roleId}`;
  }, []);

  const columns = useMemo(() => {
    const nameRenderer = (value: string, record: UserOrGroupWithRoleInfo) => {
      return isUserWithRoleInfo(record) ? (
        <UserBadge user={record} />
      ) : (
        <GroupAvatar groupName={record.groupName} />
      );
    };

    const roleRenderer = (value: string, record: UserOrGroupWithRoleInfo) => (
      <RoleRenderer
        rolesAssignableToScope={rolesAssignableToScope}
        userCanAssignRoles={userCanAssignRoles}
        userOrGroup={record}
        workspaceId={workspace.id}
      />
    );

    const actionRenderer = (value: string, record: Readonly<UserOrGroupWithRoleInfo>) => {
      return userCanAssignRoles ? (
        <GroupOrMemberActionDropdown
          fetchMembers={fetchMembers}
          name={record.roleAssignment.role.name ?? ''}
          roleIds={[record.roleAssignment.role.roleId]}
          userOrGroup={record}
          workspace={workspace}
        />
      ) : (
        <></>
      );
    };

    return [
      {
        dataIndex: 'name',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['name'],
        filterDropdown: nameFilterSearch,
        filterIcon: tableSearchIcon,
        render: nameRenderer,
        sorter: (a: UserOrGroupWithRoleInfo, b: UserOrGroupWithRoleInfo) => {
          if (isUserWithRoleInfo(a) && isUserWithRoleInfo(b)) {
            return alphaNumericSorter(a.username, b.username);
          }
          return 0;
        },
        title: 'Name',
      },
      {
        dataIndex: 'role',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['role'],
        render: roleRenderer,
        title: 'Role',
      },
      {
        dataIndex: 'action',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['action'],
        render: actionRenderer,
        title: '',
      },
    ] as ColumnDef<UserOrGroupWithRoleInfo>[];
  }, [
    nameFilterSearch,
    rolesAssignableToScope,
    tableSearchIcon,
    userCanAssignRoles,
    workspace,
    fetchMembers,
  ]);

  return (
    <>
      <div className={css.headerButton}>
        {rbacEnabled &&
          canAssignRoles({ workspace }) &&
          !workspace.immutable &&
          !workspace.archived && <Button onClick={handleAddMembersClick}> Add Members</Button>}
      </div>
      {settings ? (
        <InteractiveTable
          columns={columns}
          containerRef={pageRef}
          dataSource={UserOrGroupWithRoles}
          pagination={getFullPaginationConfig(
            { limit: settings.tableLimit, offset: settings.tableOffset },
            UserOrGroupWithRoles.length,
          )}
          rowKey={generateTableKey}
          settings={settings}
          showSorterTooltip={false}
          size="small"
          updateSettings={updateSettings as UpdateSettings}
        />
      ) : (
        <SkeletonTable columns={columns.length} />
      )}
      {workspaceAddMemberContextHolder}
    </>
  );
};

export default WorkspaceMembers;
