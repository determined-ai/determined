import { Button, Dropdown, Menu, Space } from 'antd';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import InteractiveTable, { InteractiveTableSettings,
  onRightClickableCell } from 'components/InteractiveTable';
import Page from 'components/Page';
import { checkmarkRenderer, defaultRowClassName,
  getFullPaginationConfig, relativeTimeRenderer } from 'components/Table';
import useModalCreateGroup from 'hooks/useModal/UserSettings/useModalCreateGroup';
import useSettings, { UpdateSettings } from 'hooks/useSettings';
import { getGroups, getUsers } from 'services/api';
import { V1GroupSearchResult } from 'services/api-ts-sdk';
import dropdownCss from 'shared/components/ActionDropdown/ActionDropdown.module.scss';
import Icon from 'shared/components/Icon/Icon';
import { isEqual } from 'shared/utils/data';
import { validateDetApiEnum } from 'shared/utils/service';
import { DetailedUser } from 'types';
import handleError from 'utils/error';

import css from './GroupManagement.module.scss';
import settingsConfig, { DEFAULT_COLUMN_WIDTHS,
  GroupManagementSettings } from './GroupManagement.settings';

interface DropdownProps {
  fetchGroups: () => void;
  group: V1GroupSearchResult;
  users: DetailedUser[]
}

const GroupActionDropdown = ({ fetchGroups, group, users }: DropdownProps) => {
  const {
    modalOpen: openEditGroupModal,
    contextHolder: modalEditGroupContextHolder,
  } = useModalCreateGroup({ group: group, onClose: fetchGroups, users: users });
  const onClickEditGroup = () => {
    openEditGroupModal();
  };
    // const onToggleDelete = async () => {
    //   await patchUser({ userId: user.id, userParams: { active: !user.isActive } });
    //   message.success(`User has been ${user.isActive ? 'deactivated' : 'activated'}`);
    //   fetchUsers();
    // };
  const menuItems = (
    <Menu>
      <Menu.Item key="edit" onClick={onClickEditGroup}>
        Edit
      </Menu.Item>
      <Menu.Item danger key="delete">
        Delete
      </Menu.Item>
    </Menu>
  );

  return (
    <div className={dropdownCss.base}>
      <Dropdown
        overlay={menuItems}
        placement="bottomRight"
        trigger={[ 'click' ]}>
        <Button className={css.overflow} type="text">
          <Icon name="overflow-vertical" />
        </Button>
      </Dropdown>
      {modalEditGroupContextHolder}
    </div>
  );
};

const GroupManagement: React.FC = () => {
  const [ groups, setGroups ] = useState<V1GroupSearchResult[]>([]);
  const [ users, setUsers ] = useState<DetailedUser[]>([]);
  const [ isLoading, setIsLoading ] = useState(true);
  const [ total, setTotal ] = useState(0);
  const [ canceler ] = useState(new AbortController());
  const pageRef = useRef<HTMLElement>(null);

  const {
    settings,
    updateSettings,
  } = useSettings<GroupManagementSettings>(settingsConfig);

  const fetchGroups = useCallback(async (): Promise<void> => {
    try {
      const response = await getGroups(
        {
          limit: settings.tableLimit,
          offset: settings.tableOffset,
        },
        { signal: canceler.signal },
      );

      setTotal(response.pagination?.total ?? 0);
      setGroups((prev) => {
        if (isEqual(prev, response.groups)) return prev;
        return response.groups || [];
      });
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch groups.' });
    } finally {

      setIsLoading(false);
    }
  }, [ canceler.signal,
    settings.tableLimit,
    settings.tableOffset,
  ]);

  const fetchUsers = useCallback(async (): Promise<void> => {
    try {
      const response = await getUsers(
        {},
        { signal: canceler.signal },
      );
      setUsers((prev) => {
        if (isEqual(prev, response.users)) return prev;
        return response.users;
      });
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch users.' });
    }
  }, [ canceler.signal ]);

  useEffect(() => {
    fetchGroups();
    fetchUsers();
  }, [
    settings.tableLimit,
    settings.tableOffset,
    fetchGroups ]);

  const {
    modalOpen: openCreateGroupModal,
    contextHolder: modalCreateGroupContextHolder,
  } = useModalCreateGroup({ onClose: fetchGroups, users: users });

  const onClickCreateGroup = useCallback(() => {
    openCreateGroupModal();
  }, [ openCreateGroupModal ]);

  const columns = useMemo(() => {
    const actionRenderer = (_:string, record: V1GroupSearchResult) => {
      return <GroupActionDropdown fetchGroups={fetchGroups} group={record} users={users} />;
    };

    return [
      {
        dataIndex: 'id',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['id'],
        key: 'id',
        onCell: onRightClickableCell,
        render: (_:string, r: V1GroupSearchResult) => r.group.groupId,
        title: 'Group ID',
      },
      {
        dataIndex: 'name',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['name'],
        key: 'name',
        onCell: onRightClickableCell,
        render: (_:string, r: V1GroupSearchResult) => r.group.name,
        title: 'Group Name',
      },
      {
        dataIndex: 'users',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['users'],
        key: 'users',
        onCell: onRightClickableCell,
        render: (_:string, r: V1GroupSearchResult) => r.numMembers,
        title: 'Users',
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
  }, [ users ]);

  const table = useMemo(() => {
    return (
      <InteractiveTable
        columns={columns}
        containerRef={pageRef}
        dataSource={groups}
        loading={isLoading}
        pagination={getFullPaginationConfig({
          limit: settings.tableLimit,
          offset: settings.tableOffset,
        }, total)}
        rowClassName={defaultRowClassName({ clickable: false })}
        rowKey={(r) => r.group.groupId||0}
        settings={settings as InteractiveTableSettings}
        showSorterTooltip={false}
        size="small"
        updateSettings={updateSettings as UpdateSettings<InteractiveTableSettings>}
      />
    );
  }, [ groups, isLoading, settings, columns, total, updateSettings ]);
  return (
    <Page
      containerRef={pageRef}
      options={(
        <Space>
          <Button onClick={onClickCreateGroup}>New Group</Button>
        </Space>
      )}
      title="Groups">
      <div className={css.usersTable}>{table}</div>
      {modalCreateGroupContextHolder}
    </Page>
  );
};

export default GroupManagement;
