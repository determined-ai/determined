import { DownOutlined, PushpinOutlined } from '@ant-design/icons';
import { Button, Space, Tooltip } from 'antd';
import React, { useCallback } from 'react';

import DynamicIcon from 'components/DynamicIcon';
import InlineEditor from 'components/InlineEditor';
import useModalProjectCreate from 'hooks/useModal/Project/useModalProjectCreate';
import WorkspaceActionDropdown from 'pages/WorkspaceList/WorkspaceActionDropdown';
import { patchWorkspace } from 'services/api';
import Icon from 'shared/components/Icon/Icon';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { DetailedUser, Workspace } from 'types';
import handleError from 'utils/error';

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

  const handleNameChange = useCallback(async (name: string) => {
    try {
      await patchWorkspace({ id: workspace.id, name: name });
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to edit workspace.',
        silent: false,
        type: ErrorType.Server,
      });
      return e as Error;
    }
  }, [ workspace.id ]);

  return (
    <div className={css.base}>
      <Space align="center">
        <DynamicIcon name={workspace.name} size={32} />
        <h1 className={css.name}>
          <InlineEditor
            disabled={workspace.immutable ||
                 workspace.archived
                || (!curUser?.isAdmin && curUser?.id !== workspace.userId)}
            maxLength={80}
            value={workspace.name}
            onSave={handleNameChange}
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
        {!workspace.immutable && (
          <WorkspaceActionDropdown
            curUser={curUser}
            trigger={[ 'click' ]}
            workspace={workspace}
            onComplete={fetchWorkspace}>
            <DownOutlined className={css.dropdown} />
          </WorkspaceActionDropdown>
        )}
      </Space>
      {(!workspace.immutable && !workspace.archived) &&
        <Button onClick={handleProjectCreateClick}>New Project</Button>}
    </div>
  );
};

export default WorkspaceDetailsHeader;
