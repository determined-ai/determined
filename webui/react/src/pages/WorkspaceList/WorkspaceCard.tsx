import { Tooltip, Typography } from 'antd';
import React, { useMemo } from 'react';

import Avatar from 'components/Avatar';
import Icon from 'components/Icon';
import Link from 'components/Link';
import { paths } from 'routes/utils';
import { DetailedUser, Workspace } from 'types';

import WorkspaceActionDropdown from './WorkspaceActionDropdown';
import css from './WorkspaceCard.module.scss';

interface Props {
  curUser?: DetailedUser;
  fetchWorkspaces?: () => void;
  workspace: Workspace;
}

const WorkspaceCard: React.FC<Props> = ({ workspace, curUser, fetchWorkspaces }: Props) => {

  const nameAcronym = useMemo(() => {
    return workspace.name
      .split(/\s/).reduce((response, word) => response += word.slice(0, 1), '')
      .slice(0, 3);
  }, [ workspace.name ]);

  return (
    <WorkspaceActionDropdown
      curUser={curUser}
      fetchWorkspaces={fetchWorkspaces}
      workspace={workspace}>
      <div className={css.base}>
        <div className={css.icon}>
          <span>{nameAcronym}</span>
        </div>
        <div className={css.info}>
          <div className={css.nameRow}>
            <h6 className={css.name}>
              <Link inherit path={paths.workspaceDetails(workspace.id)}>
                <Typography.Paragraph ellipsis={{ rows: 1, tooltip: true }}>
                  {workspace.name}
                </Typography.Paragraph>
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
          <div className={css.avatar}><Avatar username={workspace.username} /></div>
        </div>
        {!workspace.immutable && (
          <WorkspaceActionDropdown
            className={css.action}
            curUser={curUser}
            direction="horizontal"
            fetchWorkspaces={fetchWorkspaces}
            workspace={workspace}
          />
        )}
      </div>
    </WorkspaceActionDropdown>
  );
};

export default WorkspaceCard;
