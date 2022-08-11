import { Button, Dropdown, Menu, Space } from 'antd';
import type { MenuProps } from 'antd';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import InteractiveTable, { InteractiveTableSettings,
  onRightClickableCell } from 'components/InteractiveTable';
import Page from 'components/Page';
import { checkmarkRenderer, defaultRowClassName,
  getFullPaginationConfig, relativeTimeRenderer } from 'components/Table';
import useModalCreateUser from 'hooks/useModal/UserSettings/useModalCreateUser';
import useSettings, { UpdateSettings } from 'hooks/useSettings';
import { getUsers } from 'services/api';
import { V1GetUsersRequestSortBy } from 'services/api-ts-sdk';
import dropdownCss from 'shared/components/ActionDropdown/ActionDropdown.module.scss';
import Icon from 'shared/components/Icon/Icon';
import { isEqual } from 'shared/utils/data';
import { validateDetApiEnum } from 'shared/utils/service';
import { DetailedUser } from 'types';
import handleError from 'utils/error';

import css from './UserManagement.module.scss';
import settingsConfig, { DEFAULT_COLUMN_WIDTHS,
  UserManagementSettings } from './UserManagement.settings';

const UserManagement: React.FC = () => {
  const [ users, setUsers ] = useState<DetailedUser[]>([]);
  const [ isLoading, setIsLoading ] = useState(true);
  const [ total, setTotal ] = useState(0);
  const [ canceler ] = useState(new AbortController());
  const pageRef = useRef<HTMLElement>(null);

  const {
    settings,
    updateSettings,
  } = useSettings<UserManagementSettings>(settingsConfig);

  const fetchUsers = useCallback(async (): Promise<void> => {
    try {
      const response = await getUsers(
        {
          limit: settings.tableLimit,
          offset: settings.tableOffset,
          orderBy: settings.sortDesc ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC',
          sortBy: validateDetApiEnum(V1GetUsersRequestSortBy, settings.sortKey),
        },
        { signal: canceler.signal },
      );

      setTotal(response.pagination.total ?? 0);
      setUsers((prev) => {
        if (isEqual(prev, response.users)) return prev;
        return response.users;
      });
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch users.' });
    } finally {

      setIsLoading(false);
    }
  }, [ canceler.signal,
    settings.sortDesc,
    settings.sortKey,
    settings.tableLimit,
    settings.tableOffset,
  ]);

  useEffect(() => {
    fetchUsers();
  }, [ settings.sortDesc,
    settings.sortKey,
    settings.tableLimit,
    settings.tableOffset,
    fetchUsers ]);

  const {
    modalOpen: openCreateUserModal,
    contextHolder: modalCreateUserContextHolder,
  } = useModalCreateUser({ onClose: fetchUsers });

  const onClickCreateUser = useCallback(() => {
    openCreateUserModal();
  }, [ openCreateUserModal ]);

  const columns = useMemo(() => {
    const actionRenderer = (_:string, record: DetailedUser) => {
      const menuItems: MenuProps['items'] = [
        { key: 'edit', label: 'Edit' },
        { key: 'state', label: `${record.isActive ? 'Deactive' : 'Active'}` },
      ];

      return (
        <div className={dropdownCss.base}>
          <Dropdown
            overlay={<Menu items={menuItems} />}
            placement="bottomRight"
            trigger={[ 'click' ]}>
            <Button className={css.overflow} type="text">
              <Icon name="overflow-vertical" />
            </Button>
          </Dropdown>
        </div>

      );
    };
    return [
      {
        dataIndex: 'displayName',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['displayName'],
        key: V1GetUsersRequestSortBy.DISPLAYNAME,
        onCell: onRightClickableCell,
        sorter: true,
        title: 'Display Name',
      },
      {
        dataIndex: 'username',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['username'],
        key: V1GetUsersRequestSortBy.USERNAME,
        onCell: onRightClickableCell,
        sorter: true,
        title: 'User Name',
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
        render: (value: number): React.ReactNode =>
          relativeTimeRenderer(new Date(value)),
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
  }, []);

  const table = useMemo(() => {
    return (
      <InteractiveTable
        columns={columns}
        containerRef={pageRef}
        dataSource={users}
        loading={isLoading}
        pagination={getFullPaginationConfig({
          limit: settings.tableLimit,
          offset: settings.tableOffset,
        }, total)}
        rowClassName={defaultRowClassName({ clickable: false })}
        rowKey="id"
        settings={settings as InteractiveTableSettings}
        showSorterTooltip={false}
        size="small"
        updateSettings={updateSettings as UpdateSettings<InteractiveTableSettings>}
      />
    );
  }, [ users, isLoading, settings, columns, total, updateSettings ]);
  return (
    <Page
      containerRef={pageRef}
      options={(
        <Space>
          <Button onClick={onClickCreateUser}>New User</Button>
        </Space>
      )}
      title="Users">
      <div className={css.usersTable}>{table}</div>
      {modalCreateUserContextHolder}
    </Page>
  );
};

export default UserManagement;
