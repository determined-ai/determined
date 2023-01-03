import { PushpinOutlined } from '@ant-design/icons';
import { Tooltip, Typography } from 'antd';
import React, { useCallback } from 'react';

import DynamicIcon from 'components/DynamicIcon';
import Link from 'components/Link';
import Avatar from 'components/UserAvatar';
import { paths } from 'routes/utils';
import Icon from 'shared/components/Icon/Icon';
import { routeToReactUrl } from 'shared/utils/routes';
import { useUsers } from 'stores/users';
import { Workspace } from 'types';
import { Loadable } from 'utils/loadable';

import WorkspaceActionDropdown from './WorkspaceActionDropdown';
import css from './WorkspaceCard.module.scss';

interface Props {
  fetchWorkspaces?: () => void;
  workspace: Workspace;
}

const WorkspaceCard: React.FC<Props> = ({ workspace, fetchWorkspaces }: Props) => {
  const handleCardClick = useCallback(() => {
    routeToReactUrl(paths.workspaceDetails(workspace.id));
  }, [workspace.id]);

  const users = Loadable.match(useUsers(), {
    Loaded: (usersPagination) => usersPagination.users,
    NotLoaded: () => [],
  });
  const user = users.find((user) => user.id === workspace.userId);

  return (
    <WorkspaceActionDropdown workspace={workspace} onComplete={fetchWorkspaces}>
      <div className={css.base} onClick={handleCardClick}>
        <DynamicIcon name={workspace.name} size={70} />
        <div className={css.info}>
          <div className={css.nameRow}>
            <h6 className={css.name}>
              <Link inherit path={paths.workspaceDetails(workspace.id)}>
                <Typography.Paragraph ellipsis={true}>{workspace.name}</Typography.Paragraph>
              </Link>
            </h6>
            {workspace.archived && (
              <Tooltip title="Archived">
                <div>
                  <Icon name="archive" size="small" />
                </div>
              </Tooltip>
            )}
          </div>
          <p className={css.projects}>
            {workspace.numProjects} project{workspace.numProjects === 1 ? '' : 's'}
          </p>
          <div className={css.avatar}>
            <Avatar user={user} />
          </div>
        </div>
        {workspace.pinned && <PushpinOutlined className={css.pinned} />}
        {!workspace.immutable && (
          <WorkspaceActionDropdown
            className={css.action}
            direction="horizontal"
            workspace={workspace}
            onComplete={fetchWorkspaces}
          />
        )}
      </div>
    </WorkspaceActionDropdown>
  );
};

export default WorkspaceCard;
