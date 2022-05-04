import { DownOutlined, PushpinOutlined } from '@ant-design/icons';
import { Button, Space, Tooltip } from 'antd';
import React, { useCallback } from 'react';

import Icon from 'components/Icon';
import InlineEditor from 'components/InlineEditor';
import WorkspaceIcon from 'components/WorkspaceIcon';
import useModalProjectCreate from 'hooks/useModal/Project/useModalProjectCreate';
import WorkspaceActionDropdown from 'pages/WorkspaceList/WorkspaceActionDropdown';
import { DetailedUser, Workspace } from 'types';

import css from './WorkspaceDetailsHeader.module.scss';

interface Props {
  curUser?: DetailedUser;
  fetchWorkspace: () => void;
  workspace: Workspace;
}

const WorkspaceDetailsHeader: React.FC<Props> = ({ workspace, curUser, fetchWorkspace }: Props) => {
  const { modalOpen: openProjectCreate } = useModalProjectCreate({ workspaceId: workspace.id });

  const handleProjectCreateClick = useCallback(() => {
    openProjectCreate();
  }, [ openProjectCreate ]);

  return (
    <div className={css.base}>
      <Space align="center">
        <WorkspaceIcon name={workspace.name} size={32} />
        <div className={css.nameRow}>
          <h1 className={css.name}>
            <InlineEditor
              disabled={workspace.immutable ||
                (!curUser?.isAdmin && curUser?.username !== workspace.username)}
              maxLength={80}
              value={workspace.name}
            />
          </h1>
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
        </div>
        {!workspace.immutable && (
          <WorkspaceActionDropdown
            curUser={curUser}
            trigger={[ 'click' ]}
            workspace={workspace}
            onComplete={fetchWorkspace}>
            <DownOutlined style={{ fontSize: 12 }} />
          </WorkspaceActionDropdown>
        )}
      </Space>
      {!workspace.immutable &&
        <Button onClick={handleProjectCreateClick}>New Project</Button>}
    </div>
  );
};

export default WorkspaceDetailsHeader;
