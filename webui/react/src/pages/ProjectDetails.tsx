import type { TabsProps } from 'antd';
import _ from 'lodash';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useParams } from 'react-router-dom';

import DynamicTabs from 'components/DynamicTabs';
import Spinner from 'components/kit/Spinner';
import { Loadable, Loaded, NotLoaded } from 'components/kit/utils/loadable';
import Message, { MessageType } from 'components/Message';
import Page, { BreadCrumbRoute } from 'components/Page';
import PageNotFound from 'components/PageNotFound';
import { useProjectActionMenu } from 'components/ProjectActionDropdown';
import useFeature from 'hooks/useFeature';
import usePermissions from 'hooks/usePermissions';
import usePolling from 'hooks/usePolling';
import { paths } from 'routes/utils';
import { getProject, postUserActivity } from 'services/api';
import { V1ActivityType, V1EntityType } from 'services/api-ts-sdk';
import workspaceStore from 'stores/workspaces';
import { Project } from 'types';
import { useObservable } from 'utils/observable';
import { routeToReactUrl } from 'utils/routes';
import { isNotFound } from 'utils/service';

import ExperimentList from './ExperimentList';
import F_ExperimentList from './F_ExpList/F_ExperimentList';
import css from './ProjectDetails.module.scss';
import ProjectNotes from './ProjectNotes';

type Params = {
  projectId: string;
};

const ProjectDetails: React.FC = () => {
  const { projectId } = useParams<Params>();
  const f_explist = useFeature().isOn('explist_v2');

  const [project, setProject] = useState<Project | undefined>();

  const permissions = usePermissions();
  const [pageError, setPageError] = useState<Error>();
  const [canceler] = useState(new AbortController());
  const pageRef = useRef<HTMLElement>(null);

  const id = Number(projectId ?? '1');

  const fetchProject = useCallback(async () => {
    try {
      const response = await getProject({ id }, { signal: canceler.signal });
      setProject((prev) => {
        if (_.isEqual(prev, response)) return prev;
        return response;
      });
      setPageError(undefined);
    } catch (e) {
      if (!pageError) setPageError(e as Error);
    }
  }, [canceler.signal, id, pageError]);

  const onProjectDelete = useCallback(() => {
    if (project) routeToReactUrl(paths.workspaceDetails(project.workspaceId));
  }, [project]);

  const workspace = Loadable.getOrElse(
    undefined,
    useObservable(workspaceStore.getWorkspace(project ? Loaded(project.workspaceId) : NotLoaded)),
  );

  const postActivity = useCallback(() => {
    postUserActivity({
      activityType: V1ActivityType.GET,
      entityId: id,
      entityType: V1EntityType.PROJECT,
    });
  }, [id]);

  const { contextHolders, menu, onClick } = useProjectActionMenu({
    onDelete: onProjectDelete,
    onEdit: fetchProject,
    onMove: fetchProject,
    project,
    workspaceArchived: workspace?.archived,
  });

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
                <F_ExperimentList key={projectId} project={project} />
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
    }

    return items;
  }, [fetchProject, id, project, projectId, f_explist]);

  usePolling(fetchProject, { rerunOnNewFn: true });

  useEffect(() => {
    postActivity();
  }, [postActivity]);

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

  const pageBreadcrumb: BreadCrumbRoute[] =
    project.workspaceId !== 1
      ? [
          {
            breadcrumbName: project.workspaceName,
            path: paths.workspaceDetails(project.workspaceId),
          },

          {
            breadcrumbName: project.name,
            path: paths.projectDetails(project.id),
          },
        ]
      : [
          {
            breadcrumbName: 'Uncategorized Experiments',
            path: paths.projectDetails(project.id),
          },
        ];
  return (
    <Page
      breadcrumb={pageBreadcrumb}
      containerRef={pageRef}
      // for docTitle, when id is 1 that means Uncategorized from webui/react/src/routes/routes.ts
      docTitle={id === 1 ? 'Uncategorized Experiments' : 'Project Details'}
      id="projectDetails"
      menuItems={menu.length > 0 ? menu : undefined}
      noScroll
      onClickMenu={onClick}>
      <DynamicTabs
        basePath={paths.projectDetailsBasePath(id)}
        destroyInactiveTabPane
        items={tabItems}
      />
      {contextHolders}
    </Page>
  );
};

export default ProjectDetails;
