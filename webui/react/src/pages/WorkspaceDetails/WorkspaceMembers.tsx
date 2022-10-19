import { Button, Dropdown, Menu, Select } from 'antd';
import { FilterDropdownProps } from 'antd/lib/table/interface';
import { RawValueType } from 'rc-select/lib/BaseSelect';
import { LabelInValueType } from 'rc-select/lib/Select';
import React, { useCallback, useEffect, useMemo } from 'react';

import InteractiveTable, {
  ColumnDef,
  InteractiveTableSettings,
} from 'components/Table/InteractiveTable';
import { getFullPaginationConfig } from 'components/Table/Table';
import TableFilterSearch from 'components/Table/TableFilterSearch';
import Avatar from 'components/UserAvatar';
import { useStore } from 'contexts/Store';
import useFeature from 'hooks/useFeature';
import useModalWorkspaceRemoveMember from 'hooks/useModal/Workspace/useModalWorkspaceRemoveMember';
import usePermissions from 'hooks/usePermissions';
import useSettings, { UpdateSettings } from 'hooks/useSettings';
import {
  assignRolesToGroup,
  assignRolesToUser,
  removeRoleFromGroup,
  removeRolesFromUser,
} from 'services/api';
import { V1Group, V1GroupDetails, V1RoleWithAssignments } from 'services/api-ts-sdk';
import { Size } from 'shared/components/Avatar';
import Icon from 'shared/components/Icon/Icon';
import { alphaNumericSorter } from 'shared/utils/sort';
import { User, UserOrGroup, Workspace } from 'types';
import handleError from 'utils/error';
import { getIdFromUserOrGroup, getName, isUser } from 'utils/user';

import css from './WorkspaceMembers.module.scss';
import settingsConfig, {
  DEFAULT_COLUMN_WIDTHS,
  WorkspaceMembersSettings,
} from './WorkspaceMembers.settings';

interface Props {
  assignments: V1RoleWithAssignments[];
  groupsAssignedDirectly: V1Group[];
  onFilterUpdate: (name: string | undefined) => void;
  pageRef: React.RefObject<HTMLElement>;
  usersAssignedDirectly: User[];
  workspace: Workspace;
}

interface GroupOrMemberActionDropdownProps {
  name: string;
  userOrGroup: UserOrGroup;
  workspace: Workspace;
}

const GroupOrMemberActionDropdown: React.FC<GroupOrMemberActionDropdownProps> = ({
  userOrGroup,
  name,
}) => {
  const {
    modalOpen: openWorkspaceRemoveMemberModal,
    contextHolder: openWorkspaceRemoveMemberContextHolder,
  } = useModalWorkspaceRemoveMember({
    name,
    userOrGroup: userOrGroup,
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
  workspace,
}: Props) => {
  let { knownRoles } = useStore();

  const { canUpdateRoles } = usePermissions();

  const { settings, updateSettings } = useSettings<WorkspaceMembersSettings>(settingsConfig);
  const userCanAssignRoles = canUpdateRoles({ workspace });

  const mockWorkspaceMembers = useFeature().isOn('mock_workspace_members');

  knownRoles = mockWorkspaceMembers
    ? [
        {
          id: 1,
          name: 'Editor',
          permissions: [],
        },
        {
          id: 2,
          name: 'Viewer',
          permissions: [],
        },
      ]
    : knownRoles;

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

    const roleRenderer = (value: string, record: UserOrGroup) => {
      return (
        <Select
          className={css.selectContainer}
          disabled={!userCanAssignRoles}
          value={mockWorkspaceMembers ? 1 : assignments[0]}
          onSelect={async (value: RawValueType | LabelInValueType) => {
            const roleIdValue = value as number;
            const userOrGroupId = getIdFromUserOrGroup(record);

            // Needs to be updated to get the correct old role for the user
            // or group.
            const oldRoleId = mockWorkspaceMembers ? 1 : assignments?.[0].role?.roleId || 0;

            try {
              // Try to remove the old role and then add the new role
              isUser(record)
                ? await removeRolesFromUser({
                    roleIds: [oldRoleId],
                    userId: userOrGroupId,
                  })
                : await removeRoleFromGroup({
                    groupId: userOrGroupId,
                    roleId: oldRoleId,
                  });
              try {
                isUser(record)
                  ? await assignRolesToUser({
                      roleIds: [roleIdValue],
                      userId: userOrGroupId,
                    })
                  : await assignRolesToGroup({
                      groupId: userOrGroupId,
                      roleIds: [roleIdValue],
                    });
              } catch (addRoleError) {
                handleError(addRoleError, {
                  publicSubject: 'Unable to update role for user or group unable to add new role.',
                });
              }
            } catch (removeRoleError) {
              handleError(removeRoleError, {
                publicSubject:
                  'Unable to update role for user or group could unable to remove current role.',
              });
            }
          }}>
          {knownRoles.map((role) => (
            <Select.Option key={role.id} value={role.id}>
              {role.name}
            </Select.Option>
          ))}
        </Select>
      );
    };

    const actionRenderer = (value: string, record: UserOrGroup) => {
      return userCanAssignRoles ? (
        <GroupOrMemberActionDropdown
          name={getName(record)}
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
    knownRoles,
    mockWorkspaceMembers,
    nameFilterSearch,
    tableSearchIcon,
    userCanAssignRoles,
    workspace,
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
