import React, { useMemo } from 'react';

import Avatar from 'components/Avatar';
import Link from 'components/Link';
import { paths } from 'routes/utils';
import { DetailedUser, Workspace } from 'types';

import WorkspaceActionDropdown from './WorkspaceActionDropdown';
import css from './WorkspaceCard.module.scss';

interface Props {
  curUser?: DetailedUser;
  workspace: Workspace;
}

const WorkspaceCard: React.FC<Props> = ({ workspace, curUser }: Props) => {

  const nameAcronym = useMemo(() => {
    return workspace.name
      .split(/\s/).reduce((response, word) => response += word.slice(0, 1), '')
      .slice(0, 3);
  }, [ workspace.name ]);

  return (
    <WorkspaceActionDropdown curUser={curUser} workspace={workspace}>
      <div className={css.base}>
        <div className={css.icon}>
          <span>{nameAcronym}</span>
        </div>
        <div className={css.info}>
          <h6 className={css.name}>
            <Link inherit path={paths.workspaceDetails(workspace.id)}>
              {workspace.name}
            </Link>
          </h6>
          <div className={css.avatar}><Avatar username={workspace.username} /></div>
        </div>
        {!workspace.immutable && (
          <WorkspaceActionDropdown
            className={css.action}
            curUser={curUser}
            direction="horizontal"
            workspace={workspace}
          />
        )}
      </div>
    </WorkspaceActionDropdown>
  );
};

export default WorkspaceCard;
