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
import {
  getCommands,
  getExperiments,
  getJupyterLabs,
  getProjectsByUserActivity,
  getShells,
  getTensorBoards,
} from 'services/api';
import Icon from 'shared/components/Icon/Icon';
import { dateTimeStringSorter } from 'shared/utils/sort';
import { useAuth } from 'stores/auth';
import { ShirtSize } from 'themes';
import { CommandTask, DetailedUser, ExperimentItem, Project } from 'types';
import { Loadable } from 'utils/loadable';
const Dashboard: React.FC = () => {
  const [experiments, setExperiments] = useState<ExperimentItem[]>([]);
  const [tasks, setTasks] = useState<CommandTask[]>([]);
  const [projects, setProjects] = useState<Project[]>([]);
  const [submissions, setSubmissions] = useState<Submission[]>([]);
  const [canceler] = useState(new AbortController());
  const loadableAuth = useAuth();
  const authUser = Loadable.match(loadableAuth.auth, {
    Loaded: (auth) => auth.user,
    NotLoaded: () => undefined,
  });

  type Submission = ExperimentItem & CommandTask;

  const fetchTasks = useCallback(
    async (user: DetailedUser) => {
      const [commands, jupyterLabs, shells, tensorboards] = await Promise.all([
        getCommands({
          orderBy: 'ORDER_BY_DESC',
          signal: canceler.signal,
          sortBy: 'SORT_BY_START_TIME',
          users: [user.id.toString()],
        }),
        getJupyterLabs({
          orderBy: 'ORDER_BY_DESC',
          signal: canceler.signal,
          sortBy: 'SORT_BY_START_TIME',
          users: [user.id.toString()],
        }),
        getShells({
          orderBy: 'ORDER_BY_DESC',
          signal: canceler.signal,
          sortBy: 'SORT_BY_START_TIME',
          users: [user.id.toString()],
        }),
        getTensorBoards({
          orderBy: 'ORDER_BY_DESC',
          signal: canceler.signal,
          sortBy: 'SORT_BY_START_TIME',
          users: [user.id.toString()],
        }),
      ]);
      const newTasks = [...commands, ...jupyterLabs, ...shells, ...tensorboards];
      setTasks(newTasks);
    },
    [canceler],
  );

  const fetchExperiments = useCallback(async (user: DetailedUser) => {
    const response = await getExperiments({
      orderBy: 'ORDER_BY_DESC',
      sortBy: 'SORT_BY_START_TIME',
      userIds: [user.id],
    });
    setExperiments(response.experiments);
  }, []);

  const fetchProjects = useCallback(async () => {
    const projects = await getProjectsByUserActivity({});
    setProjects(projects);
  }, []);

  useEffect(() => {
    fetchProjects();
    if (!authUser) return;
    fetchExperiments(authUser);
    fetchTasks(authUser);
  }, [authUser, fetchExperiments, fetchTasks, fetchProjects]);

  useEffect(() => {
    let submissions: Submission[] = [];
    submissions = submissions
      .concat(experiments as Submission[])
      .concat(tasks as Submission[])
      .sort((a, b) => dateTimeStringSorter(b.startTime, a.startTime));
    setSubmissions(submissions);
  }, [experiments, tasks]);

  return (
    <Page title="Home">
      <Section title="Recent projects">
        <Grid
          count={projects.length}
          gap={ShirtSize.Medium}
          minItemWidth={250}
          mode={GridMode.ScrollableRow}>
          {projects.map((project) => (
            <ProjectCard
              curUser={authUser}
              fetchProjects={fetchProjects}
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
                  return experimentNameRenderer(name, row);
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
                      <Breadcrumb.Item>{row.workspaceName}</Breadcrumb.Item>
                      <Breadcrumb.Item>
                        <Link path={paths.projectDetails(projectId)}>{row.projectName}</Link>
                      </Breadcrumb.Item>
                    </Breadcrumb>
                  );
                }
                if (row.projectName) {
                  return (
                    <Breadcrumb>
                      <Breadcrumb.Item>
                        <Link path={paths.projectDetails(projectId)}>{row.projectName}</Link>
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
          dataSource={submissions}
          rowKey="id"
          showHeader={false}
        />
      </Section>
    </Page>
  );
};

export default Dashboard;
