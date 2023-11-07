import Avatar from 'hew/Avatar';
import React from 'react';

import { Workspace } from 'types';

import css from './WorkspaceFilter.module.scss';

interface Props {
  workspace: Workspace;
}

const WorkspaceFilter: React.FC<Props> = ({ workspace }: Props) => {
  return (
    <div className={css.item}>
      <Avatar palette="muted" square text={workspace.name} />
      <span className={css.name}>{workspace.name}</span>
    </div>
  );
};

export default WorkspaceFilter;
