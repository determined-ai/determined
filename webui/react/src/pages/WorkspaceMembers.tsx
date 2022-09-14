import { Select, Button } from 'antd';
import React, { useCallback, useMemo, useRef } from 'react';
import { FilterDropdownProps } from 'antd/lib/table/interface';
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

const isMember = (obj: MemberOrGroup) => {
  const member = obj as Member;
  return !!member.username;
}

const getName = (obj: MemberOrGroup): string => {
  const member = obj as Member;
  const group = obj as Group;
  return isMember(obj) ? getDisplayName(member) : group.name;
}
export interface Group {
  role: string
  name: string
}

type MemberOrGroup = Member | Group;

interface Props {
  users: DetailedUser[];
  workspace: Workspace;
}

const roles = ["Admin", "Editor", "Viewer"]

const WorkspaceMembers: React.FC<Props> = ({users, workspace}: Props) => {
  const pageRef = useRef<HTMLElement>(null);

  const members: MemberOrGroup[] = [];
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

  const projectColumns = useMemo(() => {

    const nameRenderer = (value: string, record: MemberOrGroup) => {
      if(isMember(record)){
        const member = record as Member;
        if(member?.displayName){
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
        } else {
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
              roles.map((role) => (
                <Select.Option key={role} value={role}>
                  {role}
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
 
  const membersSettings = {
    columns: ['username', 'role'],
    columnWidths: [150, 50],
    tableLimit: 10,
    tableOffset: 0,
    sortDesc: true
  }

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
          />
          </div>
  );
};

export default WorkspaceMembers;
