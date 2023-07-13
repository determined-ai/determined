import { Space, Table } from 'antd';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import dropdownCss from 'components/ActionDropdown/ActionDropdown.module.scss';
import CreateGroupModalComponent from 'components/CreateGroupModal';
import DeleteGroupModalComponent from 'components/DeleteGroupModal';
import Button from 'components/kit/Button';
import Dropdown from 'components/kit/Dropdown';
import Icon from 'components/kit/Icon';
import { useModal } from 'components/kit/Modal';
import Nameplate from 'components/kit/Nameplate';
import Section from 'components/Section';
import InteractiveTable, { onRightClickableCell } from 'components/Table/InteractiveTable';
import SkeletonTable from 'components/Table/SkeletonTable';
import { defaultRowClassName, getFullPaginationConfig } from 'components/Table/Table';
import usePermissions from 'hooks/usePermissions';
import { useSettings } from 'hooks/useSettings';
import { getGroup, getGroups, getUsers, updateGroup } from 'services/api';
import { V1GroupDetails, V1GroupSearchResult, V1User } from 'services/api-ts-sdk';
import determinedStore from 'stores/determinedInfo';
import roleStore from 'stores/roles';
import { DetailedUser } from 'types';
import { clone, isEqual } from 'utils/data';
import { message } from 'utils/dialogApi';
import { ErrorType } from 'utils/error';
import handleError from 'utils/error';
import { useObservable } from 'utils/observable';

import css from './GroupManagement.module.scss';
import settingsConfig, { DEFAULT_COLUMN_WIDTHS, DEFAULT_COLUMNS } from './GroupManagement.settings';

interface DropdownProps {
  expanded: boolean;
  fetchGroup: (groupId: number) => void;
  fetchGroups: () => void;
  group: V1GroupSearchResult;
  users: DetailedUser[];
}

const MenuKey = {
  Delete: 'delete',
  Edit: 'edit',
} as const;

const DROPDOWN_MENU = [
  { key: MenuKey.Edit, label: 'Edit' },
  { key: MenuKey.Delete, label: 'Delete' },
];

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
  const EditGroupModal = useModal(CreateGroupModalComponent);
  const DeleteGroupModal = useModal(DeleteGroupModalComponent);

  const handleDropdown = useCallback(
    (key: string) => {
      switch (key) {
        case MenuKey.Delete:
          DeleteGroupModal.open();
          break;
        case MenuKey.Edit:
          EditGroupModal.open();
          break;
      }
    },
    [DeleteGroupModal, EditGroupModal],
  );

  return (
    <div className={dropdownCss.base}>
      <Dropdown menu={DROPDOWN_MENU} placement="bottomRight" onClick={handleDropdown}>
        <Button icon={<Icon name="overflow-vertical" size="small" title="Action menu" />} />
      </Dropdown>
      <EditGroupModal.Component group={group} users={users} onClose={onFinishEdit} />
      <DeleteGroupModal.Component group={group} onClose={fetchGroups} />
    </div>
  );
};

const GroupManagement: React.FC = () => {
  const { rbacEnabled } = useObservable(determinedStore.info);
  const [groups, setGroups] = useState<V1GroupSearchResult[]>([]);
  const [groupUsers, setGroupUsers] = useState<V1GroupDetails[]>([]);
  const [users, setUsers] = useState<DetailedUser[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [expandedKeys, setExpandedKeys] = useState<readonly React.Key[]>([]);
  const pageRef = useRef<HTMLElement>(null);
  const canceler = useRef(new AbortController());

  const { settings, updateSettings } = useSettings(settingsConfig);

  const { canModifyGroups, canViewGroups } = usePermissions();

  const fetchGroups = useCallback(async (): Promise<void> => {
    if (!('tableLimit' in settings) || !('tableOffset' in settings)) return;
    try {
      const response = await getGroups(
        {
          limit: settings.tableLimit,
          offset: settings.tableOffset,
        },
        { signal: canceler.current.signal },
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
  }, [settings]);

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
      const response = await getUsers({}, { signal: canceler.current.signal });
      setUsers((prev) => {
        if (isEqual(prev, response.users)) return prev;
        return response.users;
      });
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch users.' });
    }
  }, []);

  useEffect(() => {
    const currentCanceler = canceler.current;
    return () => currentCanceler.abort();
  }, []);

  useEffect(() => {
    fetchGroups();
    fetchUsers();
  }, [fetchGroups, fetchUsers]);

  useEffect(() => (rbacEnabled ? roleStore.fetch() : undefined), [rbacEnabled]);

  const CreateGroupModal = useModal(CreateGroupModalComponent);

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
    async (record: V1GroupSearchResult, userId?: number) => {
      const {
        group: { groupId },
      } = record;
      if (!groupId || !userId) return;
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
        render: (_: string, r: V1GroupSearchResult) => (
          <Nameplate icon={<Icon name="group" title="Group" />} name={r.group.name ?? ''} />
        ),
        title: 'Group',
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
    return settings ? (
      <InteractiveTable<V1GroupSearchResult>
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
        settings={{
          ...settings,
          columns: DEFAULT_COLUMNS,
        }}
        showSorterTooltip={false}
        size="small"
        updateSettings={updateSettings}
      />
    ) : (
      <SkeletonTable columns={columns.length} />
    );
  }, [groups, isLoading, settings, columns, total, updateSettings, expandedUserRender, onExpand]);

  return (
    <>
      <Section
        options={
          <Space>
            <Button disabled={!canModifyGroups} onClick={CreateGroupModal.open}>
              New Group
            </Button>
          </Space>
        }
        title="Groups">
        {canViewGroups && <div className={css.usersTable}>{table}</div>}
      </Section>
      <CreateGroupModal.Component users={users} onClose={fetchGroups} />
    </>
  );
};

export default GroupManagement;
