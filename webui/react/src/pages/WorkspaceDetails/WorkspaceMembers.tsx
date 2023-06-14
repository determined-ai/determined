import { Space } from 'antd';
import { FilterDropdownProps } from 'antd/lib/table/interface';
import React, { useCallback, useEffect, useMemo } from 'react';

import Button from 'components/kit/Button';
import Dropdown from 'components/kit/Dropdown';
import Icon from 'components/kit/Icon';
import { useModal } from 'components/kit/Modal';
import Nameplate from 'components/kit/Nameplate';
import InteractiveTable, { ColumnDef } from 'components/Table/InteractiveTable';
import SkeletonTable from 'components/Table/SkeletonTable';
import { getFullPaginationConfig } from 'components/Table/Table';
import TableFilterSearch from 'components/Table/TableFilterSearch';
import UserBadge from 'components/UserBadge';
import WorkspaceMemberAddModalComponent from 'components/WorkspaceMemberAddModal';
import WorkspaceMemberRemoveComponent from 'components/WorkspaceMemberRemoveModal';
import usePermissions from 'hooks/usePermissions';
import { useSettings } from 'hooks/useSettings';
import { V1Group, V1Role, V1RoleWithAssignments } from 'services/api-ts-sdk';
import determinedStore from 'stores/determinedInfo';
import { User, UserOrGroup, UserOrGroupWithRoleInfo, Workspace } from 'types';
import { useObservable } from 'utils/observable';
import { alphaNumericSorter } from 'utils/sort';
import { getUserOrGroupWithRoleInfo, isUserWithRoleInfo } from 'utils/user';

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

const MenuKey = {
  Remove: 'Remove',
} as const;

const DROPDOWN_MENU = [{ danger: true, key: MenuKey.Remove, label: MenuKey.Remove }];

const GroupOrMemberActionDropdown: React.FC<GroupOrMemberActionDropdownProps> = ({
  name,
  roleIds,
  userOrGroup,
  workspace,
  fetchMembers,
}) => {
  const WorkspaceMemberRemoveModal = useModal(WorkspaceMemberRemoveComponent);

  const handleDropdown = useCallback(
    (key: string) => {
      switch (key) {
        case MenuKey.Remove:
          WorkspaceMemberRemoveModal.open();
          break;
      }
    },
    [WorkspaceMemberRemoveModal],
  );

  return (
    <div className={css.dropdown}>
      <Dropdown menu={DROPDOWN_MENU} placement="bottomRight" onClick={handleDropdown}>
        <Button icon={<Icon name="overflow-vertical" title="Action menu" />} type="text" />
      </Dropdown>
      <WorkspaceMemberRemoveModal.Component
        {...{
          name,
          onClose: fetchMembers,
          roleIds,
          scopeWorkspaceId: workspace.id,
          userOrGroup,
          userOrGroupId: isUserWithRoleInfo(userOrGroup)
            ? userOrGroup.userId
            : userOrGroup.groupId ?? 0,
        }}
      />
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

  const userOrGroupWithRoles = getUserOrGroupWithRoleInfo(
    assignments,
    groupsAssignedDirectly,
    usersAssignedDirectly,
  );

  const WorkspaceMemberAddModal = useModal(WorkspaceMemberAddModalComponent);

  const { rbacEnabled } = useObservable(determinedStore.info);

  useEffect(() => {
    onFilterUpdate(settings.name);
  }, [onFilterUpdate, settings.name]);

  const handleNameSearchApply = useCallback(
    (newSearch: string) => {
      updateSettings({ name: newSearch || undefined, tableOffset: 0 });
    },
    [updateSettings],
  );

  const handleNameSearchReset = useCallback(() => {
    updateSettings({ name: undefined, tableOffset: 0 });
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

  const tableSearchIcon = useCallback(() => <Icon name="search" size="tiny" title="Search" />, []);

  const generateTableKey = useCallback((record: Readonly<UserOrGroupWithRoleInfo>) => {
    const roleId = record.roleAssignment.role.roleId;
    return isUserWithRoleInfo(record)
      ? `user-${record.userId}-${roleId}`
      : `group-${record.groupId}-${roleId}`;
  }, []);

  const columns = useMemo(() => {
    const nameRenderer = (value: string, record: Readonly<UserOrGroupWithRoleInfo>) => {
      if (isUserWithRoleInfo(record)) {
        const member: User = {
          displayName: record.displayName,
          id: record.userId,
          username: record.username,
        };
        return <UserBadge user={member} />;
      }
      return <Nameplate icon={<Icon name="group" title="Group" />} name={record.groupName ?? ''} />;
    };

    const roleRenderer = (value: string, record: Readonly<UserOrGroupWithRoleInfo>) => (
      <RoleRenderer
        fetchMembers={fetchMembers}
        rolesAssignableToScope={rolesAssignableToScope}
        userCanAssignRoles={userCanAssignRoles}
        userOrGroupWithRoleInfo={record}
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
        isFiltered: (settings: unknown) => !!(settings as WorkspaceMembersSettings)?.name,
        key: 'name',
        render: nameRenderer,
        sorter: (a: Readonly<UserOrGroupWithRoleInfo>, b: Readonly<UserOrGroupWithRoleInfo>) => {
          const aName = isUserWithRoleInfo(a) ? a.displayName || a.username : a.groupName ?? '';
          const bName = isUserWithRoleInfo(b) ? b.displayName || b.username : b.groupName ?? '';
          return alphaNumericSorter(aName, bName);
        },
        title: 'Name',
      },
      {
        dataIndex: 'role',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['role'],
        key: 'role',
        render: roleRenderer,
        title: 'Role',
      },
      {
        dataIndex: 'action',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['action'],
        key: 'action',
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
        <Space>
          {rbacEnabled &&
            canAssignRoles({ workspace }) &&
            !workspace.immutable &&
            !workspace.archived && (
              <Button onClick={WorkspaceMemberAddModal.open}> Add Member</Button>
            )}
          {settings.name && <Button onClick={handleNameSearchReset}>{'Clear Filter'}</Button>}
        </Space>
      </div>
      {settings ? (
        <InteractiveTable<UserOrGroupWithRoleInfo, WorkspaceMembersSettings>
          columns={columns}
          containerRef={pageRef}
          dataSource={userOrGroupWithRoles}
          pagination={getFullPaginationConfig(
            { limit: settings.tableLimit, offset: settings.tableOffset },
            userOrGroupWithRoles.length,
          )}
          rowKey={generateTableKey}
          settings={settings}
          showSorterTooltip={false}
          size="small"
          updateSettings={updateSettings}
        />
      ) : (
        <SkeletonTable columns={columns.length} />
      )}
      <WorkspaceMemberAddModal.Component
        {...{
          addableUsersAndGroups,
          onClose: fetchMembers,
          rolesAssignableToScope,
          workspaceId: workspace.id,
        }}
      />
    </>
  );
};

export default WorkspaceMembers;
