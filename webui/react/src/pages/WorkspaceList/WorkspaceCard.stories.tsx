import { Meta } from '@storybook/react';
import React, { useMemo } from 'react';

import { generateTestWorkspaceData } from 'storybook/shared/generateTestData';
import { Workspace } from 'types';

import WorkspaceCard from './WorkspaceCard';

export default {
  argTypes: { workspace: { table: { disable: true } } },
  component: WorkspaceCard,
  title: 'Determined/Cards/WorkspaceCard',
} as Meta<typeof WorkspaceCard>;

const args: Partial<Workspace> = { name: 'Workspace Name', numProjects: 1 };

export const Default = (args: Partial<Workspace>): React.ReactElement => {
  const workspace = useMemo(() => generateTestWorkspaceData(args), [args]);

  return <WorkspaceCard workspace={workspace} />;
};

Default.args = args;
