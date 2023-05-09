import React, { useCallback, useEffect, useState } from 'react';

import ExperimentIcons from 'components/ExperimentIcons';
import JupyterLabButton from 'components/JupyterLabButton';
import Breadcrumb from 'components/kit/Breadcrumb';
import Card from 'components/kit/Card';
import Empty from 'components/kit/Empty';
import Icon from 'components/kit/Icon';
import Link from 'components/Link';
import Page from 'components/Page';
import ProjectCard from 'components/ProjectCard';
import Section from 'components/Section';
import ResponsiveTable from 'components/Table/ResponsiveTable';
import {
  experimentNameRenderer,
  relativeTimeRenderer,
  taskNameRenderer,
  taskTypeRenderer,
} from 'components/Table/Table';
import usePermissions from 'hooks/usePermissions';
import { paths } from 'routes/utils';
import {
  getCommands,
  getExperiments,
  getJupyterLabs,
  getProjectsByUserActivity,
  getShells,
  getTensorBoards,
} from 'services/api';
import Spinner from 'shared/components/Spinner';
import usePolling from 'shared/hooks/usePolling';
import { ErrorType } from 'shared/utils/error';
import { dateTimeStringSorter } from 'shared/utils/sort';
import userStore from 'stores/users';
import { CommandTask, DetailedUser, ExperimentItem, Project } from 'types';
import handleError from 'utils/error';
import { Loadable } from 'utils/loadable';
import { useObservable } from 'utils/observable';

import css from './Dashboard.module.scss';

const SUBMISSIONS_FETCH_LIMIT = 25;
const PROJECTS_FETCH_LIMIT = 5;
const DISPLAY_LIMIT = 25;

