import { Space } from 'antd';
import { SortOrder } from 'antd/es/table/interface';
import { FilterDropdownProps } from 'antd/lib/table/interface';
import _ from 'lodash';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import dropdownCss from 'components/ActionDropdown/ActionDropdown.module.scss';
import ChangeUserStatusModalComponent from 'components/ChangeUserStatusModal';
import ConfigureAgentModalComponent from 'components/ConfigureAgentModal';
import CreateUserModalComponent from 'components/CreateUserModal';
import Button from 'components/kit/Button';
import Dropdown, { MenuItem } from 'components/kit/Dropdown';
import Icon from 'components/kit/Icon';
import { useModal } from 'components/kit/Modal';
import { makeToast } from 'components/kit/Toast';
import { Loadable } from 'components/kit/utils/loadable';
import ManageGroupsModalComponent from 'components/ManageGroupsModal';
import Section from 'components/Section';
import InteractiveTable, { onRightClickableCell } from 'components/Table/InteractiveTable';
import SkeletonTable from 'components/Table/SkeletonTable';
import {
  checkmarkRenderer,
  defaultRowClassName,
  relativeTimeRenderer,
} from 'components/Table/Table';
import TableFilterSearch from 'components/Table/TableFilterSearch';
import UserBadge from 'components/UserBadge';
import usePermissions from 'hooks/usePermissions';
import { useSettings } from 'hooks/useSettings';
import { getGroups, patchUsers } from 'services/api';
import { V1GetUsersRequestSortBy, V1GroupSearchResult } from 'services/api-ts-sdk';
import determinedStore from 'stores/determinedInfo';
import roleStore from 'stores/roles';
import userStore from 'stores/users';
import { DetailedUser } from 'types';
import handleError, { ErrorType } from 'utils/error';
import { useObservable } from 'utils/observable';
import { validateDetApiEnum } from 'utils/service';
import { alphaNumericSorter, booleanSorter, numericSorter } from 'utils/sort';

import css from './UserManagement.module.scss';
import settingsConfig, {
  DEFAULT_COLUMN_WIDTHS,
  DEFAULT_COLUMNS,
  UserColumnName,
  UserManagementSettings,
} from './UserManagement.settings';

export const USER_TITLE = 'Users';
export const CREATE_USER = 'Add User';
export const CREATE_USER_LABEL = 'add_user';

interface DropdownProps {
  fetchUsers: () => void;
  groups: V1GroupSearchResult[];
  user: DetailedUser;
  userManagementEnabled: boolean;
}

const MenuKey = {
  Agent: 'agent',
  Edit: 'edit',
  Groups: 'groups',
  State: 'state',
  View: 'view',
} as const;

const ActionMenuKey = {
  AddToGroups: 'add-to-groups',
  ChangeStatus: 'change-status',
  SetRoles: 'set-roles',
} as const;

