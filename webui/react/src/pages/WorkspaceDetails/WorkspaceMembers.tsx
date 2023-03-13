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
import { V1Group, V1GroupDetails, V1Role, V1RoleWithAssignments } from 'services/api-ts-sdk';
import Icon from 'shared/components/Icon/Icon';
import { ValueOf } from 'shared/types';
import { alphaNumericSorter } from 'shared/utils/sort';
import { User, UserOrGroup, Workspace } from 'types';
import { getAssignedRole, getIdFromUserOrGroup, getName, isUser } from 'utils/user';

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
  userOrGroup: UserOrGroup;
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
    userOrGroupId: getIdFromUserOrGroup(userOrGroup),
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

  const usersAndGroups: UserOrGroup[] = [...usersAssignedDirectly, ...groupsAssignedDirectly];

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

  const generateTableKey = useCallback(
    (record: UserOrGroup) =>
      isUser(record)
        ? `user-${getIdFromUserOrGroup(record)}`
        : `group-${getIdFromUserOrGroup(record)}`,
    [],
  );

  const columns = useMemo(() => {
    const nameRenderer = (value: string, record: UserOrGroup) => {
      if (isUser(record)) {
        const member = record as User;
        return <UserBadge user={member} />;
      }
      const group = record as V1GroupDetails;
      return <GroupAvatar groupName={group.name} />;
    };

    const roleRenderer = (value: string, record: UserOrGroup) => (
      <RoleRenderer
        assignments={assignments}
        rolesAssignableToScope={rolesAssignableToScope}
        userCanAssignRoles={userCanAssignRoles}
        userOrGroup={record}
        workspaceId={workspace.id}
      />
    );

    const actionRenderer = (value: string, record: UserOrGroup) => {
      const assignedRole = getAssignedRole(record, assignments);

      return userCanAssignRoles && assignedRole?.role.roleId ? (
        <GroupOrMemberActionDropdown
          fetchMembers={fetchMembers}
          name={getName(record)}
          roleIds={[assignedRole.role.roleId]}
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
        sorter: (a: UserOrGroup, b: UserOrGroup) => alphaNumericSorter(getName(a), getName(b)),
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
    ] as ColumnDef<UserOrGroup>[];
  }, [
    assignments,
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
          dataSource={usersAndGroups}
          pagination={getFullPaginationConfig(
            {
              limit: settings.tableLimit,
              offset: settings.tableOffset,
            },
            usersAndGroups.length,
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
