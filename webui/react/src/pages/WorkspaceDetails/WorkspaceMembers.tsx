import { Button, Dropdown, Menu, Select } from 'antd';
import { FilterDropdownProps } from 'antd/lib/table/interface';
import React, { useCallback, useMemo } from 'react';

import InteractiveTable, { ColumnDef,
  InteractiveTableSettings } from 'components/InteractiveTable';
import { getFullPaginationConfig } from 'components/Table';
import TableFilterSearch from 'components/TableFilterSearch';
import Avatar from 'components/UserAvatar';
import useSettings, { UpdateSettings } from 'hooks/useSettings';
import groupIcon from 'shared/assets/images/People.svg';
import { Size } from 'shared/components/Avatar';
import Icon from 'shared/components/Icon/Icon';
import {
  alphaNumericSorter,
} from 'shared/utils/sort';
import { DetailedUser } from 'types';
import { getDisplayName } from 'utils/user';

import settingsConfig, { DEFAULT_COLUMN_WIDTHS,
  WorkspaceMembersSettings } from './WorkspaceMembers.settings';
import css from './WorkspaceMembers.module.scss';

export interface Member extends DetailedUser {
  role?: string;
}

const isMember = (obj: MemberOrGroup) => {
  const member = obj as Member;
  return member.username || member.displayName;
};

const getName = (obj: MemberOrGroup): string => {
  const member = obj as Member;
  const group = obj as Group;
  return isMember(obj) ? getDisplayName(member) : group.name;
};
export interface Group {
  id: number;
  name: string;
  role: string;
}

type MemberOrGroup = Member | Group;

interface Props {
  pageRef: React.RefObject<HTMLElement>;
  users: DetailedUser[];
}

const roles = [ 'Admin', 'Editor', 'Viewer' ];

const GroupOrMemberActionDropdown = () => {
  const menuItems = (
    <Menu>
      <Menu.Item danger key="delete">
        Delete
      </Menu.Item>
    </Menu>
  );

  return (
    <div>
      <Dropdown
        overlay={menuItems}
        placement="bottomRight"
        trigger={[ 'click' ]}>
        <Button className={css.overflow} type="text">
          <Icon name="overflow-vertical" />
        </Button>
      </Dropdown>
    </div>
  );
};

const WorkspaceMembers: React.FC<Props> = ({ users, pageRef }: Props) => {

  const members: Member[] = [];
  users.forEach((u) => {
    const m: Member = u;
    m.role = 'Editor';
    members.push(m);
  });

  const groups: MemberOrGroup[] = [
    { id: Math.random() * 1000, name: 'Group One', role: 'Admin' },
    { id: Math.random() * 1000, name: 'Group Two', role: 'Admin' },
    { id: Math.random() * 1000, name: 'Group Three', role: 'Admin' },
  ];

  const membersAndGroups = groups.concat(members);

  const {
    settings,
    updateSettings,
  } = useSettings<WorkspaceMembersSettings>(settingsConfig);

  const handleNameSearchApply = useCallback((newSearch: string) => {
    updateSettings({ name: newSearch || undefined });
  }, [ updateSettings ]);

  const handleNameSearchReset = useCallback(() => {
    updateSettings({ name: undefined });
  }, [ updateSettings ]);

  const nameFilterSearch = useCallback((filterProps: FilterDropdownProps) => (
    <TableFilterSearch
      {...filterProps}
      value={settings.name || ''}
      onReset={handleNameSearchReset}
      onSearch={handleNameSearchApply}
    />
  ), [ handleNameSearchApply, handleNameSearchReset, settings.name ]);

  const tableSearchIcon = useCallback(() => <Icon name="search" size="tiny" />, []);

  const columns = useMemo(() => {

    const nameRenderer = (value: string, record: MemberOrGroup) => {
      if (isMember(record)){
        const member = record as Member;
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
      const group = record as Group;
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

    const roleRenderer = (value: string, record: Member) => {
      return (
        <Select
          value={record.role}>{
            roles.map((role) => (
              <Select.Option key={role} value={role}>
                {role}
              </Select.Option>
            ))
          }
        </Select>
      );
    };

    const actionRenderer = () => {
      return (
        <GroupOrMemberActionDropdown />
      );
    };

    return [
      {
        dataIndex: 'name',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['name'],
        filterDropdown: nameFilterSearch,
        filterIcon: tableSearchIcon,
        render: nameRenderer,
        sorter: (a: MemberOrGroup, b: MemberOrGroup) => alphaNumericSorter(getName(a), getName(b)),
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
    ] as ColumnDef<MemberOrGroup>[];
  }, [ nameFilterSearch, tableSearchIcon ]);

  return (
    <div className={css.membersContainer}>
      <InteractiveTable
        columns={columns}
        containerRef={pageRef}
        dataSource={membersAndGroups}
        pagination={getFullPaginationConfig({
          limit: settings.tableLimit,
          offset: settings.tableOffset,
        }, membersAndGroups.length)}
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
