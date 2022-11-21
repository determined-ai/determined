import { Button, Dropdown, Menu } from 'antd';
import { FilterDropdownProps } from 'antd/lib/table/interface';
import React, { useCallback, useEffect, useMemo } from 'react';

import InteractiveTable, {
  ColumnDef,
  InteractiveTableSettings,
} from 'components/Table/InteractiveTable';
import { getFullPaginationConfig } from 'components/Table/Table';
import TableFilterSearch from 'components/Table/TableFilterSearch';
import Avatar from 'components/UserAvatar';
import useFeature from 'hooks/useFeature';
import useModalWorkspaceRemoveMember from 'hooks/useModal/Workspace/useModalWorkspaceRemoveMember';
import usePermissions from 'hooks/usePermissions';
import useSettings, { UpdateSettings } from 'hooks/useSettings';
import { V1Group, V1GroupDetails, V1Role, V1RoleWithAssignments } from 'services/api-ts-sdk/models';
import { Size } from 'shared/components/Avatar';
import Icon from 'shared/components/Icon/Icon';
import { alphaNumericSorter } from 'shared/utils/sort';
import { User, UserOrGroup, Workspace } from 'types';
import { getAssignedRole, getIdFromUserOrGroup, getName, isUser } from 'utils/user';

import RoleRenderer from './RoleRenderer';
import css from './WorkspaceMembers.module.scss';
import settingsConfig, {
  DEFAULT_COLUMN_WIDTHS,
  WorkspaceMembersSettings,
} from './WorkspaceMembers.settings';

interface Props {
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
  roleId: number;
  userOrGroup: UserOrGroup;
  workspace: Workspace;
}

const GroupOrMemberActionDropdown: React.FC<GroupOrMemberActionDropdownProps> = ({
  name,
  roleId,
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
    roleIds: [roleId],
    scopeWorkspaceId: workspace.id,
    userOrGroup,
    userOrGroupId: getIdFromUserOrGroup(userOrGroup),
  });

  const menuItems = (
    <Menu>
      <Menu.Item danger key="remove" onClick={() => openWorkspaceRemoveMemberModal()}>
        Remove
      </Menu.Item>
      {openWorkspaceRemoveMemberContextHolder}
    </Menu>
  );

  return (
    <div>
      <Dropdown overlay={menuItems} placement="bottomRight" trigger={['click']}>
        <Button type="text">
          <Icon name="overflow-vertical" />
        </Button>
      </Dropdown>
    </div>
  );
};

const WorkspaceMembers: React.FC<Props> = ({
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
  const { settings, updateSettings } = useSettings<WorkspaceMembersSettings>(settingsConfig);
  const userCanAssignRoles = canAssignRoles({ workspace });

  const mockWorkspaceMembers = useFeature().isOn('mock_workspace_members');

  const usersAndGroups: UserOrGroup[] = useMemo(
    () =>
      mockWorkspaceMembers
        ? [
            {
              displayName: 'Test User One Display Name',
              id: 1,
              username: 'TestUserOneUserName',
            },
            {
              id: 2,
              username: 'TestUserTwoUserName',
            },
            {
              groupId: 1,
              name: 'Test Group 1 Name',
            },
            {
              groupId: 2,
              name: 'Test Group 2 Name',
            },
          ]
        : [...usersAssignedDirectly, ...groupsAssignedDirectly],
    [groupsAssignedDirectly, mockWorkspaceMembers, usersAssignedDirectly],
  );
  if (mockWorkspaceMembers) {
    assignments = [
      {
        groupRoleAssignments: [
          {
            groupId: 1,
            roleAssignment: {
              role: { roleId: 1 },
            },
          },
          {
            groupId: 2,
            roleAssignment: {
              role: { roleId: 1 },
            },
          },
        ],
        userRoleAssignments: [
          {
            roleAssignment: {
              role: { roleId: 1 },
            },
            userId: 1,
          },
          {
            roleAssignment: {
              role: { roleId: 1 },
            },
            userId: 2,
          },
        ],
      },
    ];
  }

  useEffect(() => {
    onFilterUpdate(settings.name);
  }, [onFilterUpdate, settings.name]);

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
        return (
          <>
            <div className={css.userAvatarRowItem}>
              <Avatar size={Size.Medium} userId={member.id} />
            </div>
            <div className={css.userRowItem}>
              {member?.displayName ? (
                <>
                  <div>{member.displayName}</div>
                  <div>{member.username}</div>
                </>
              ) : (
                <div>{member.username}</div>
              )}
            </div>
          </>
        );
      }
      const group = record as V1GroupDetails;
      return (
        <>
          <div className={css.userAvatarRowItem}>
            <Icon name="group" />
          </div>
          <div className={css.userRowItem}>
            <div>{group.name}</div>
          </div>
        </>
      );
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
          roleId={assignedRole.role.roleId}
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
        align: 'right',
        dataIndex: 'action',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['action'],
        fixed: 'right',
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
    <div className={css.membersContainer}>
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
        updateSettings={updateSettings as UpdateSettings<InteractiveTableSettings>}
      />
    </div>
  );
};

export default WorkspaceMembers;
