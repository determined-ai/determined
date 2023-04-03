import { Dropdown, Space } from 'antd';
import type { MenuProps } from 'antd';
import { FilterDropdownProps } from 'antd/lib/table/interface';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import ConfigureAgentModalComponent from 'components/ConfigureAgentModal';
import CreateUserModalComponent from 'components/CreateUserModal';
import Button from 'components/kit/Button';
import { useModal } from 'components/kit/Modal';
import ManageGroupsModalComponent from 'components/ManageGroupsModal';
import Page from 'components/Page';
import Section from 'components/Section';
import InteractiveTable, {
  InteractiveTableSettings,
  onRightClickableCell,
} from 'components/Table/InteractiveTable';
import SkeletonTable from 'components/Table/SkeletonTable';
import {
  checkmarkRenderer,
  defaultRowClassName,
  getFullPaginationConfig,
  relativeTimeRenderer,
} from 'components/Table/Table';
import TableFilterSearch from 'components/Table/TableFilterSearch';
import UserBadge from 'components/UserBadge';
import useFeature from 'hooks/useFeature';
import usePermissions from 'hooks/usePermissions';
import { UpdateSettings, useSettings } from 'hooks/useSettings';
import { getGroups, patchUser } from 'services/api';
import { V1GetUsersRequestSortBy, V1GroupSearchResult } from 'services/api-ts-sdk';
import dropdownCss from 'shared/components/ActionDropdown/ActionDropdown.module.scss';
import Icon from 'shared/components/Icon/Icon';
import { ValueOf } from 'shared/types';
import { validateDetApiEnum } from 'shared/utils/service';
import { initInfo, useDeterminedInfo } from 'stores/determinedInfo';
import { RolesStore } from 'stores/roles';
import usersStore, { FetchUsersConfig } from 'stores/users';
import { DetailedUser } from 'types';
import { message } from 'utils/dialogApi';
import { Loadable, NotLoaded } from 'utils/loadable';
import { useObservable } from 'utils/observable';

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
  user: DetailedUser;
  userManagementEnabled: boolean;
}

const UserActionDropdown = ({ fetchUsers, user, userManagementEnabled }: DropdownProps) => {
  const EditUserModal = useModal(CreateUserModalComponent);
  const ViewUserModal = useModal(CreateUserModalComponent);
  const ManageGroupsModal = useModal(ManageGroupsModalComponent);
  const ConfigureAgentModal = useModal(ConfigureAgentModalComponent);
  const [selectedUserGroups, setSelectedUserGroups] = useState<V1GroupSearchResult[]>();

  const { canModifyUsers } = usePermissions();

  const onToggleActive = async () => {
    await patchUser({ userId: user.id, userParams: { active: !user.isActive } });
    message.success(`User has been ${user.isActive ? 'deactivated' : 'activated'}`);
    fetchUsers();
  };

  const MenuKey = {
    Agent: 'agent',
    Edit: 'edit',
    Groups: 'groups',
    State: 'state',
    View: 'view',
  } as const;

  const funcs = {
    [MenuKey.Edit]: () => {
      EditUserModal.open();
    },
    [MenuKey.State]: () => {
      onToggleActive();
    },
    [MenuKey.View]: () => {
      ViewUserModal.open();
    },
    [MenuKey.Groups]: async () => {
      const response = await getGroups({ userId: user.id });
      setSelectedUserGroups(response.groups ?? []);
      ManageGroupsModal.open();
    },
    [MenuKey.Agent]: () => {
      ConfigureAgentModal.open();
    },
  };

  const onItemClick: MenuProps['onClick'] = (e) => {
    funcs[e.key as ValueOf<typeof MenuKey>]();
  };

  const menuItems: MenuProps['items'] =
    userManagementEnabled && canModifyUsers
      ? [
          { key: MenuKey.Edit, label: 'Edit User' },
          { key: MenuKey.Groups, label: 'Manage Groups' },
          { key: MenuKey.Agent, label: 'Configure Agent' },
          { key: MenuKey.State, label: `${user.isActive ? 'Deactivate' : 'Activate'}` },
        ]
      : [{ key: MenuKey.View, label: 'View User' }];

  return (
    <div className={dropdownCss.base}>
      <Dropdown
        menu={{ items: menuItems, onClick: onItemClick }}
        placement="bottomRight"
        trigger={['click']}>
        <Button ghost icon={<Icon name="overflow-vertical" />} />
      </Dropdown>
      <ViewUserModal.Component user={user} viewOnly onClose={fetchUsers} />
      <EditUserModal.Component user={user} onClose={fetchUsers} />
      <ManageGroupsModal.Component groups={selectedUserGroups ?? []} user={user} />
      <ConfigureAgentModal.Component user={user} onClose={fetchUsers} />
    </div>
  );
};

