import { Select } from 'antd';
import React, { useCallback, useEffect, useState } from 'react';
import { useParams } from 'react-router';

import Grid, { GridMode } from 'components/Grid';
import Message, { MessageType } from 'components/Message';
import Page from 'components/Page';
import SelectFilter from 'components/SelectFilter';
import Spinner from 'components/Spinner';
import { useStore } from 'contexts/Store';
import { useFetchUsers } from 'hooks/useFetch';
import usePolling from 'hooks/usePolling';
import useSettings from 'hooks/useSettings';
import { getWorkspace, getWorkspaceProjects, isNotFound } from 'services/api';
import { V1GetWorkspaceProjectsRequestSortBy } from 'services/api-ts-sdk';
import { validateDetApiEnum } from 'services/utils';
import { ShirtSize } from 'themes';
import { Project, Workspace } from 'types';
import handleError from 'utils/error';

import css from './WorkspaceDetails.module.scss';
import settingsConfig, { WorkspaceDetailsSettings } from './WorkspaceDetails.settings';
import ProjectCard from './WorkspaceDetails/ProjectCard';
import WorkspaceDetailsHeader from './WorkspaceDetails/WorkspaceDetailsHeader';

const { Option } = Select;

interface Params {
  workspaceId: string;
}

enum ProjectFilters {
  All = 'ALL_PROJECTS',
  Mine = 'MY_PROJECTS',
  Others = 'OTHERS_PROJECTS'
}

const WorkspaceDetails: React.FC = () => {
  const { users, auth: { user } } = useStore();
  const { workspaceId } = useParams<Params>();
  const [ workspace, setWorkspace ] = useState<Workspace>();
  const [ projects, setProjects ] = useState<Project[]>([]);
  const [ projectFilter, setProjectFilter ] = useState<ProjectFilters>(ProjectFilters.All);
  const [ pageError, setPageError ] = useState<Error>();
  const [ isLoading, setIsLoading ] = useState(true);
  const [ canceler ] = useState(new AbortController());

  const id = parseInt(workspaceId);

  const {
    settings,
    updateSettings,
  } = useSettings<WorkspaceDetailsSettings>(settingsConfig);

  const fetchWorkspace = useCallback(async () => {
    try {
      const response = await getWorkspace({ id }, { signal: canceler.signal });
      setWorkspace(response);
    } catch (e) {
      if (!pageError) setPageError(e as Error);
    }
  }, [ canceler.signal, id, pageError ]);

  const fetchProjects = useCallback(async () => {
    try {
      const response = await getWorkspaceProjects({
        archived: settings.archived,
        id,
        limit: settings.tableLimit,
        name: settings.name,
        offset: settings.tableOffset,
        orderBy: settings.sortDesc ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC',
        sortBy: validateDetApiEnum(V1GetWorkspaceProjectsRequestSortBy, settings.sortKey),
        users: settings.user,
      }, { signal: canceler.signal });
      setProjects(response.projects);
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch projects.' });
    } finally {
      setIsLoading(false);
    }
  }, [ canceler.signal,
    id,
    settings.archived,
    settings.name,
    settings.sortDesc,
    settings.sortKey,
    settings.tableLimit,
    settings.tableOffset,
    settings.user ]);

  const fetchUsers = useFetchUsers(canceler);

  const fetchAll = useCallback(async () => {
    await Promise.allSettled([ fetchWorkspace(), fetchProjects(), fetchUsers() ]);
  }, [ fetchWorkspace, fetchProjects, fetchUsers ]);

  usePolling(fetchAll);

  const handleSelectFilter = useCallback((value) => {
    setProjectFilter(value as ProjectFilters);
  }, []);

  // useEffect(() => {
  //   switch (projectFilter) {
  //     case ProjectFilters.All:
  //       updateSettings({ user: undefined });
  //       break;
  //     case ProjectFilters.Mine:
  //       updateSettings({ user: user ? [ user.username ] : undefined });
  //       break;
  //     case ProjectFilters.Others:
  //       updateSettings({ user: users.filter(u => u.id !== user?.id).map(u => u.username) });
  //       break;
  //   }
  // }, [ projectFilter, updateSettings, user, users ]);

  useEffect(() => {
    return () => canceler.abort();
  }, [ canceler ]);

  if (isNaN(id)) {
    return <Message title={`Invalid Workspace ID ${workspaceId}`} />;
  } else if (pageError) {
    const message = isNotFound(pageError) ?
      `Unable to find Workspace ${workspaceId}` :
      `Unable to fetch Workspace ${workspaceId}`;
    return <Message title={message} type={MessageType.Warning} />;
  } else if (!workspace) {
    return <Spinner tip={`Loading workspace ${workspaceId} details...`} />;
  }

  return (
    <Page
      className={css.base}
      headerComponent={<WorkspaceDetailsHeader workspace={workspace} />}
      id="workspaceDetails">
      <div className={css.controls}>
        <SelectFilter label="View:" value={projectFilter} onSelect={handleSelectFilter}>
          <Option value={ProjectFilters.All}>All projects</Option>
          <Option value={ProjectFilters.Mine}>My projects</Option>
          <Option value={ProjectFilters.Others}>Others projects</Option>
        </SelectFilter>
      </div>
      <Spinner spinning={isLoading}>
        {projects.length !== 0 ? (
          <Grid gap={ShirtSize.medium} mode={GridMode.AutoFill}>
            {projects.map(project => (
              <ProjectCard key={project.id} project={project} />
            ))}
          </Grid>
        ) : (
          <Message
            title="No projects matching the current filters"
            type={MessageType.Empty}
          />
        )}
      </Spinner>
    </Page>
  );
};

export default WorkspaceDetails;
