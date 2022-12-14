import { Breadcrumb, Table } from 'antd';
import React, { useCallback, useEffect, useState } from 'react';

import ExperimentIcons from 'components/ExperimentIcons';
import Grid, { GridMode } from 'components/Grid';
import Link from 'components/Link';
import Page from 'components/Page';
import ProjectCard from 'components/ProjectCard';
import Section from 'components/Section';
import { experimentNameRenderer, relativeTimeRenderer } from 'components/Table/Table';
import { paths } from 'routes/utils';
import { getExperiments } from 'services/api';
import { getCommands, getJupyterLabs, getShells, getTensorBoards } from 'services/api';
import Icon from 'shared/components/Icon/Icon';
import { dateTimeStringSorter } from 'shared/utils/sort';
import { useAuth } from 'stores/auth';
import { ShirtSize } from 'themes';
import { CommandTask, DetailedUser, ExperimentItem, ExperimentWithNames } from 'types';
import { WorkspaceState } from 'types';
import { Loadable } from 'utils/loadable';

const Dashboard: React.FC = () => {
  const [experiments, setExperiments] = useState<ExperimentWithNames[]>([]);
  const [tasks, setTasks] = useState<CommandTask[]>([]);
  const [submissions, setSubmissions] = useState<Submission[]>([]);
  const [canceler] = useState(new AbortController());
  const loadableAuth = useAuth();
  const authUser = Loadable.match(loadableAuth.auth, {
    Loaded: (auth) => auth.user,
    NotLoaded: () => undefined,
  });

  interface CommandTaskWithOptionalNames extends CommandTask {
    // not expecting Tasks to actually contain these fields,
    // but need them to list both Tasks and Experiments in the same table:
    projectId?: number;
    projectName?: string;
    workspaceName?: string;
  }
  type Submission = ExperimentWithNames | CommandTaskWithOptionalNames;

  const fetchTasks = useCallback(async (user: DetailedUser) => {
    const [commands, jupyterLabs, shells, tensorboards] = await Promise.all([
      getCommands({ orderBy: 'ORDER_BY_DESC', signal: canceler.signal, sortBy: 'SORT_BY_START_TIME', users: [user.id.toString()] }),
      getJupyterLabs({ orderBy: 'ORDER_BY_DESC', signal: canceler.signal, sortBy: 'SORT_BY_START_TIME', users: [user.id.toString()] }),
      getShells({ orderBy: 'ORDER_BY_DESC', signal: canceler.signal, sortBy: 'SORT_BY_START_TIME', users: [user.id.toString()] }),
      getTensorBoards({ orderBy: 'ORDER_BY_DESC', signal: canceler.signal, sortBy: 'SORT_BY_START_TIME', users: [user.id.toString()] }),
    ]);
    const newTasks = [...commands, ...jupyterLabs, ...shells, ...tensorboards];
    setTasks(newTasks);
  }, [canceler]);

  const fetchExperiments = useCallback(async (user: DetailedUser) => {
    const experiments = await getExperiments(
      {
        orderBy: 'ORDER_BY_DESC',
        sortBy: 'SORT_BY_START_TIME',
        userIds: [user.id],
      },
    );
    setExperiments(experiments.experiments as ExperimentWithNames[]);
  }, []);

  useEffect(() => {
    if (!authUser) return;
    fetchExperiments(authUser);
    fetchTasks(authUser);
  }, [authUser, fetchExperiments, fetchTasks]);

  useEffect(() => {
    let submissions: Submission[] = [];
    submissions = submissions.concat(experiments).concat(tasks).sort((a, b) => dateTimeStringSorter(b.startTime, a.startTime));
    setSubmissions(submissions);
  }, [experiments, tasks]);

  const mockProjects = [
    {
      archived: false,
      description: '',
      id: 168,
      immutable: false,
      lastExperimentStartedAt: new Date('2022-12-05T23:32:12.374578Z'),
      name: 'not a project',
      notes: [
        {
          contents: '',
          name: 'Untitled',
        },
      ],
      numActiveExperiments: 0,
      numExperiments: 0,
      state: WorkspaceState.Unspecified,
      userId: 1,
      workspaceId: 100,
      workspaceName: '',
    },
    {
      archived: false,
      description: '',
      id: 106,
      immutable: false,
      lastExperimentStartedAt: new Date('2022-12-05T23:32:12.374578Z'),
      name: 'Sentiment detection',
      notes: [],
      numActiveExperiments: 0,
      numExperiments: 0,
      state: WorkspaceState.Unspecified,
      userId: 2,
      workspaceId: 100,
      workspaceName: '',
    },
    {
      archived: false,
      description: '',
      id: 115,
      immutable: false,
      lastExperimentStartedAt: new Date('2022-12-05T23:32:12.374578Z'),
      name: 'Audio segmentation',
      notes: [],
      numActiveExperiments: 0,
      numExperiments: 1,
      state: WorkspaceState.Unspecified,
      userId: 2,
      workspaceId: 100,
      workspaceName: '',
    },
    {
      archived: false,
      description: '',
      id: 100,
      immutable: false,
      lastExperimentStartedAt: new Date('2022-12-05T18:54:20.475839Z'),
      name: 'Text Transcription Model',
      notes: [],
      numActiveExperiments: 0,
      numExperiments: 3,
      state: WorkspaceState.Unspecified,
      userId: 2,
      workspaceId: 100,
      workspaceName: '',
    },
  ];

  return (
    <Page title="Home">
      <Section title="Recent projects">
        <Grid count={mockProjects.length} gap={ShirtSize.Medium} minItemWidth={250} mode={GridMode.ScrollableRow}>
          {mockProjects.map((project) => (
            <ProjectCard
              curUser={authUser}
              // fetchProjects={fetchProjects}
              key={project.id}
              project={project}
            />
          ))}
        </Grid>
      </Section>
      <Section title="Recently submitted">
        <Table<Submission>
          columns={[
            {
              dataIndex: 'state',
              render: (state) => {
                return <ExperimentIcons state={state} />;
              },
              width: 1,
            },
            {
              dataIndex: 'projectId',
              render: (projectId) => {
                if (projectId) {
                  return <Icon name="experiment" />;
                } else {
                  return <Icon name="tasks" />;
                }
              },
              width: 1,
            },
            {
              dataIndex: 'name',
              render: (name, row) => {
                if (row.projectId) {
                  // only for Experiments, not Tasks:
                  return experimentNameRenderer(name, row as ExperimentItem);
                } else {
                  return name;
                }
              },
            },
            {
              dataIndex: 'projectId',
              render: (projectId, row) => {
                if (row.workspaceName && row.projectId !== 1) {
                  return (
                    <Breadcrumb>
                      <Breadcrumb.Item>
                        {row.workspaceName}
                      </Breadcrumb.Item>
                      <Breadcrumb.Item>
                        <Link path={paths.projectDetails(projectId)}>
                          {row.projectName}
                        </Link>
                      </Breadcrumb.Item>
                    </Breadcrumb>
                  );
                }
                if (row.projectName) {
                  return (
                    <Breadcrumb>
                      <Breadcrumb.Item>
                        <Link path={paths.projectDetails(projectId)}>
                          {row.projectName}
                        </Link>
                      </Breadcrumb.Item>
                    </Breadcrumb>
                  );
                }
              },
            },
            {
              dataIndex: 'startTime',
              render: relativeTimeRenderer,
            },
          ]}
          dataSource={submissions as Submission[]}
          rowKey="id"
          showHeader={false}
        />
      </Section>
    </Page>
  );
};

export default Dashboard;
