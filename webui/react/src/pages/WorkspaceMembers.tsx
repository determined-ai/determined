import { Select, Button } from 'antd';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { FilterDropdownProps, SorterResult } from 'antd/lib/table/interface';
import InteractiveTable, { ColumnDef,
  InteractiveTableSettings} from 'components/InteractiveTable';
import useSettings, { UpdateSettings } from 'hooks/useSettings';
import { DetailedUser, Workspace } from 'types';
import TableFilterSearch from 'components/TableFilterSearch';
import { getFullPaginationConfig} from 'components/Table';
import {
  alphaNumericSorter
} from 'shared/utils/sort';
import css from './WorkspaceDetails.module.scss';
import settingsConfig, { WorkspaceMembersSettings, DEFAULT_COLUMN_WIDTHS } from './WorkspaceMembers.settings';
import Avatar from 'components/UserAvatar';
import groupIcon from 'shared/assets/images/People.svg';
import { Size } from 'shared/components/Avatar';
import Icon from 'shared/components/Icon/Icon';
import { getDisplayName } from 'utils/user';

export interface Member extends DetailedUser {
  role?: string;
}

const hasDisplayName = (object: MemberOrGroup) => object.hasOwnProperty('displayName');
const hasUserName = (object: MemberOrGroup) => object.hasOwnProperty('username');

const isUser = (u: MemberOrGroup) => {
  const b = u as Member;
  return !!b.username;
}

const getName = (u: MemberOrGroup): string => {
  const m= u as Member;
  const g=u as Group;
  return isUser(u) ? getDisplayName(m) : g.name;
}
export interface Group {
  role: string
  name: string
}

type MemberOrGroup = Member | Group;

export enum WorkspaceDetailsTab {
  Projects = 'projects',
  Members = 'members'
}

interface Props {
  users: DetailedUser[];
  workspace: Workspace;
}

const roles = ["Admin", "Editor", "Viewer"]

const WorkspaceMembers: React.FC<Props> = ({users, workspace}: Props) => {
  const pageRef = useRef<HTMLElement>(null);

  const members: MemberOrGroup[] = [];

  const {
    settings,
    updateSettings,
  } = useSettings<WorkspaceMembersSettings>(settingsConfig);

  console.log(settings);

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

  const projectColumns = useMemo(() => {

    const nameRenderer = (value: string, record: MemberOrGroup) => {
      let member;
      if(hasDisplayName(record)){
        member = record as Member
        return (
          <>
          <div className={css.userAvatarRowItem}> 
          <Avatar size={Size.Medium} userId={member.id} />
          </div>
          <div className={css.userRowItem}>
          <div>{member.displayName}</div>
          <div>{member.username}</div>
          </div>
          </>
        )
      } else if(hasUserName(record)){
        member = record as Member
        return (
          <>
          <div className={css.userRowItem}> 
          <Avatar userId={member.id} />
          </div>
          <div className={css.userRowItem}> 
          <div>{member.username}</div>
          </div>
          </>
        )
      }
      const group = record as Group
      return (
        <>
        <div className={css.userRowItem}> 
        <img src={groupIcon} />
        </div>
        <div className={css.userRowItem}> 
        <div>{group.name}</div>
        </div>
        </>
      )
    }

    const roleRenderer = (value: string, record: Member) => {
      return ( 
      <Select
            value={record.role}
            >{
              roles.map((r) => (
                <Select.Option key={r} value={r}>
                  {r}
                </Select.Option>
              ))
            }
          </Select>
      )
    }

    const actionRenderer = () => (
      <Button>Remove</Button>
    );

    return [
      {
        dataIndex: 'username',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['name'],
        key: 'username',
        width: 100,
        filterDropdown: nameFilterSearch,
        filterIcon: tableSearchIcon,
        render: nameRenderer,
        title: 'Name',
        sorter: (a: MemberOrGroup, b: MemberOrGroup) => alphaNumericSorter(getName(a), getName(b))
      },
      {
        dataIndex: 'role',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['role'],
        width: 75,
        render: roleRenderer,
        title: 'Role',
      },
      {
      dataIndex: 'action',
      defaultWidth: DEFAULT_COLUMN_WIDTHS['action'],
      render: actionRenderer,
      title: '',
      align: 'right',
      fixed: 'right',
      }
    ] as ColumnDef<MemberOrGroup>[];
  }, [ users, workspace ]);

  console.log(projectColumns);
 
  const membersSettings = {
    columns: ['username', 'role'],
    columnWidths: [150, 50],
    tableLimit: 10,
    tableOffset: 0,
    sortDesc: true
  }

  users.forEach( u => {
    const m: Member = u;
    m.role = "Editor";
    members.push(m);
    const group = {
      role: "Admin",
      name: "group name"
    }
    members.push(group);
  })

  console.log("MEMBERS", members);
  return (
            <div className={css.membersContainer}>
            <InteractiveTable
            columns={projectColumns}
            containerRef={pageRef}
            dataSource={members}        
            pagination={getFullPaginationConfig({
              limit: settings.tableLimit,
              offset: settings.tableOffset,
            }, users.length)}
            rowKey="username"
            settings={membersSettings}
            showSorterTooltip={false}
            size="small"
            updateSettings={updateSettings as UpdateSettings<InteractiveTableSettings>}
            //onChange={handleTableChange}
          />
          </div>
  );
};

export default WorkspaceMembers;
