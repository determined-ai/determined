import { Button, Dropdown, message, Space } from 'antd';
import type { MenuProps } from 'antd';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import Page from 'components/Page';
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
import UserBadge from 'components/UserBadge';
import useFeature from 'hooks/useFeature';
import useModalCreateUser from 'hooks/useModal/UserSettings/useModalCreateUser';
import usePermissions from 'hooks/usePermissions';
import { UpdateSettings, useSettings } from 'hooks/useSettings';
import { getGroups, patchUser } from 'services/api';
import { V1GetUsersRequestSortBy, V1GroupSearchResult } from 'services/api-ts-sdk';
import dropdownCss from 'shared/components/ActionDropdown/ActionDropdown.module.scss';
import Icon from 'shared/components/Icon/Icon';
import { ValueOf } from 'shared/types';
import { isEqual } from 'shared/utils/data';
import { validateDetApiEnum } from 'shared/utils/service';
import { useFetchKnownRoles } from 'stores/knowRoles';
import { FetchUsersConfig, useFetchUsers, useUsers } from 'stores/users';
import { DetailedUser } from 'types';
import handleError from 'utils/error';
import { Loadable, NotLoaded } from 'utils/loadable';

import css from './UserManagement.module.scss';
import settingsConfig, {
  DEFAULT_COLUMN_WIDTHS,
  UserManagementSettings,
} from './UserManagement.settings';

export const USER_TITLE = 'Users';
export const CREATE_USER = 'New User';
export const CREAT_USER_LABEL = 'new_user';

interface DropdownProps {
  fetchUsers: () => void;
  groups: V1GroupSearchResult[];
  user: DetailedUser;
}

const UserActionDropdown = ({ fetchUsers, user, groups }: DropdownProps) => {
  const { modalOpen: openEditUserModal, contextHolder: modalEditUserContextHolder } =
    useModalCreateUser({ groups, onClose: fetchUsers, user });

  const { canModifyUsers } = usePermissions();

  const onToggleActive = async () => {
    await patchUser({ userId: user.id, userParams: { active: !user.isActive } });
    message.success(`User has been ${user.isActive ? 'deactivated' : 'activated'}`);
    fetchUsers();
  };

  const MenuKey = {
    Edit: 'edit',
    State: 'state',
    View: 'view',
  } as const;

  const funcs = {
    [MenuKey.Edit]: () => {
      openEditUserModal();
    },
    [MenuKey.State]: () => {
      onToggleActive();
    },
    [MenuKey.View]: () => {
      openEditUserModal(true);
    },
  };

  const onItemClick: MenuProps['onClick'] = (e) => {
    funcs[e.key as ValueOf<typeof MenuKey>]();
  };

  const menuItems: MenuProps['items'] = canModifyUsers
    ? [
        { key: MenuKey.View, label: 'View Profile' },
        { key: MenuKey.Edit, label: 'Edit' },
        { key: MenuKey.State, label: `${user.isActive ? 'Deactivate' : 'Activate'}` },
      ]
    : [{ key: MenuKey.View, label: 'View Profile' }];

  return (
    <div className={dropdownCss.base}>
      <Dropdown
        menu={{ items: menuItems, onClick: onItemClick }}
        placement="bottomRight"
        trigger={['click']}>
        <Button className={css.overflow} type="text">
          <Icon name="overflow-vertical" />
        </Button>
      </Dropdown>
      {modalEditUserContextHolder}
    </div>
  );
};

const UserManagement: React.FC = () => {
  const [groups, setGroups] = useState<V1GroupSearchResult[]>([]);
  const [canceler] = useState(new AbortController());
  const pageRef = useRef<HTMLElement>(null);
  const fetchUsersHook = useFetchUsers(canceler);
  const { settings, updateSettings } = useSettings<UserManagementSettings>(settingsConfig);
  const apiConfig = useMemo<FetchUsersConfig>(
    () => ({
      limit: settings.tableLimit,
      offset: settings.tableOffset,
      orderBy: settings.sortDesc ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC',
      sortBy: validateDetApiEnum(V1GetUsersRequestSortBy, settings.sortKey),
    }),
    [settings],
  );
  const loadableUser = useUsers(apiConfig);
  const users = Loadable.match(loadableUser, {
    Loaded: (users) => users.users,
    NotLoaded: () => [],
  });
  const total = Loadable.match(loadableUser, {
    Loaded: (users) => users.pagination.total ?? 0,
    NotLoaded: () => 0,
  });

  const rbacEnabled = useFeature().isOn('rbac');
  const { canModifyUsers } = usePermissions();

  const fetchKnownRoles = useFetchKnownRoles(canceler);

  const fetchUsers = useCallback((): void => {
    if (!settings) return;

    fetchUsersHook(apiConfig);
  }, [settings, apiConfig, fetchUsersHook]);

  const fetchGroups = useCallback(async (): Promise<void> => {
    try {
      const response = await getGroups({}, { signal: canceler.signal });

      setGroups((prev) => {
        if (isEqual(prev, response.groups)) return prev;
        return response.groups || [];
      });
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch groups.' });
    }
  }, [canceler.signal]);

  useEffect(() => {
    fetchUsers();
  }, [fetchUsers]);

  useEffect(() => {
    fetchGroups();
  }, [fetchGroups]);

  useEffect(() => {
    if (rbacEnabled) {
      fetchKnownRoles();
    }
  }, [fetchKnownRoles, rbacEnabled]);

  const { modalOpen: openCreateUserModal, contextHolder: modalCreateUserContextHolder } =
    useModalCreateUser({ groups, onClose: fetchUsers });

  const onClickCreateUser = useCallback(() => {
    openCreateUserModal();
  }, [openCreateUserModal]);

  const columns = useMemo(() => {
    const actionRenderer = (_: string, record: DetailedUser) => {
      return <UserActionDropdown fetchUsers={fetchUsers} groups={groups} user={record} />;
    };
    const columns = [
      {
        dataIndex: 'displayName',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['displayName'],
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
  }, [fetchUsers, groups, rbacEnabled]);

  const table = useMemo(() => {
    return settings ? (
      <InteractiveTable
        columns={columns}
        containerRef={pageRef}
        dataSource={users}
        interactiveColumns={false}
        loading={loadableUser === NotLoaded}
        pagination={getFullPaginationConfig(
          {
            limit: settings.tableLimit,
            offset: settings.tableOffset,
          },
          total,
        )}
        rowClassName={defaultRowClassName({ clickable: false })}
        rowKey="id"
        settings={settings as InteractiveTableSettings}
        showSorterTooltip={false}
        size="small"
        updateSettings={updateSettings as UpdateSettings}
      />
    ) : (
      <SkeletonTable columns={columns.length} />
    );
  }, [users, loadableUser, settings, columns, total, updateSettings]);
  return (
    <Page
      containerRef={pageRef}
      options={
        <Space>
          <Button
            aria-label={CREAT_USER_LABEL}
            disabled={!canModifyUsers}
            onClick={onClickCreateUser}>
            {CREATE_USER}
          </Button>
        </Space>
      }
      title={USER_TITLE}>
      <div className={css.usersTable}>{table}</div>
      {modalCreateUserContextHolder}
    </Page>
  );
};

export default UserManagement;
