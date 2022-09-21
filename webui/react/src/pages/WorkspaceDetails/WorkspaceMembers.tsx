import { Button, Dropdown, Menu, Select } from 'antd';
import { FilterDropdownProps } from 'antd/lib/table/interface';
import React, { useCallback, useMemo } from 'react';

import InteractiveTable, { ColumnDef, InteractiveTableSettings } from 'components/InteractiveTable';
import { getFullPaginationConfig } from 'components/Table';
import TableFilterSearch from 'components/TableFilterSearch';
import Avatar from 'components/UserAvatar';
import useModalWorkspaceRemoveMember from 'hooks/useModal/Workspace/useModalWorkspaceRemoveMember';
import usePermissions from 'hooks/usePermissions';
import useSettings, { UpdateSettings } from 'hooks/useSettings';
import { Size } from 'shared/components/Avatar';
import Icon from 'shared/components/Icon/Icon';
import { alphaNumericSorter } from 'shared/utils/sort';
import { DetailedUser, GroupDetailsWithRole, User, UserWithRole, UserOrGroupDetails, Workspace } from 'types';
import {V1Group, V1GroupDetails, V1RoleWithAssignments} from 'services/api-ts-sdk';
import { getName, isUser } from 'utils/user';

import css from './WorkspaceMembers.module.scss';
import settingsConfig, {
  DEFAULT_COLUMN_WIDTHS,
  WorkspaceMembersSettings,
} from './WorkspaceMembers.settings';

const roles = ['Basic', 'Cluster Admin', 'Editor', 'Viewer', 'Restricted', 'Workspace Admin'];
interface Props {
  groups: V1Group[];
  pageRef: React.RefObject<HTMLElement>;
  workspace: Workspace;
}

interface GroupOrMemberActionDropdownProps {
  userOrGroupDetails: UserOrGroupDetails;
  name: string;
  workspace: Workspace;
}

const GroupOrMemberActionDropdown: React.FC<GroupOrMemberActionDropdownProps> = ({
  userOrGroupDetails,
  workspace,
  name,
}) => {
  const {
    modalOpen: openWorkspaceRemoveMemberModal,
    contextHolder: openWorkspaceRemoveMemberContextHolder,
  } = useModalWorkspaceRemoveMember({ userOrGroup: userOrGroupDetails, name, workspace });

  const menuItems = (
    <Menu>
      <Menu.Item danger key="delete" onClick={() => openWorkspaceRemoveMemberModal()}>
        Delete
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

const WorkspaceMembers: React.FC<Props> = ({ pageRef, workspace, groups }: Props) => {

  const users: User[] = [];
  const rolesWithAssignments: V1RoleWithAssignments[] = []

  const usersWithRoles: UserWithRole[] = users.map(user => {
    const userWithRole = user as UserWithRole;
    rolesWithAssignments.forEach(roleWithAssignment => {
      roleWithAssignment?.userRoleAssignments?.forEach(userRole => {
        if( userRole.userId === user.id && roleWithAssignment.role) userWithRole.role = roleWithAssignment.role;
    })});
    return userWithRole;
  });

  const groupsWithRoles: GroupDetailsWithRole[] = groups.map(group => {
    const groupDetailsWithRole = group as GroupDetailsWithRole;
    rolesWithAssignments.forEach(roleWithAssignment => {
      roleWithAssignment?.groupRoleAssignments?.forEach(groupRole => {
        if( groupRole.groupId === group.groupId && roleWithAssignment.role) groupDetailsWithRole.role = roleWithAssignment.role;
    })});
    return groupDetailsWithRole;
  });

  const usersAndGroupDetails: UserOrGroupDetails[] = [...usersWithRoles, ...groupsWithRoles];

  const { canUpdateRoles } = usePermissions();
  const { settings, updateSettings } = useSettings<WorkspaceMembersSettings>(settingsConfig);
  const userCanAssignRoles = canUpdateRoles({ workspace });

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

  const columns = useMemo(() => {
    const nameRenderer = (value: string, record: UserOrGroupDetails) => {
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

    const roleRenderer = (value: string, record: UserOrGroupDetails) => {
      return (
        <Select
          className={css.selectContainer}
          disabled={!userCanAssignRoles}
          value={record.role}>
          {roles.map((role) => (
            <Select.Option key={role} value={role}>
              {role}
            </Select.Option>
          ))}
        </Select>
      );
    };

    const actionRenderer = (value: string, record: UserOrGroupDetails) => {
      return userCanAssignRoles ? (
        <GroupOrMemberActionDropdown
          userOrGroupDetails={record}
          name={getName(record)}
          workspace={workspace}
        />
      ) : (<></>);
    };

    return [
      {
        dataIndex: 'name',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['name'],
        filterDropdown: nameFilterSearch,
        filterIcon: tableSearchIcon,
        render: nameRenderer,
        sorter: (a: UserOrGroupDetails, b: UserOrGroupDetails) => alphaNumericSorter(getName(a), getName(b)),
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
    ] as ColumnDef<UserOrGroupDetails>[];
  }, [nameFilterSearch, tableSearchIcon, workspace, userCanAssignRoles]);

  const membersAndGroups: UserOrGroupDetails[] = [];

  return (
    <div className={css.membersContainer}>
      <InteractiveTable
        columns={columns}
        containerRef={pageRef}
        dataSource={membersAndGroups}
        pagination={getFullPaginationConfig(
          {
            limit: settings.tableLimit,
            offset: settings.tableOffset,
          },
          membersAndGroups.length,
        )}
        rowKey="id"
        settings={settings}
        showSorterTooltip={false}
        size="small"
        updateSettings={updateSettings as UpdateSettings<InteractiveTableSettings>}
      />
    </div>
  );
};

export default WorkspaceMembers;
