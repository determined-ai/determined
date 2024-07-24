import { SortOrder } from 'antd/es/table/interface';
import Button from 'hew/Button';
import Column from 'hew/Column';
import Dropdown, { MenuItem } from 'hew/Dropdown';
import Icon from 'hew/Icon';
import Input from 'hew/Input';
import { useModal } from 'hew/Modal';
import Row from 'hew/Row';
import Select, { SelectValue } from 'hew/Select';
import { useToast } from 'hew/Toast';
import { Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';
import { debounce } from 'lodash';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import dropdownCss from 'components/ActionDropdown/ActionDropdown.module.scss';
import AddUsersToGroupsModalComponent from 'components/AddUsersToGroupsModal';
import ChangeUserStatusModalComponent from 'components/ChangeUserStatusModal';
import ConfigureAgentModalComponent from 'components/ConfigureAgentModal';
import CreateUserModalComponent from 'components/CreateUserModal';
import ManageGroupsModalComponent from 'components/ManageGroupsModal';
import Section from 'components/Section';
import SetUserRolesModalComponent from 'components/SetUserRolesModal';
import InteractiveTable, { onRightClickableCell } from 'components/Table/InteractiveTable';
import SkeletonTable from 'components/Table/SkeletonTable';
import {
  defaultRowClassName,
  getFullPaginationConfig,
  relativeTimeRenderer,
} from 'components/Table/Table';
import UserBadge from 'components/UserBadge';
import { useAsync } from 'hooks/useAsync';
import usePermissions from 'hooks/usePermissions';
import { getGroups, getUserRoles, getUsers, patchUsers } from 'services/api';
import { V1GetUsersRequestSortBy, V1GroupSearchResult, V1OrderBy } from 'services/api-ts-sdk';
import determinedStore from 'stores/determinedInfo';
import roleStore from 'stores/roles';
import userStore from 'stores/users';
import userSettings from 'stores/userSettings';
import { DetailedUser, UserRole as UserRoleType } from 'types';
import handleError, { ErrorType } from 'utils/error';
import { useObservable } from 'utils/observable';
import { validateDetApiEnum } from 'utils/service';
import { alphaNumericSorter, booleanSorter, numericSorter } from 'utils/sort';

import css from './UserManagement.module.scss';
import {
  DEFAULT_COLUMN_WIDTHS,
  DEFAULT_COLUMNS,
  DEFAULT_SETTINGS,
  UserManagementSettings,
  UserRole,
  UserStatus,
} from './UserManagement.settings';

export const CREATE_USER = 'Add User';
export const CREATE_USER_LABEL = 'add_user';

interface DropdownProps {
  fetchUsers: () => void;
  groups: V1GroupSearchResult[];
  user: DetailedUser;
  patchUserEnabled: boolean;
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

const UserActionDropdown = ({ fetchUsers, user, groups, patchUserEnabled }: DropdownProps) => {
  const EditUserModal = useModal(CreateUserModalComponent);
  const ViewUserModal = useModal(CreateUserModalComponent);
  const ManageGroupsModal = useModal(ManageGroupsModalComponent);
  const ConfigureAgentModal = useModal(ConfigureAgentModalComponent);
  const [selectedUserGroups, setSelectedUserGroups] = useState<V1GroupSearchResult[]>();
  const { openToast } = useToast();
  const { canModifyUsers, canAssignRoles } = usePermissions();
  const [userRoles, setUserRoles] = useState<Loadable<UserRoleType[]>>(NotLoaded);
  const canAssignRolesFlag: boolean = canAssignRoles({});
  const { rbacEnabled } = useObservable(determinedStore.info);

  const fetchUserRoles = useCallback(async () => {
    if (user !== undefined && rbacEnabled && canAssignRolesFlag) {
      try {
        const roles = await getUserRoles({ userId: user.id });
        setUserRoles(Loaded(roles?.filter((r) => r.fromUser ?? false)));
      } catch (e) {
        handleError(e, { publicSubject: "Unable to fetch this user's roles." });
      }
    }
  }, [user, canAssignRolesFlag, rbacEnabled]);

  const onToggleActive = useCallback(async () => {
    try {
      await patchUsers({ activate: !user.isActive, userIds: [user.id] });
      openToast({
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
  }, [fetchUsers, openToast, user]);

  const menuItems = useMemo(() => {
    if (!canModifyUsers) return [{ key: MenuKey.View, label: 'View User' }];

    const items: MenuItem[] = [{ key: MenuKey.Edit, label: 'Edit User' }];

    if (rbacEnabled) items.push({ key: MenuKey.Groups, label: 'Manage Groups' });
    if (patchUserEnabled)
      items.push(
        { key: MenuKey.Agent, label: 'Link with Agent UID/GID' },
        { key: MenuKey.State, label: `${user.isActive ? 'Deactivate' : 'Activate'}` },
      );
    return items;
  }, [canModifyUsers, rbacEnabled, user.isActive, patchUserEnabled]);

  const handleDropdown = useCallback(
    async (key: string) => {
      switch (key) {
        case MenuKey.Agent:
          ConfigureAgentModal.open();
          break;
        case MenuKey.Edit:
          await fetchUserRoles();
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
    [
      ConfigureAgentModal,
      fetchUserRoles,
      EditUserModal,
      onToggleActive,
      ViewUserModal,
      user.id,
      ManageGroupsModal,
    ],
  );

  return (
    <div className={dropdownCss.base}>
      <Dropdown menu={menuItems} placement="bottomRight" onClick={handleDropdown}>
        <Button icon={<Icon name="overflow-vertical" size="small" title="Action menu" />} />
      </Dropdown>
      <ViewUserModal.Component user={user} userRoles={userRoles} viewOnly onClose={fetchUsers} />
      <EditUserModal.Component user={user} userRoles={userRoles} onClose={fetchUsers} />
      <ManageGroupsModal.Component
        groupOptions={groups}
        user={user}
        userGroups={selectedUserGroups ?? []}
      />
      <ConfigureAgentModal.Component user={user} onClose={fetchUsers} />
    </div>
  );
};

const roleOptions = [
  { label: 'All Roles', value: '' },
  { label: 'Admin', value: UserRole.ADMIN },
  { label: 'Non-Admin', value: UserRole.MEMBER },
];

const statusOptions = [
  { label: 'All Statuses', value: '' },
  { label: 'Active Users', value: UserStatus.ACTIVE },
  { label: 'Deactivated Users', value: UserStatus.INACTIVE },
];
type UserManagementSettingsWithColumns = UserManagementSettings & {
  columns: string[];
  columnWidths: number[];
};

const columnSettings = {
  columns: DEFAULT_COLUMNS,
  columnWidths: DEFAULT_COLUMNS.map((c) => DEFAULT_COLUMN_WIDTHS[c]),
};

const userManagementSettings = userSettings.get(UserManagementSettings, 'user-management');
const UserManagement: React.FC = () => {
  const [selectedUserIds, setSelectedUserIds] = useState<React.Key[]>([]);
  const [refresh, setRefresh] = useState<Record<string, never>>({});
  const [nameFilter, setNameFilter] = useState<string>('');
  const [roleFilter, setRoleFilter] = useState<UserRole | number[]>();
  const [statusFilter, setStatusFilter] = useState<UserStatus>();
  const pageRef = useRef<HTMLElement>(null);
  const currentUser = Loadable.getOrElse(undefined, useObservable(userStore.currentUser));
  const loadableSettings = useObservable(userManagementSettings);
  const settings = useMemo(() => {
    return Loadable.match(loadableSettings, {
      _: () => ({ ...DEFAULT_SETTINGS, ...columnSettings }),
      Loaded: (s) => ({ ...DEFAULT_SETTINGS, ...s, ...columnSettings }),
    });
  }, [loadableSettings]);
  const updateSettings = useCallback(
    (p: Partial<UserManagementSettings>) =>
      userSettings.setPartial(UserManagementSettings, 'user-management', p),
    [],
  );

  const userResponse = useAsync(async () => {
    try {
      const params = {
        active: (statusFilter || undefined) && statusFilter === UserStatus.ACTIVE,
        limit: settings.tableLimit,
        name: nameFilter,
        offset: settings.tableOffset,
        orderBy: settings.sortDesc ? V1OrderBy.DESC : V1OrderBy.ASC,
        sortBy: settings.sortKey,
      };
      const roleParams = Array.isArray(roleFilter)
        ? {
            roleIdAssignedDirectlyToUser: roleFilter,
          }
        : {
            admin: (roleFilter || undefined) && roleFilter === UserRole.ADMIN,
          };
      return await getUsers({
        ...params,
        ...roleParams,
      });
    } catch (e) {
      handleError(e, { publicSubject: 'Could not fetch user search results' });
      return NotLoaded;
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [nameFilter, roleFilter, statusFilter, settings, refresh]);

  const users = useMemo(
    () =>
      Loadable.match(userResponse, {
        _: () => [],
        Loaded: (r) => r.users,
      }),
    [userResponse],
  );

  const { rbacEnabled } = useObservable(determinedStore.info);
  const { canModifyUsers, canModifyPermissions } = usePermissions();
  const info = useObservable(determinedStore.info);
  const ChangeUserStatusModal = useModal(ChangeUserStatusModalComponent);
  const SetUserRolesModal = useModal(SetUserRolesModalComponent);
  const AddUsersToGroupsModal = useModal(AddUsersToGroupsModalComponent);

  const fetchUsers = useCallback((): void => {
    if (!settings) return;

    setRefresh({});
  }, [settings, setRefresh]);

  const groupsResponse = useAsync(async (canceler) => {
    try {
      return await getGroups({ limit: 500 }, { signal: canceler.signal });
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch groups.' });
      return NotLoaded;
    }
  }, []);
  const groups = useMemo(
    () =>
      Loadable.match(groupsResponse, {
        _: () => [],
        Loaded: (g) => g.groups || [],
      }),
    [groupsResponse],
  );

  useEffect(() => (rbacEnabled ? roleStore.fetch() : undefined), [rbacEnabled]);

  useEffect(() => {
    // reset invalid settings
    Loadable.forEach(loadableSettings, (s) => s || updateSettings(DEFAULT_SETTINGS));
  }, [loadableSettings, updateSettings]);

  const CreateUserModal = useModal(CreateUserModalComponent);

  const handleNameSearchApply = useMemo(
    () =>
      debounce((e: React.ChangeEvent<HTMLInputElement>) => {
        setNameFilter(e.target.value);
        updateSettings({ row: undefined, tableOffset: 0 });
      }, 500),
    [updateSettings],
  );

  const handleStatusFilterApply = useCallback(
    (statusFilter?: SelectValue) => {
      setStatusFilter(statusFilter as UserStatus);
      updateSettings({ row: undefined, tableOffset: 0 });
    },
    [updateSettings],
  );

  const handleRoleFilterApply = useCallback(
    (roleFilter?: SelectValue) => {
      setRoleFilter(roleFilter as UserRole | number[]);
      updateSettings({ row: undefined, tableOffset: 0 });
    },
    [updateSettings],
  );

  const handleTableRowSelect = useCallback((rowKeys: React.Key[]) => {
    setSelectedUserIds(rowKeys);
  }, []);

  const actionDropdownMenu: MenuItem[] = useMemo(() => {
    const menuItems: MenuItem[] = [{ key: ActionMenuKey.ChangeStatus, label: 'Change Status' }];

    if (rbacEnabled) {
      if (canModifyPermissions) {
        menuItems.push({ key: ActionMenuKey.SetRoles, label: 'Set Roles' });
      }
      if (canModifyUsers) {
        menuItems.push({ key: ActionMenuKey.AddToGroups, label: 'Add to Groups' });
      }
    }

    return menuItems;
  }, [rbacEnabled, canModifyPermissions, canModifyUsers]);

  const handleActionDropdown = useCallback(
    (key: string) => {
      switch (key) {
        case ActionMenuKey.AddToGroups:
          AddUsersToGroupsModal.open();
          break;
        case ActionMenuKey.ChangeStatus:
          ChangeUserStatusModal.open();
          break;
        case ActionMenuKey.SetRoles:
          SetUserRolesModal.open();
          break;
      }
    },
    [AddUsersToGroupsModal, ChangeUserStatusModal, SetUserRolesModal],
  );

  const clearTableSelection = useCallback(() => {
    setSelectedUserIds([]);
  }, []);

  const columns = useMemo(() => {
    const actionRenderer = (_: string, record: DetailedUser) => {
      return (
        <UserActionDropdown
          fetchUsers={fetchUsers}
          groups={groups}
          patchUserEnabled={info.patchUserEnabled}
          user={record}
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
        key: V1GetUsersRequestSortBy.NAME,
        onCell: () => ({ ...onRightClickableCell(), 'data-testid': 'user' }),
        render: (_: string, r: DetailedUser) => <UserBadge user={r} />,
        sorter: (a: DetailedUser, b: DetailedUser) => {
          return alphaNumericSorter(a.displayName || a.username, b.displayName || b.username);
        },
        title: 'User',
      },
      {
        dataIndex: 'isActive',
        defaultSortOrder:
          defaultSortKey === V1GetUsersRequestSortBy.ACTIVE ? defaultSortOrder : undefined,
        defaultWidth: DEFAULT_COLUMN_WIDTHS['isActive'],
        key: V1GetUsersRequestSortBy.ACTIVE,
        onCell: () => ({ ...onRightClickableCell(), 'data-testid': 'status' }),
        render: (isActive: boolean) => <>{isActive ? 'Active' : 'Inactive'}</>,
        sorter: (a: DetailedUser, b: DetailedUser) => booleanSorter(b.isActive, a.isActive),
        title: 'Status',
      },
      {
        dataIndex: 'isAdmin',
        defaultSortOrder:
          defaultSortKey === V1GetUsersRequestSortBy.ADMIN ? defaultSortOrder : undefined,
        defaultWidth: DEFAULT_COLUMN_WIDTHS['isAdmin'],
        key: V1GetUsersRequestSortBy.ADMIN,
        onCell: () => ({ ...onRightClickableCell(), 'data-testid': 'role' }),
        render: (isAdmin: boolean) => <>{isAdmin ? 'Admin' : 'Member'}</>,
        sorter: (a: DetailedUser, b: DetailedUser) => booleanSorter(a.isAdmin, b.isAdmin),
        title: 'Role',
      },
      {
        dataIndex: 'remote',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['remote'],
        key: V1GetUsersRequestSortBy.REMOTE,
        onCell: () => ({ ...onRightClickableCell(), 'data-testid': 'remote' }),
        render: (value: boolean): React.ReactNode => (value ? 'Remote' : 'Local'),
        sorter: (a: DetailedUser, b: DetailedUser) => booleanSorter(b.remote, a.remote),
        title: 'Remote',
      },
      {
        dataIndex: 'modifiedAt',
        defaultSortOrder:
          defaultSortKey === V1GetUsersRequestSortBy.MODIFIEDTIME ? defaultSortOrder : undefined,
        defaultWidth: DEFAULT_COLUMN_WIDTHS['modifiedAt'],
        key: V1GetUsersRequestSortBy.MODIFIEDTIME,
        onCell: () => ({ ...onRightClickableCell(), 'data-testid': 'modified' }),
        render: (value: number): React.ReactNode => relativeTimeRenderer(new Date(value)),
        sorter: (a: DetailedUser, b: DetailedUser) => numericSorter(a.modifiedAt, b.modifiedAt),
        title: 'Modified',
      },
      {
        dataIndex: 'lastAuthAt',
        defaultSortOrder:
          defaultSortKey === V1GetUsersRequestSortBy.LASTAUTHTIME ? defaultSortOrder : undefined,
        defaultWidth: DEFAULT_COLUMN_WIDTHS['lastAuthAt'],
        key: V1GetUsersRequestSortBy.LASTAUTHTIME,
        onCell: () => ({ ...onRightClickableCell(), 'data-testid': 'lastSeen' }),
        render: (value: number | undefined): React.ReactNode => {
          return value ? relativeTimeRenderer(new Date(value)) : 'N/A';
        },
        sorter: (a: DetailedUser, b: DetailedUser) => numericSorter(a.lastAuthAt, b.lastAuthAt),
        title: 'Last Seen',
      },
      {
        className: 'fullCell',
        dataIndex: 'action',
        defaultWidth: 46,
        key: 'action',
        onCell: () => ({ ...onRightClickableCell(), 'data-testid': 'actions' }),
        render: actionRenderer,
        title: '',
      },
    ];
    return rbacEnabled
      ? columns.filter((c) => c.dataIndex !== 'isAdmin')
      : columns.filter((c) => c.dataIndex !== 'remote');
  }, [fetchUsers, groups, info.patchUserEnabled, rbacEnabled, settings]);

  return (
    <>
      <Section className={css.usersTable}>
        <div className={css.actionBar} data-testid="actionRow">
          <Row>
            <Column>
              <Row>
                {/* input is uncontrolled */}
                <Input
                  allowClear
                  data-testid="search"
                  defaultValue={nameFilter}
                  placeholder="Find user"
                  prefix={<Icon color="cancel" decorative name="search" size="tiny" />}
                  onChange={handleNameSearchApply}
                />
                <Select
                  data-testid="roleSelect"
                  options={roleOptions}
                  searchable={false}
                  value={roleFilter}
                  width={120}
                  onChange={handleRoleFilterApply}
                />
                <Select
                  data-testid="statusSelect"
                  options={statusOptions}
                  searchable={false}
                  value={statusFilter}
                  width={170}
                  onChange={handleStatusFilterApply}
                />
              </Row>
            </Column>
            <Column align="right">
              <Row>
                {selectedUserIds.length > 0 && (
                  <Dropdown menu={actionDropdownMenu} onClick={handleActionDropdown}>
                    <Button data-testid="actions">Actions</Button>
                  </Dropdown>
                )}
                <Button
                  aria-label={CREATE_USER_LABEL}
                  data-testid="addUser"
                  disabled={!info.patchUserEnabled || !canModifyUsers}
                  onClick={CreateUserModal.open}>
                  {CREATE_USER}
                </Button>
              </Row>
            </Column>
          </Row>
        </div>
        {settings ? (
          <InteractiveTable<DetailedUser, UserManagementSettingsWithColumns>
            columns={columns}
            containerRef={pageRef}
            dataSource={users}
            interactiveColumns={false}
            loading={Loadable.isNotLoaded(userResponse)}
            pagination={Loadable.match(userResponse, {
              _: () => undefined,
              Loaded: (r) =>
                getFullPaginationConfig(
                  {
                    limit: settings.tableLimit,
                    offset: settings.tableOffset,
                  },
                  r.pagination.total || 0,
                ),
            })}
            rowClassName={defaultRowClassName({ clickable: false })}
            rowKey="id"
            rowSelection={{
              columnWidth: '20px',
              fixed: true,
              getCheckboxProps: (record) => ({
                disabled: record.id === currentUser?.id, // disable the current user not to select onself
              }),
              onChange: handleTableRowSelect,
              preserveSelectedRowKeys: false,
              selectedRowKeys: selectedUserIds,
            }}
            settings={settings}
            showSorterTooltip={false}
            size="small"
            updateSettings={updateSettings}
          />
        ) : (
          <SkeletonTable columns={columns.length} />
        )}
      </Section>
      <CreateUserModal.Component onClose={fetchUsers} />
      <ChangeUserStatusModal.Component
        clearTableSelection={clearTableSelection}
        fetchUsers={fetchUsers}
        userIds={selectedUserIds.map((id) => Number(id))}
      />
      <SetUserRolesModal.Component
        clearTableSelection={clearTableSelection}
        fetchUsers={fetchUsers}
        userIds={selectedUserIds.map((id) => Number(id))}
      />
      <AddUsersToGroupsModal.Component
        clearTableSelection={clearTableSelection}
        fetchUsers={fetchUsers}
        groupOptions={groups}
        userIds={selectedUserIds.map((id) => Number(id))}
      />
    </>
  );
};

export default UserManagement;