const UserActionDropdown = ({ fetchUsers, user, groups, userManagementEnabled }: DropdownProps) => {
  const EditUserModal = useModal(CreateUserModalComponent);
  const ViewUserModal = useModal(CreateUserModalComponent);
  const ManageGroupsModal = useModal(ManageGroupsModalComponent);
  const ConfigureAgentModal = useModal(ConfigureAgentModalComponent);
  const [selectedUserGroups, setSelectedUserGroups] = useState<V1GroupSearchResult[]>();

  const { canModifyUsers } = usePermissions();
  const { rbacEnabled } = useObservable(determinedStore.info);

  const onToggleActive = useCallback(async () => {
    try {
      await patchUsers({ activate: !user.isActive, userIds: [user.id] });
      makeToast({
        severity: 'Confirm',
        title: `User has been ${user.isActive ? 'deactivated' : 'activated'}`,
      });
      fetchUsers();
    } catch (e) {
      handleError(e, {
        isUserTriggered: true,
        publicSubject: `Unable to ${user.isActive ? 'deactivate' : 'activate'} user.`,
        silent: false,
        type: ErrorType.Api,
      });
    }
  }, [fetchUsers, user]);

  const menuItems =
    userManagementEnabled && canModifyUsers
      ? rbacEnabled
        ? [
            { key: MenuKey.Edit, label: 'Edit User' },
            { key: MenuKey.Groups, label: 'Manage Groups' },
            { key: MenuKey.Agent, label: 'Configure Agent' },
            { key: MenuKey.State, label: `${user.isActive ? 'Deactivate' : 'Activate'}` },
          ]
        : [
            { key: MenuKey.Edit, label: 'Edit User' },
            { key: MenuKey.Agent, label: 'Configure Agent' },
            { key: MenuKey.State, label: `${user.isActive ? 'Deactivate' : 'Activate'}` },
          ]
      : [{ key: MenuKey.View, label: 'View User' }];

  const handleDropdown = useCallback(
    async (key: string) => {
      switch (key) {
        case MenuKey.Agent:
          ConfigureAgentModal.open();
          break;
        case MenuKey.Edit:
          EditUserModal.open();
          break;
        case MenuKey.Groups: {
          const response = await getGroups({ limit: 500, userId: user.id });
          setSelectedUserGroups(response.groups ?? []);
          ManageGroupsModal.open();
          break;
        }
        case MenuKey.State:
          await onToggleActive();
          break;
        case MenuKey.View:
          ViewUserModal.open();
          break;
      }
    },
    [ConfigureAgentModal, EditUserModal, ManageGroupsModal, onToggleActive, user, ViewUserModal],
  );

  return (
    <div className={dropdownCss.base}>
      <Dropdown menu={menuItems} placement="bottomRight" onClick={handleDropdown}>
        <Button icon={<Icon name="overflow-vertical" size="small" title="Action menu" />} />
      </Dropdown>
      <ViewUserModal.Component user={user} viewOnly onClose={fetchUsers} />
      <EditUserModal.Component user={user} onClose={fetchUsers} />
      <ManageGroupsModal.Component
        groupOptions={groups}
        user={user}
        userGroups={selectedUserGroups ?? []}
      />
      <ConfigureAgentModal.Component user={user} onClose={fetchUsers} />
    </div>
  );
};

