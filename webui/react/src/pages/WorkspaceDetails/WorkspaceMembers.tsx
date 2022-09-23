import { Button, Dropdown, Menu, Select } from 'antd';
import { BaseOptionType } from 'antd/lib/select';
import { FilterDropdownProps } from 'antd/lib/table/interface';
import { RawValueType } from 'rc-select/lib/BaseSelect';
import { LabelInValueType } from 'rc-select/lib/Select';
import React, { useCallback, useMemo } from 'react';

import InteractiveTable, { ColumnDef, InteractiveTableSettings } from 'components/InteractiveTable';
import { getFullPaginationConfig } from 'components/Table';
import TableFilterSearch from 'components/TableFilterSearch';
import Avatar from 'components/UserAvatar';
import { useStore } from 'contexts/Store';
import useModalWorkspaceRemoveMember from 'hooks/useModal/Workspace/useModalWorkspaceRemoveMember';
import usePermissions from 'hooks/usePermissions';
import useSettings, { UpdateSettings } from 'hooks/useSettings';
import { assignRoles, removeAssignments } from 'services/api';
import { V1Group, V1GroupDetails, V1RoleWithAssignments } from 'services/api-ts-sdk';
import { Size } from 'shared/components/Avatar';
import Icon from 'shared/components/Icon/Icon';
import { alphaNumericSorter } from 'shared/utils/sort';
import { User, UserOrGroup, Workspace } from 'types';
import { createAssignmentRequest, getIdFromUserOrGroup, getName, isUser } from 'utils/user';

import css from './WorkspaceMembers.module.scss';
import settingsConfig, {
  DEFAULT_COLUMN_WIDTHS,
  WorkspaceMembersSettings,
} from './WorkspaceMembers.settings';

interface Props {
  assignments: V1RoleWithAssignments[];
  groupsAssignedDirectly: V1Group[];
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
  workspace,
  name,
}) => {
  const {
    modalOpen: openWorkspaceRemoveMemberModal,
    contextHolder: openWorkspaceRemoveMemberContextHolder,
  } = useModalWorkspaceRemoveMember({
    name,
    userOrGroup: userOrGroup,
    userOrGroupId: getIdFromUserOrGroup(userOrGroup),
    workspaceId: workspace.id,
  });

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

const WorkspaceMembers: React.FC<Props> = ({
  assignments,
  usersAssignedDirectly,
  groupsAssignedDirectly,
  pageRef,
  workspace,
}: Props) => {
  const { knownRoles } = useStore();

  const { canUpdateRoles } = usePermissions();

  const { settings, updateSettings } = useSettings<WorkspaceMembersSettings>(settingsConfig);
  const userCanAssignRoles = canUpdateRoles({ workspace });

  const usersAndGroups: UserOrGroup[] = useMemo(
    () => [...usersAssignedDirectly, ...groupsAssignedDirectly],
    [groupsAssignedDirectly, usersAssignedDirectly],
  );

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
          value={assignments[0]}
          onSelect={async (value: RawValueType | LabelInValueType, option: BaseOptionType) => {
            const assignmentToRemove = createAssignmentRequest(
              record,
              getIdFromUserOrGroup(record),
              0,
              workspace.id,
            );
            const AssignmentToAdd = createAssignmentRequest(
              record,
              getIdFromUserOrGroup(record),
              1,
              workspace.id,
            );
            // Remove the old role
            await removeAssignments(assignmentToRemove);

            // Add the new role
            await assignRoles(AssignmentToAdd);
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
  }, [assignments, knownRoles, nameFilterSearch, tableSearchIcon, userCanAssignRoles, workspace]);

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
