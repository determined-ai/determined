import { Breadcrumb, Button } from 'antd';
import React, { useCallback, useEffect, useState } from 'react';

import ExperimentIcons from 'components/ExperimentIcons';
import Grid, { GridMode } from 'components/Grid';
import Link from 'components/Link';
import Page from 'components/Page';
import ProjectCard from 'components/ProjectCard';
import Section from 'components/Section';
import ResponsiveTable from 'components/Table/ResponsiveTable';
import { experimentNameRenderer, relativeTimeRenderer } from 'components/Table/Table';
import useModalJupyterLab from 'hooks/useModal/JupyterLab/useModalJupyterLab';
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

import css from './Dashboard.module.scss';

const FETCH_LIMIT = 12;

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
  const { contextHolder: modalJupyterLabContextHolder, modalOpen: openJupyterLabModal } =
    useModalJupyterLab();
  type Submission = ExperimentItem & CommandTask;

  const fetchTasks = useCallback(
    async (user: DetailedUser) => {
      const [commands, jupyterLabs, shells, tensorboards] = await Promise.all([
        getCommands({
          limit: FETCH_LIMIT,
          orderBy: 'ORDER_BY_DESC',
          signal: canceler.signal,
          sortBy: 'SORT_BY_START_TIME',
          users: [user.id.toString()],
        }),
        getJupyterLabs({
          limit: FETCH_LIMIT,
          orderBy: 'ORDER_BY_DESC',
          signal: canceler.signal,
          sortBy: 'SORT_BY_START_TIME',
          users: [user.id.toString()],
        }),
        getShells({
          limit: FETCH_LIMIT,
          orderBy: 'ORDER_BY_DESC',
          signal: canceler.signal,
          sortBy: 'SORT_BY_START_TIME',
          users: [user.id.toString()],
        }),
        getTensorBoards({
          limit: FETCH_LIMIT,
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
      limit: FETCH_LIMIT,
      orderBy: 'ORDER_BY_DESC',
      sortBy: 'SORT_BY_START_TIME',
      userIds: [user.id],
    }, {
      signal: canceler.signal,
    });
    setExperiments(response.experiments);
  }, [canceler]);

  const fetchProjects = useCallback(async () => {
    const projects = await getProjectsByUserActivity({ limit: FETCH_LIMIT },
      {
        signal: canceler.signal,
      });
    setProjects(projects);
  }, [canceler]);

  useEffect(() => {
    fetchProjects();
    if (!authUser) return;
    fetchExperiments(authUser);
    fetchTasks(authUser);
  }, [authUser, fetchExperiments, fetchTasks, fetchProjects]);

  useEffect(() => {
    setSubmissions(
      (experiments as Submission[])
        .concat(tasks as Submission[])
        .sort((a, b) => dateTimeStringSorter(b.startTime, a.startTime)));
  }, [experiments, tasks]);

  const JupyterLabButton = () => {
    return (
      <Button onClick={() => openJupyterLabModal()}>
        Launch JupyterLab
      </Button>
    );
  };

  return (
    <Page
      options={<JupyterLabButton />}
      title="Home">
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
        <ResponsiveTable<Submission>
          className={css.table}
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
                  return <Icon name="experiment" title="Experiment" />;
                } else {
                  return <Icon name="tasks" title="Task" />;
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
                if (row.workspaceId && row.projectId !== 1) {
                  return (
                    <Breadcrumb>
                      <Breadcrumb.Item>
                        <Link path={paths.workspaceDetails(row.workspaceId)}>{row.workspaceName}</Link>
                      </Breadcrumb.Item>
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
      {modalJupyterLabContextHolder}
    </Page>
  );
};

export default Dashboard;
