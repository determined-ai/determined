import { Space, Table } from 'antd';
import dayjs from 'dayjs';
import Button from 'determined-ui/Button';
import Dropdown, { MenuItem } from 'determined-ui/Dropdown';
import Icon from 'determined-ui/Icon';
import { useModal } from 'determined-ui/Modal';
import Nameplate from 'determined-ui/Nameplate';
import Section from 'determined-ui/Section';
import _ from 'lodash';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import dropdownCss from 'components/ActionDropdown/ActionDropdown.module.scss';
import CreateGroupModalComponent from 'components/CreateGroupModal';
import DeleteGroupModalComponent from 'components/DeleteGroupModal';
import InteractiveTable, { onRightClickableCell } from 'components/Table/InteractiveTable';
import SkeletonTable from 'components/Table/SkeletonTable';
import { defaultRowClassName, getFullPaginationConfig } from 'components/Table/Table';
import UserBadge from 'components/UserBadge';
import usePermissions from 'hooks/usePermissions';
import { useSettings } from 'hooks/useSettings';
import { getGroup, getGroups } from 'services/api';
import { V1GroupDetails, V1GroupSearchResult, V1User } from 'services/api-ts-sdk';
import determinedStore from 'stores/determinedInfo';
import roleStore from 'stores/roles';
import { User } from 'types';
import handleError from 'utils/error';
import { useObservable } from 'utils/observable';
import { alphaNumericSorter } from 'utils/sort';

import css from './GroupManagement.module.scss';
import settingsConfig, { DEFAULT_COLUMN_WIDTHS, DEFAULT_COLUMNS } from './GroupManagement.settings';
import RemoveUserFromGroupModalComponent from './RemoveUserFromGroupModal';

interface DropdownProps {
  expanded: boolean;
  fetchGroup: (groupId: number) => void;
  fetchGroups: () => void;
  group: V1GroupSearchResult;
}

const MenuKey = {
  Delete: 'delete',
  Edit: 'edit',
} as const;

const DROPDOWN_MENU: MenuItem[] = [
  { key: MenuKey.Edit, label: 'Edit Group' },
  { danger: true, key: MenuKey.Delete, label: 'Delete Group' },
];

const GroupActionDropdown = ({ expanded, fetchGroups, fetchGroup, group }: DropdownProps) => {
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
      <EditGroupModal.Component group={group} onClose={onFinishEdit} />
      <DeleteGroupModal.Component group={group} onClose={fetchGroups} />
    </div>
  );
};

const GroupManagement: React.FC = () => {
  const { rbacEnabled } = useObservable(determinedStore.info);
  const [groups, setGroups] = useState<V1GroupSearchResult[]>([]);
  const [groupUsers, setGroupUsers] = useState<V1GroupDetails[]>([]);
  const [groupResult, setGroupResult] = useState<V1GroupSearchResult | undefined>(undefined);
  const [user, setUser] = useState<V1User | undefined>(undefined);
  const [isLoading, setIsLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [expandedKeys, setExpandedKeys] = useState<readonly React.Key[]>([]);
  const pageRef = useRef<HTMLElement>(null);
  const canceler = useRef(new AbortController());

  const { settings, updateSettings } = useSettings(settingsConfig);

  const { canModifyGroups, canViewGroups } = usePermissions();
  const RemoveUserFromGroupModal = useModal(RemoveUserFromGroupModalComponent);

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
        if (_.isEqual(prev, response.groups)) return prev;
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
      setGroupUsers(structuredClone(groupUsers));
    },
    [groupUsers],
  );

  useEffect(() => {
    const currentCanceler = canceler.current;
    return () => currentCanceler.abort();
  }, []);

  useEffect(() => {
    fetchGroups();
  }, [fetchGroups]);

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

  const expandedUserRender = useCallback(
    (record: V1GroupSearchResult) => {
      const {
        group: { groupId },
      } = record;
      const g = groupUsers.find((gr) => gr.groupId === groupId);
      const userColumn = [
        {
          dataIndex: 'username',
          key: 'username',
          render: (_: string, r: V1User) => {
            const user: User = {
              displayName: r.displayName,
              id: r.id ?? 0,
              lastAuthAt: r.lastAuthAt ? dayjs(r.lastAuthAt).unix() : undefined,
              modifiedAt: r.modifiedAt ? dayjs(r.modifiedAt).unix() : undefined,
              username: r.username,
            };
            return <UserBadge user={user} />;
          },
          title: 'User Name',
          width: '90%',
        },
        {
          key: 'action',
          render: (_: string, r: V1User) => {
            if (canModifyGroups) {
              return (
                <Button
                  onClick={() => {
                    setGroupResult(record);
                    setUser(r);
                    RemoveUserFromGroupModal.open();
                  }}>
                  Remove
                </Button>
              );
            }
            return null;
          },
          title: '',
        },
      ];

      return (
        <Table
          columns={userColumn}
          dataSource={g?.users?.sort((a, b) =>
            alphaNumericSorter(a.displayName || a.username, b.displayName || b.username),
          )}
          loading={!g}
          pagination={false}
          rowKey="id"
        />
      );
    },
    [groupUsers, canModifyGroups, RemoveUserFromGroupModal],
  );

  const columns = useMemo(() => {
    const actionRenderer = (_: string, record: V1GroupSearchResult) => {
      return canModifyGroups ? (
        <GroupActionDropdown
          expanded={!!(record.group.groupId && expandedKeys.includes(record.group.groupId))}
          fetchGroup={fetchGroup}
          fetchGroups={fetchGroups}
          group={record}
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
        title: 'Members',
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
  }, [fetchGroups, expandedKeys, fetchGroup, canModifyGroups]);

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
      <Section title="Groups">
        <Space>
          <Button disabled={!canModifyGroups} onClick={CreateGroupModal.open}>
            New Group
          </Button>
        </Space>
        {canViewGroups && <div className={css.usersTable}>{table}</div>}
      </Section>
      <CreateGroupModal.Component onClose={fetchGroups} />
      {groupResult && (
        <RemoveUserFromGroupModal.Component
          fetchGroups={fetchGroups}
          groupResult={groupResult}
          user={user}
          onExpand={onExpand}
        />
      )}
    </>
  );
};

export default GroupManagement;
