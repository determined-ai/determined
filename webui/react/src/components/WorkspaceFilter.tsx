import React from 'react';

import DynamicIcon from 'components/DynamicIcon';
import css from 'components/WorkspaceFilter.module.scss';
import { Workspace } from 'types';

interface Props {
  workspace: Workspace;
}

const WorkspaceFilter: React.FC<Props> = ({ workspace }: Props) => {
  return (
    <div className={css.item}>
      <DynamicIcon name={workspace.name} size={24} />
      <span className={css.name}>{workspace.name}</span>
    </div>
  );
};

export default WorkspaceFilter;
