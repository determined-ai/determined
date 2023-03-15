import { DownOutlined, PushpinOutlined } from '@ant-design/icons';
import { Space } from 'antd';
import React, { useCallback } from 'react';

import DynamicIcon from 'components/DynamicIcon';
import InlineEditor from 'components/InlineEditor';
import Tooltip from 'components/kit/Tooltip';
import usePermissions from 'hooks/usePermissions';
import WorkspaceActionDropdown from 'pages/WorkspaceList/WorkspaceActionDropdown';
import { patchWorkspace } from 'services/api';
import { V1Role } from 'services/api-ts-sdk';
import Icon from 'shared/components/Icon/Icon';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { UserOrGroup, Workspace } from 'types';
import handleError from 'utils/error';

import css from './WorkspaceDetailsHeader.module.scss';

interface Props {
  addableUsersAndGroups: UserOrGroup[];
  fetchWorkspace: () => void;
  rolesAssignableToScope: V1Role[];
  workspace: Workspace;
}

const WorkspaceDetailsHeader: React.FC<Props> = ({ workspace, fetchWorkspace }: Props) => {
  const { canModifyWorkspace } = usePermissions();

  const handleNameChange = useCallback(
    async (name: string) => {
      try {
        await patchWorkspace({ id: workspace.id, name });
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
    },
    [workspace.id],
  );

  return (
    <div className={css.base}>
      <Space align="center">
        <DynamicIcon name={workspace.name} size={32} />
        <h1 className={css.name}>
          <InlineEditor
            disabled={
              workspace.immutable ||
              workspace.archived ||
              !canModifyWorkspace({ workspace: workspace })
            }
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
