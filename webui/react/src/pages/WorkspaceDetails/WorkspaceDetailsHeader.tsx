import { DownOutlined, PushpinOutlined } from '@ant-design/icons';
import { Space } from 'antd';
import React from 'react';

import DynamicIcon from 'components/DynamicIcon';
import Tooltip from 'components/kit/Tooltip';
import WorkspaceActionDropdown from 'pages/WorkspaceList/WorkspaceActionDropdown';
import { V1Role } from 'services/api-ts-sdk';
import Icon from 'shared/components/Icon/Icon';
import { UserOrGroup, Workspace } from 'types';

import css from './WorkspaceDetailsHeader.module.scss';

interface Props {
  addableUsersAndGroups: UserOrGroup[];
  fetchWorkspace: () => void;
  rolesAssignableToScope: V1Role[];
  workspace: Workspace;
}

const WorkspaceDetailsHeader: React.FC<Props> = ({ workspace, fetchWorkspace }: Props) => {
  return (
    <div className={css.base}>
      <Space align="center">
        <DynamicIcon name={workspace.name} size={32} />
        <h1 className={css.name}>{workspace.name}</h1>
        {workspace.archived && (
          <Tooltip title="Archived">
            <div>
              <Icon name="archive" size="small" />
            </div>
          </Tooltip>
        )}
        {workspace.pinned && (
          <Tooltip title="Pinned to sidebar">
            <PushpinOutlined className={css.pinned} />
          </Tooltip>
        )}
        {!workspace.immutable && (
          <WorkspaceActionDropdown
            trigger={['click']}
            workspace={workspace}
            onComplete={fetchWorkspace}>
            <DownOutlined className={css.dropdown} />
          </WorkspaceActionDropdown>
        )}
      </Space>
    </div>
  );
};

export default WorkspaceDetailsHeader;