const UserManagement: React.FC = () => {
  const [canceler] = useState(new AbortController());
  const pageRef = useRef<HTMLElement>(null);
  const { settings, updateSettings } = useSettings<UserManagementSettings>(settingsConfig);
  const apiConfig = useMemo<FetchUsersConfig>(
    () => ({
      limit: settings.tableLimit,
      name: settings.name,
      offset: settings.tableOffset,
      orderBy: settings.sortDesc ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC',
      sortBy: validateDetApiEnum(V1GetUsersRequestSortBy, settings.sortKey),
    }),
    [settings],
  );
  const loadableUsers = useObservable(usersStore.getUsers(apiConfig));
  const users: Readonly<DetailedUser[]> = Loadable.match(loadableUsers, {
    Loaded: (usersPagination) => usersPagination.users,
    NotLoaded: () => [],
  });
  const total = Loadable.match(loadableUsers, {
    Loaded: (users) => users.pagination.total ?? 0,
    NotLoaded: () => 0,
  });
  const rbacEnabled = useFeature().isOn('rbac');
  const { canModifyUsers } = usePermissions();
  const info = Loadable.getOrElse(initInfo, useDeterminedInfo());

  const fetchUsers = useCallback((): void => {
    if (!settings) return;

    usersStore.ensureUsersFetched(canceler, apiConfig, true);
  }, [settings, canceler, apiConfig]);

  useEffect(() => {
    fetchUsers();
  }, [fetchUsers]);

  useEffect(() => {
    if (rbacEnabled) {
      RolesStore.fetchRoles(canceler);
    }
  }, [canceler, rbacEnabled]);

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

  const filterIcon = useCallback(() => <Icon name="search" size="tiny" />, []);

  const columns = useMemo(() => {
    const actionRenderer = (_: string, record: DetailedUser) => {
      return (
        <UserActionDropdown
          fetchUsers={fetchUsers}
          user={record}
          userManagementEnabled={info.userManagementEnabled}
        />
      );
    };
    const columns = [
      {
        dataIndex: 'displayName',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['displayName'],
        filterDropdown: nameFilterSearch,
        filterIcon: filterIcon,
        isFiltered: (settings: unknown) => !!(settings as UserManagementSettings)?.name,
        key: V1GetUsersRequestSortBy.NAME,
        onCell: onRightClickableCell,
        render: (_: string, r: DetailedUser) => <UserBadge user={r} />,
        sorter: true,
        title: 'Name',
      },
      {
        dataIndex: 'isActive',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['isActive'],
        key: V1GetUsersRequestSortBy.ACTIVE,
        onCell: onRightClickableCell,
        render: checkmarkRenderer,
        sorter: true,
        title: 'Active',
      },
      {
        dataIndex: 'isAdmin',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['isAdmin'],
        key: V1GetUsersRequestSortBy.ADMIN,
        onCell: onRightClickableCell,
        render: checkmarkRenderer,
        sorter: true,
        title: 'Admin',
      },
      {
        dataIndex: 'modifiedAt',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['modifiedAt'],
        key: V1GetUsersRequestSortBy.MODIFIEDTIME,
        onCell: onRightClickableCell,
        render: (value: number): React.ReactNode => relativeTimeRenderer(new Date(value)),
        sorter: true,
        title: 'Modified Time',
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
  }, [fetchUsers, filterIcon, info.userManagementEnabled, nameFilterSearch, rbacEnabled]);

  const table = useMemo(() => {
    return settings ? (
      <InteractiveTable
        columns={columns}
        containerRef={pageRef}
        dataSource={users}
        interactiveColumns={false}
        loading={loadableUsers === NotLoaded}
        pagination={getFullPaginationConfig(
          {
            limit: settings.tableLimit,
            offset: settings.tableOffset,
          },
          total,
        )}
        rowClassName={defaultRowClassName({ clickable: false })}
        rowKey="id"
        settings={
          {
            ...settings,
            columns: DEFAULT_COLUMNS,
            columnWidths: DEFAULT_COLUMNS.map((col: UserColumnName) => DEFAULT_COLUMN_WIDTHS[col]),
          } as InteractiveTableSettings
        }
        showSorterTooltip={false}
        size="small"
        updateSettings={updateSettings as UpdateSettings}
      />
    ) : (
      <SkeletonTable columns={columns.length} />
    );
  }, [users, loadableUsers, settings, columns, total, updateSettings]);
  return (
    <Page bodyNoPadding containerRef={pageRef}>
      <Section
        className={css.usersTable}
        options={
          <Space>
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
    </Page>
  );
};

export default UserManagement;