const UserManagement: React.FC = () => {
  const [groups, setGroups] = useState<V1GroupSearchResult[]>([]);
  const [selectedUserIds, setSelectedUserIds] = useState<React.Key[]>([]);
  const pageRef = useRef<HTMLElement>(null);
  const { settings, updateSettings } = useSettings<UserManagementSettings>(settingsConfig);

  const loadableUsers = useObservable(userStore.getUsers());
  const users = Loadable.getOrElse([], loadableUsers);

  const nameRegex = useMemo(() => {
    if (settings.name === undefined) return new RegExp('.*');
    const escapedName = settings.name?.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    return new RegExp(escapedName, 'i');
  }, [settings.name]);
  const filteredUsers = users.filter((user) => nameRegex.test(user.displayName || user.username));

  const { rbacEnabled } = useObservable(determinedStore.info);
  const { canModifyUsers } = usePermissions();
  const info = useObservable(determinedStore.info);
  const ChangeUserStatusModal = useModal(ChangeUserStatusModalComponent);

  const canceler = useRef(new AbortController());
  const fetchUsers = useCallback((): void => {
    if (!settings) return;

    userStore.fetchUsers(canceler.current.signal);
  }, [settings]);

  const fetchGroups = useCallback(async (): Promise<void> => {
    try {
      const response = await getGroups({ limit: 500 }, { signal: canceler.current.signal });

      setGroups((prev) => {
        if (_.isEqual(prev, response.groups)) return prev;
        return response.groups || [];
      });
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch groups.' });
    }
  }, []);

  useEffect(() => {
    const currentCanceler = canceler.current;
    return () => currentCanceler.abort();
  }, []);

  useEffect(() => {
    fetchGroups();
  }, [fetchGroups]);

  useEffect(() => (rbacEnabled ? roleStore.fetch() : undefined), [rbacEnabled]);

  const CreateUserModal = useModal(CreateUserModalComponent);

  const handleNameSearchApply = useCallback(
    (name: string) => {
      updateSettings({ name: name || undefined, row: undefined, tableOffset: 0 });
    },
    [updateSettings],
  );

  const handleNameSearchReset = useCallback(() => {
    updateSettings({ name: undefined, row: undefined, tableOffset: 0 });
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

  const handleTableRowSelect = useCallback((rowKeys: React.Key[]) => {
    setSelectedUserIds(rowKeys);
  }, []);

  const actionDropdownMenu: MenuItem[] = useMemo(() => {
    const menuItems: MenuItem[] = [{ key: ActionMenuKey.ChangeStatus, label: 'Change Status' }];

    if (rbacEnabled) {
      menuItems.push(
        { key: ActionMenuKey.SetRoles, label: 'Set Roles' },
        { key: ActionMenuKey.AddToGroups, label: 'Add to Groups' },
      );
    }
    return menuItems;
  }, [rbacEnabled]);

  const handleActionDropdown = useCallback(
    (key: string) => {
      switch (key) {
        case ActionMenuKey.AddToGroups:
          break;
        case ActionMenuKey.ChangeStatus:
          ChangeUserStatusModal.open();
          break;
        case ActionMenuKey.SetRoles:
          break;
      }
    },
    [ChangeUserStatusModal],
  );

  const clearTableSelection = useCallback(() => {
    setSelectedUserIds([]);
  }, []);

  const filterIcon = useCallback(() => <Icon name="search" size="tiny" title="Search" />, []);

  const columns = useMemo(() => {
    const actionRenderer = (_: string, record: DetailedUser) => {
      return (
        <UserActionDropdown
          fetchUsers={fetchUsers}
          groups={groups}
          user={record}
          userManagementEnabled={info.userManagementEnabled}
        />
      );
    };
    const defaultSortKey: V1GetUsersRequestSortBy = validateDetApiEnum(
      V1GetUsersRequestSortBy,
      settings.sortKey,
    );
    const defaultSortOrder: SortOrder = settings.sortDesc ? 'descend' : 'ascend';
    const columns = [
      {
        dataIndex: 'displayName',
        defaultSortOrder:
          defaultSortKey === V1GetUsersRequestSortBy.NAME ? defaultSortOrder : undefined,
        defaultWidth: DEFAULT_COLUMN_WIDTHS['displayName'],
        filterDropdown: nameFilterSearch,
        filterIcon: filterIcon,
        isFiltered: (settings: unknown) => !!(settings as UserManagementSettings)?.name,
        key: V1GetUsersRequestSortBy.NAME,
        onCell: onRightClickableCell,
        render: (_: string, r: DetailedUser) => <UserBadge user={r} />,
        sorter: (a: DetailedUser, b: DetailedUser) => {
          return alphaNumericSorter(a.displayName || a.username, b.displayName || b.username);
        },
        title: 'Name',
      },
      {
        dataIndex: 'isActive',
        defaultSortOrder:
          defaultSortKey === V1GetUsersRequestSortBy.ACTIVE ? defaultSortOrder : undefined,
        defaultWidth: DEFAULT_COLUMN_WIDTHS['isActive'],
        key: V1GetUsersRequestSortBy.ACTIVE,
        onCell: onRightClickableCell,
        render: checkmarkRenderer,
        sorter: (a: DetailedUser, b: DetailedUser) => booleanSorter(a.isActive, b.isActive),
        title: 'Active',
      },
      {
        dataIndex: 'isAdmin',
        defaultSortOrder:
          defaultSortKey === V1GetUsersRequestSortBy.ADMIN ? defaultSortOrder : undefined,
        defaultWidth: DEFAULT_COLUMN_WIDTHS['isAdmin'],
        key: V1GetUsersRequestSortBy.ADMIN,
        onCell: onRightClickableCell,
        render: checkmarkRenderer,
        sorter: (a: DetailedUser, b: DetailedUser) => booleanSorter(a.isAdmin, b.isAdmin),
        title: 'Admin',
      },
      {
        dataIndex: 'modifiedAt',
        defaultSortOrder:
          defaultSortKey === V1GetUsersRequestSortBy.MODIFIEDTIME ? defaultSortOrder : undefined,
        defaultWidth: DEFAULT_COLUMN_WIDTHS['modifiedAt'],
        key: V1GetUsersRequestSortBy.MODIFIEDTIME,
        onCell: onRightClickableCell,
        render: (value: number): React.ReactNode => relativeTimeRenderer(new Date(value)),
        sorter: (a: DetailedUser, b: DetailedUser) => numericSorter(a.modifiedAt, b.modifiedAt),
        title: 'Modified Time',
      },
      {
        dataIndex: 'lastAuthAt',
        defaultSortOrder:
          defaultSortKey === V1GetUsersRequestSortBy.LASTAUTHTIME ? defaultSortOrder : undefined,
        defaultWidth: DEFAULT_COLUMN_WIDTHS['lastAuthAt'],
        key: V1GetUsersRequestSortBy.LASTAUTHTIME,
        onCell: onRightClickableCell,
        render: (value: number | undefined): React.ReactNode => {
          return value ? (
            relativeTimeRenderer(new Date(value))
          ) : (
            <div className={css.rightAligned}>N/A</div>
          );
        },
        sorter: (a: DetailedUser, b: DetailedUser) => numericSorter(a.lastAuthAt, b.lastAuthAt),
        title: 'Last Seen',
      },
      {
        className: 'fullCell',
        dataIndex: 'action',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['action'],
        key: 'action',
        onCell: onRightClickableCell,
        render: actionRenderer,
        title: '',
        width: DEFAULT_COLUMN_WIDTHS['action'],
      },
    ];
    return rbacEnabled ? columns.filter((c) => c.dataIndex !== 'isAdmin') : columns;
  }, [
    fetchUsers,
    filterIcon,
    groups,
    info.userManagementEnabled,
    nameFilterSearch,
    rbacEnabled,
    settings,
  ]);

  const table = useMemo(() => {
    return settings ? (
      <InteractiveTable<DetailedUser, UserManagementSettings>
        columns={columns}
        containerRef={pageRef}
        dataSource={filteredUsers}
        interactiveColumns={false}
        loading={Loadable.isNotLoaded(loadableUsers)}
        rowClassName={defaultRowClassName({ clickable: false })}
        rowKey="id"
        rowSelection={{
          columnWidth: '20px',
          fixed: true,
          onChange: handleTableRowSelect,
          preserveSelectedRowKeys: false,
          selectedRowKeys: selectedUserIds,
        }}
        settings={{
          ...settings,
          columns: DEFAULT_COLUMNS,
          columnWidths: DEFAULT_COLUMNS.map((col: UserColumnName) => DEFAULT_COLUMN_WIDTHS[col]),
        }}
        showSorterTooltip={false}
        size="small"
        updateSettings={updateSettings}
      />
    ) : (
      <SkeletonTable columns={columns.length} />
    );
  }, [
    settings,
    columns,
    filteredUsers,
    loadableUsers,
    handleTableRowSelect,
    selectedUserIds,
    updateSettings,
  ]);

  return (
    <>
      <Section
        className={css.usersTable}
        options={
          <Space>
            {selectedUserIds.length > 0 && (
              <Dropdown menu={actionDropdownMenu} onClick={handleActionDropdown}>
                <Button>Actions</Button>
              </Dropdown>
            )}
            <Button
              aria-label={CREATE_USER_LABEL}
              disabled={!info.userManagementEnabled || !canModifyUsers}
              onClick={CreateUserModal.open}>
              {CREATE_USER}
            </Button>
            {settings.name && <Button onClick={handleNameSearchReset}>{'Clear Filter'}</Button>}
          </Space>
        }
        title={USER_TITLE}>
        {table}
      </Section>
      <CreateUserModal.Component onClose={fetchUsers} />
      <ChangeUserStatusModal.Component
        clearTableSelection={clearTableSelection}
        fetchUsers={fetchUsers}
        userIds={selectedUserIds.map((id) => Number(id))}
      />
    </>
  );
};

export default UserManagement;
