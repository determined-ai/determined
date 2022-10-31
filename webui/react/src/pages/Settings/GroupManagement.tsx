import { Button, Dropdown, Menu, message, Space, Table } from 'antd';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import Page from 'components/Page';
import InteractiveTable, {
  InteractiveTableSettings,
  onRightClickableCell,
} from 'components/Table/InteractiveTable';
import { defaultRowClassName, getFullPaginationConfig } from 'components/Table/Table';
import useFeature from 'hooks/useFeature';
import { useFetchKnownRoles } from 'hooks/useFetch';
import useModalCreateGroup from 'hooks/useModal/UserSettings/useModalCreateGroup';
import useModalDeleteGroup from 'hooks/useModal/UserSettings/useModalDeleteGroup';
import usePermissions from 'hooks/usePermissions';
import useSettings, { UpdateSettings } from 'hooks/useSettings';
import { getGroup, getGroups, getUsers, updateGroup } from 'services/api';
import { V1GroupDetails, V1GroupSearchResult, V1User } from 'services/api-ts-sdk';
import dropdownCss from 'shared/components/ActionDropdown/ActionDropdown.module.scss';
import Icon from 'shared/components/Icon/Icon';
import { clone, isEqual } from 'shared/utils/data';
import { ErrorType } from 'shared/utils/error';
import { DetailedUser } from 'types';
import handleError from 'utils/error';

import css from './GroupManagement.module.scss';
import settingsConfig, {
  DEFAULT_COLUMN_WIDTHS,
  GroupManagementSettings,
} from './GroupManagement.settings';

interface DropdownProps {
  expanded: boolean;
  fetchGroup: (groupId: number) => void;
  fetchGroups: () => void;
  group: V1GroupSearchResult;
  users: DetailedUser[];
}

const GroupActionDropdown = ({
  expanded,
  fetchGroups,
  fetchGroup,
  group,
  users,
}: DropdownProps) => {
  const onFinishEdit = () => {
    fetchGroups();
    expanded && group.group.groupId && fetchGroup(group.group.groupId);
  };
  const { modalOpen: openEditGroupModal, contextHolder: modalEditGroupContextHolder } =
    useModalCreateGroup({ group, onClose: onFinishEdit, users });
  const { modalOpen: openDeleteGroupModal, contextHolder: modalDeleteGroupContextHolder } =
    useModalDeleteGroup({ group, onClose: fetchGroups });

  const menuItems = (
    <Menu>
      <Menu.Item key="edit" onClick={() => openEditGroupModal()}>
        Edit
      </Menu.Item>
      <Menu.Item danger key="delete" onClick={() => openDeleteGroupModal()}>
        Delete
      </Menu.Item>
    </Menu>
  );

  return (
    <div className={dropdownCss.base}>
      <Dropdown overlay={menuItems} placement="bottomRight" trigger={['click']}>
        <Button className={css.overflow} type="text">
          <Icon name="overflow-vertical" />
        </Button>
      </Dropdown>
      {modalEditGroupContextHolder}
      {modalDeleteGroupContextHolder}
    </div>
  );
};

