import { Meta } from '@storybook/react';
import React, { useMemo } from 'react';

import { useAuth } from 'stores/auth';
import { generateTestProjectData } from 'storybook/shared/generateTestData';
import { Project } from 'types';
import { Loadable } from 'utils/loadable';

import ProjectCard from './ProjectCard';

export default {
  argTypes: {
    curUser: { table: { disable: true } },
    project: { table: { disable: true } },
    workspaceArchived: { table: { disable: true } },
  },
  component: ProjectCard,
  title: 'Determined/Cards/ProjectCard',
} as Meta<typeof ProjectCard>;

const args: Partial<Project> = { name: 'Project Name', numExperiments: 1 };

export const Default = (args: Partial<Project>): React.ReactElement => {
  const loadableAuth = useAuth();
  const user = Loadable.match(loadableAuth.auth, {
    Loaded: (auth) => auth.user,
    NotLoaded: () => undefined,
  });
  const project = useMemo(() => generateTestProjectData(args), [args]);

  return <ProjectCard curUser={user} project={project} />;
};

Default.args = args;
