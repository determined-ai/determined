import React, { useCallback, useEffect, useState } from 'react';
import { useParams } from 'react-router';

import Grid, { GridMode } from 'components/Grid';
import Message, { MessageType } from 'components/Message';
import Page from 'components/Page';
import Section from 'components/Section';
import Spinner from 'components/Spinner';
import usePolling from 'hooks/usePolling';
import { getWorkspace, getWorkspaceProjects, isNotFound } from 'services/api';
import { ShirtSize } from 'themes';
import { Project, Workspace } from 'types';
import handleError from 'utils/error';

import ProjectCard from './WorkspaceDetails/ProjectCard';
import WorkspaceDetailsHeader from './WorkspaceDetails/WorkspaceDetailsHeader';

interface Params {
  workspaceId: string;
}

const WorkspaceDetails: React.FC = () => {
  const { workspaceId } = useParams<Params>();
  const [ workspace, setWorkspace ] = useState<Workspace>();
  const [ projects, setProjects ] = useState<Project[]>([]);
  const [ pageError, setPageError ] = useState<Error>();
  const [ isLoading, setIsLoading ] = useState(true);
  const [ canceler ] = useState(new AbortController());

  const id = parseInt(workspaceId);

  const fetchWorkspace = useCallback(async () => {
    try {
      const response = await getWorkspace({ id });
      setWorkspace(response);
    } catch (e) {
      if (!pageError) setPageError(e as Error);
    }
  }, [ id, pageError ]);

  const fetchProjects = useCallback(async () => {
    try {
      const response = await getWorkspaceProjects({ id });
      setProjects(response.projects);
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch projects.' });
    } finally {
      setIsLoading(false);
    }
  }, [ id ]);

  const fetchAll = useCallback(async () => {
    await Promise.allSettled([ fetchWorkspace(), fetchProjects() ]);
  }, [ fetchWorkspace, fetchProjects ]);

  usePolling(fetchAll);

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
      headerComponent={<WorkspaceDetailsHeader workspace={workspace} />}
      id="workspaceDetails">
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
