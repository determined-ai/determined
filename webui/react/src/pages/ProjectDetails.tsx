import { InfoCircleOutlined } from '@ant-design/icons';
import { Space } from 'antd';
import type { TabsProps } from 'antd';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useParams } from 'react-router-dom';

import BreadcrumbBar from 'components/BreadcrumbBar';
import DynamicTabs from 'components/DynamicTabs';
import Tooltip from 'components/kit/Tooltip';
import Page from 'components/Page';
import PageNotFound from 'components/PageNotFound';
import ProjectActionDropdown from 'components/ProjectActionDropdown';
import useFeature from 'hooks/useFeature';
import usePermissions from 'hooks/usePermissions';
import { paths } from 'routes/utils';
import { getProject, getWorkspace, postUserActivity } from 'services/api';
import { V1ActivityType, V1EntityType } from 'services/api-ts-sdk';
import Icon from 'shared/components/Icon/Icon';
import Message, { MessageType } from 'shared/components/Message';
import Spinner from 'shared/components/Spinner';
import usePolling from 'shared/hooks/usePolling';
import { isEqual, isNumber } from 'shared/utils/data';
import { routeToReactUrl } from 'shared/utils/routes';
import { isNotFound } from 'shared/utils/service';
import { Project, Workspace } from 'types';
import handleError from 'utils/error';

import ExperimentList from './ExperimentList';
import F_ExperimentList from './F_ExpList/F_ExperimentList';
import css from './ProjectDetails.module.scss';
import ProjectNotes from './ProjectNotes';
import TrialsComparison from './TrialsComparison/TrialsComparison';

type Params = {
  projectId: string;
};

const ProjectDetails: React.FC = () => {
  const { projectId } = useParams<Params>();
  const trialsComparisonEnabled = useFeature().isOn('trials_comparison');
  const f_explist = useFeature().isOn('explist_v2');

  const [project, setProject] = useState<Project | undefined>();

  const permissions = usePermissions();
  const [pageError, setPageError] = useState<Error>();
  const [canceler] = useState(new AbortController());
  const pageRef = useRef<HTMLElement>(null);

  const [workspace, setWorkspace] = useState<Workspace>();

  const id = Number(projectId ?? '1');

  const postActivity = useCallback(() => {
    postUserActivity({
      activityType: V1ActivityType.GET,
      entityId: id,
      entityType: V1EntityType.PROJECT,
    });
  }, [id]);

  const fetchWorkspace = useCallback(async () => {
    const workspaceId = project?.workspaceId;
    if (!isNumber(workspaceId)) return;
    try {
      const response = await getWorkspace({ id: workspaceId });
      setWorkspace(response);
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch workspace.' });
    }
  }, [project?.workspaceId]);

  const fetchProject = useCallback(async () => {
    try {
      const response = await getProject({ id }, { signal: canceler.signal });
      setProject((prev) => {
        if (isEqual(prev, response)) return prev;
        return response;
      });
      setPageError(undefined);
    } catch (e) {
      if (!pageError) setPageError(e as Error);
    }
  }, [canceler.signal, id, pageError]);

  const tabItems: TabsProps['items'] = useMemo(() => {
    if (!project) {
      return [];
    }

    const items: TabsProps['items'] = [
      {
        children: (
          <div className={css.tabPane}>
            <div className={css.base}>
              {f_explist ? (
                <F_ExperimentList project={project} />
              ) : (
                <ExperimentList project={project} />
              )}
            </div>
          </div>
        ),
        key: 'experiments',
        label: id === 1 ? '' : 'Experiments',
      },
    ];

    if (!project.immutable && projectId) {
      items.push({
        children: (
          <div className={css.tabPane}>
            <div className={css.base}>
              <ProjectNotes fetchProject={fetchProject} project={project} />
            </div>
          </div>
        ),
        key: 'notes',
        label: 'Notes',
      });
      if (trialsComparisonEnabled) {
        items.push({
          children: (
            <div className={css.tabPane}>
              <div className={css.base}>
                <TrialsComparison projectId={projectId} workspaceId={project?.workspaceId} />
              </div>
            </div>
          ),
          key: 'trials',
          label: 'Trials',
        });
      }
    }

    return items;
  }, [fetchProject, id, project, trialsComparisonEnabled, projectId, f_explist]);

  usePolling(fetchProject, { rerunOnNewFn: true });
  usePolling(fetchWorkspace, { rerunOnNewFn: true });

  useEffect(() => {
    postActivity();
  }, [postActivity]);

  const onProjectDelete = useCallback(() => {
    if (project) routeToReactUrl(paths.workspaceDetails(project.workspaceId));
  }, [project]);

  if (isNaN(id)) {
    return <Message title={`Invalid Project ID ${projectId}`} />;
  } else if (pageError && !isNotFound(pageError)) {
    const message = `Unable to fetch Project ${projectId}`;
    return <Message title={message} type={MessageType.Warning} />;
  } else if (
    (!permissions.loading &&
      project &&
      !permissions.canViewWorkspace({ workspace: { id: project.workspaceId } })) ||
    (pageError && isNotFound(pageError))
  ) {
    return <PageNotFound />;
  } else if (!project) {
    return <Spinner spinning tip={id === 1 ? 'Loading...' : `Loading project ${id} details...`} />;
  }
  return (
    <Page
      bodyNoPadding
      containerRef={pageRef}
      // for docTitle, when id is 1 that means Uncategorized from webui/react/src/routes/routes.ts
      docTitle={id === 1 ? 'Uncategorized Experiments' : 'Project Details'}
      id="projectDetails">
      <BreadcrumbBar
        extra={
          <Space>
            {project.description && (
              <Tooltip title={project.description}>
                <InfoCircleOutlined style={{ color: 'var(--theme-float-on)' }} />
              </Tooltip>
            )}
            {id !== 1 && (
              <ProjectActionDropdown
                project={project}
                showChildrenIfEmpty={false}
                trigger={['click']}
                workspaceArchived={workspace?.archived}
                onComplete={fetchProject}
                onDelete={onProjectDelete}>
                <div style={{ cursor: 'pointer' }}>
                  <Icon name="arrow-down" size="tiny" />
                </div>
              </ProjectActionDropdown>
            )}
          </Space>
        }
        id={project.id}
        project={project}
        type="project"
      />
      {/* TODO: Clean up once we standardize page layouts */}
      <div style={{ height: '100%', padding: '16px 16px 0px 16px' }}>
        <DynamicTabs
          basePath={paths.projectDetailsBasePath(id)}
          destroyInactiveTabPane
          items={tabItems}
        />
      </div>
    </Page>
  );
};

export default ProjectDetails;