const Dashboard: React.FC = () => {
  const [experiments, setExperiments] = useState<ExperimentItem[]>([]);
  const [tasks, setTasks] = useState<CommandTask[]>([]);
  const [projects, setProjects] = useState<Project[]>([]);
  const [submissions, setSubmissions] = useState<Submission[]>([]);
  const [canceler] = useState(new AbortController());
  const [submissionsLoading, setSubmissionsLoading] = useState<boolean>(true);
  const [projectsLoading, setProjectsLoading] = useState<boolean>(true);
  const currentUser = Loadable.getOrElse(undefined, useObservable(userStore.currentUser));
  const { canCreateNSC } = usePermissions();
  type Submission = ExperimentItem & CommandTask;

  const fetchTasks = useCallback(
    async (user: DetailedUser) => {
      const results = await Promise.allSettled([
        getCommands({
          limit: SUBMISSIONS_FETCH_LIMIT,
          orderBy: 'ORDER_BY_DESC',
          signal: canceler.signal,
          sortBy: 'SORT_BY_START_TIME',
          users: [user.id.toString()],
        }),
        getJupyterLabs({
          limit: SUBMISSIONS_FETCH_LIMIT,
          orderBy: 'ORDER_BY_DESC',
          signal: canceler.signal,
          sortBy: 'SORT_BY_START_TIME',
          users: [user.id.toString()],
        }),
        getShells({
          limit: SUBMISSIONS_FETCH_LIMIT,
          orderBy: 'ORDER_BY_DESC',
          signal: canceler.signal,
          sortBy: 'SORT_BY_START_TIME',
          users: [user.id.toString()],
        }),
        getTensorBoards({
          limit: SUBMISSIONS_FETCH_LIMIT,
          orderBy: 'ORDER_BY_DESC',
          signal: canceler.signal,
          sortBy: 'SORT_BY_START_TIME',
          users: [user.id.toString()],
        }),
      ]);
      const newTasks = results.reduce((acc, current) => {
        if (current.status === 'fulfilled') return acc.concat(current.value);
        return acc;
      }, [] as CommandTask[]);
      setTasks(newTasks);
    },
    [canceler],
  );

  const fetchExperiments = useCallback(
    async (user: DetailedUser) => {
      try {
        const response = await getExperiments(
          {
            limit: SUBMISSIONS_FETCH_LIMIT,
            orderBy: 'ORDER_BY_DESC',
            sortBy: 'SORT_BY_START_TIME',
            users: [user.id.toString()],
          },
          {
            signal: canceler.signal,
          },
        );
        setExperiments(response.experiments);
      } catch (e) {
        handleError(e, {
          publicSubject: 'Error fetching experiments for dashboard',
          silent: false,
          type: ErrorType.Api,
        });
      }
    },
    [canceler],
  );

  const fetchProjects = useCallback(async () => {
    try {
      const projects = await getProjectsByUserActivity(
        { limit: PROJECTS_FETCH_LIMIT },
        {
          signal: canceler.signal,
        },
      );
      setProjects(projects);
      setProjectsLoading(false);
    } catch (e) {
      handleError(e, {
        publicSubject: 'Error fetching projects for dashboard',
        silent: false,
        type: ErrorType.Api,
      });
    }
  }, [canceler]);

  const fetchSubmissions = useCallback(async () => {
    if (!currentUser) return;
    await Promise.allSettled([fetchExperiments(currentUser), fetchTasks(currentUser)]);
    setSubmissionsLoading(false);
  }, [currentUser, fetchExperiments, fetchTasks]);

  const fetchAll = useCallback(() => {
    fetchProjects();
    fetchSubmissions();
  }, [fetchSubmissions, fetchProjects]);

  const { stopPolling } = usePolling(fetchAll, { rerunOnNewFn: true });

  useEffect(() => {
    setSubmissions(
      (experiments as Submission[])
        .concat(tasks as Submission[])
        .sort((a, b) => dateTimeStringSorter(b.startTime, a.startTime))
        .slice(0, DISPLAY_LIMIT),
    );
  }, [experiments, tasks]);

  useEffect(() => {
    return () => {
      canceler.abort();
      stopPolling();
    };
  }, [canceler, stopPolling]);

  if (projectsLoading && submissionsLoading) {
    return (
      <Page options={<JupyterLabButton enabled={canCreateNSC} />} title="Home">
        <Spinner center />
      </Page>
    );
  }

  return (
    <Page options={<JupyterLabButton enabled={canCreateNSC} />} title="Home">
      {projectsLoading ? (
        <Section>
          <Spinner center />
        </Section>
      ) : projects.length > 0 ? (
        // hide Projects header when empty:
        <Section title="Recently Viewed Projects">
          <Card.Group size="small" wrap={false}>
            {projects.map((project) => (
              <ProjectCard
                fetchProjects={fetchProjects}
                key={project.id}
                project={project}
                showWorkspace
              />
            ))}
          </Card.Group>
        </Section>
      ) : null}
      {/* show Submissions header even when empty: */}
      <Section title="Your Recent Submissions">
        {submissionsLoading ? (
          <Spinner center />
        ) : submissions.length > 0 ? (
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
                render: (projectId, row, index) => {
                  if (projectId) {
                    return <Icon name="experiment" title="Experiment" />;
                  } else {
                    return taskTypeRenderer(row.type, row, index);
                  }
                },
                width: 1,
              },
              {
                dataIndex: 'name',
                render: (name, row, index) => {
                  if (row.projectId) {
                    // only for Experiments, not Tasks:
                    return experimentNameRenderer(name, row);
                  } else {
                    return taskNameRenderer(row.id, row, index);
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
                          <Link path={paths.workspaceDetails(row.workspaceId)}>
                            {row.workspaceName}
                          </Link>
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
            loading={submissionsLoading}
            pagination={false}
            rowKey="id"
            showHeader={false}
            size="middle"
          />
        ) : (
          <Empty
            description={
              <>
                Your recent experiments and tasks will show up here.{' '}
                <Link external path={paths.docs('/quickstart-mdldev.html')}>
                  Get started
                </Link>
              </>
            }
            icon="experiment"
            title="No submissions"
          />
        )}
      </Section>
    </Page>
  );
};

export default Dashboard;
