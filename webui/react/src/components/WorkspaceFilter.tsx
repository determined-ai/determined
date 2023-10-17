import React from 'react';

import Avatar from 'components/kit/Avatar';
import { Workspace } from 'types';

import css from './WorkspaceFilter.module.scss';

interface Props {
  workspace: Workspace;
}

const WorkspaceFilter: React.FC<Props> = ({ workspace }: Props) => {
  return (
    <div className={css.item}>
      <Avatar square text={workspace.name} textColor="black" />
      <span className={css.name}>{workspace.name}</span>
    </div>
  );
};

export default WorkspaceFilter;