const GroupManagement: React.FC = () => {
  const [groups, setGroups] = useState<V1GroupSearchResult[]>([]);
  const [groupUsers, setGroupUsers] = useState<V1GroupDetails[]>([]);
  const [users, setUsers] = useState<DetailedUser[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [expandedKeys, setExpandedKeys] = useState<readonly React.Key[]>([]);
  const [canceler] = useState(new AbortController());
  const pageRef = useRef<HTMLElement>(null);

  const { settings, updateSettings } = useSettings<GroupManagementSettings>(settingsConfig);

  const { canModifyGroups, canViewGroups } = usePermissions();

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
  }, [canceler.signal, settings.tableLimit, settings.tableOffset]);

  const fetchKnownRoles = useFetchKnownRoles(canceler);

  const fetchGroup = useCallback(
    async (groupId: number): Promise<void> => {
      const response = await getGroup({ groupId });
      const i = groupUsers.findIndex((gr) => gr.groupId === groupId);
      i >= 0 ? (groupUsers[i] = response.group) : groupUsers.push(response.group);
      setGroupUsers(clone(groupUsers));
    },
    [groupUsers],
  );

  const fetchUsers = useCallback(async (): Promise<void> => {
    try {
      const response = await getUsers({}, { signal: canceler.signal });
      setUsers((prev) => {
        if (isEqual(prev, response.users)) return prev;
        return response.users;
      });
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch users.' });
    }
  }, [canceler.signal]);

  useEffect(() => {
    fetchGroups();
    fetchUsers();
  }, [settings.tableLimit, settings.tableOffset, fetchGroups, fetchUsers]);

  const rbacEnabled = useFeature().isOn('rbac');
  useEffect(() => {
    if (rbacEnabled) {
      fetchKnownRoles();
    }
  }, [fetchKnownRoles, rbacEnabled]);

  const { modalOpen: openCreateGroupModal, contextHolder: modalCreateGroupContextHolder } =
    useModalCreateGroup({ onClose: fetchGroups, users: users });

  const onClickCreateGroup = useCallback(() => {
    openCreateGroupModal();
  }, [openCreateGroupModal]);

  const onExpand = useCallback(
    (expand: boolean, record: V1GroupSearchResult) => {
      const {
        group: { groupId },
      } = record;
      if (!groupId || !expand) return;
      fetchGroup(groupId);
    },
    [fetchGroup],
  );

  const onExpandedRowsChange = (keys: readonly React.Key[]) => setExpandedKeys(keys);

  const onRemoveUser = useCallback(
    async (record: V1GroupSearchResult, userId) => {
      const {
        group: { groupId },
      } = record;
      if (!groupId) return;
      try {
        await updateGroup({ groupId, removeUsers: [userId] });
        message.success('User has been deleted.');
        onExpand(true, record);
        fetchGroups();
      } catch (e) {
        message.error('Error deleting group.');
        handleError(e, { silent: true, type: ErrorType.Input });
      }
    },
    [onExpand, fetchGroups],
  );

  const expandedUserRender = useCallback(
    (record: V1GroupSearchResult) => {
      const {
        group: { groupId },
      } = record;
      const g = groupUsers.find((gr) => gr.groupId === groupId);
      const userColumn = [
        {
          dataIndex: 'displayName',
          key: 'displayName',
          title: 'Display Name',
          width: '40%',
        },
        {
          dataIndex: 'username',
          key: 'username',
          title: 'User Name',
          width: '50%',
        },
        {
          key: 'action',
          render: (_: string, r: V1User) =>
            canModifyGroups ? (
              <Button onClick={() => onRemoveUser(record, r.id)}>Remove</Button>
            ) : null,
          title: '',
        },
      ];

      return (
        <Table
          columns={userColumn}
          dataSource={g?.users}
          loading={!g}
          pagination={false}
          rowKey="id"
        />
      );
    },
    [onRemoveUser, groupUsers, canModifyGroups],
  );

  const columns = useMemo(() => {
    const actionRenderer = (_: string, record: V1GroupSearchResult) => {
      return canModifyGroups ? (
        <GroupActionDropdown
          expanded={!!(record.group.groupId && expandedKeys.includes(record.group.groupId))}
          fetchGroup={fetchGroup}
          fetchGroups={fetchGroups}
          group={record}
          users={users}
        />
      ) : null;
    };

    return [
      {
        dataIndex: 'name',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['name'],
        key: 'name',
        onCell: onRightClickableCell,
        render: (_: string, r: V1GroupSearchResult) => r.group.name,
        title: 'Group Name',
      },
      {
        dataIndex: 'users',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['users'],
        key: 'users',
        onCell: onRightClickableCell,
        render: (_: string, r: V1GroupSearchResult) => r.numMembers,
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
  }, [users, fetchGroups, expandedKeys, fetchGroup, canModifyGroups]);

  const table = useMemo(() => {
    return (
      <InteractiveTable
        columns={columns}
        containerRef={pageRef}
        dataSource={groups}
        expandable={{ expandedRowRender: expandedUserRender, onExpand, onExpandedRowsChange }}
        loading={isLoading}
        pagination={getFullPaginationConfig(
          {
            limit: settings.tableLimit,
            offset: settings.tableOffset,
          },
          total,
        )}
        rowClassName={defaultRowClassName({ clickable: false })}
        rowKey={(r) => r.group.groupId || 0}
        settings={settings as InteractiveTableSettings}
        showSorterTooltip={false}
        size="small"
        updateSettings={updateSettings as UpdateSettings<InteractiveTableSettings>}
      />
    );
  }, [groups, isLoading, settings, columns, total, updateSettings, expandedUserRender, onExpand]);

  return (
    <Page
      containerRef={pageRef}
      options={
        <Space>
          <Button disabled={!canModifyGroups} onClick={onClickCreateGroup}>
            New Group
          </Button>
        </Space>
      }
      title="Groups">
      {canViewGroups && <div className={css.usersTable}>{table}</div>}
      {modalCreateGroupContextHolder}
    </Page>
  );
};

export default GroupManagement;
