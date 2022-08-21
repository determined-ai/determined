import { InfoCircleOutlined } from '@ant-design/icons';
import { PageHeader, Space, Tabs, Tooltip } from 'antd';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import { useParams } from 'react-router-dom';

import BreadcrumbBar from 'components/BreadcrumbBar';
import DynamicTabs from 'components/DynamicTabs';
import Page from 'components/Page';
import { useStore } from 'contexts/Store';
import usePolling from 'hooks/usePolling';
import { paths } from 'routes/utils';
import {
  getProject,
  getWorkspace,
} from 'services/api';
import Icon from 'shared/components/Icon/Icon';
import Message, { MessageType } from 'shared/components/Message';
import Spinner from 'shared/components/Spinner';
import { isEqual, isNumber } from 'shared/utils/data';
import { isNotFound } from 'shared/utils/service';
import {
  Project, Workspace,
} from 'types';
import handleError from 'utils/error';

import ExperimentList from './ExperimentList';
import css from './ProjectDetails.module.scss';
import ProjectNotes from './ProjectNotes';
import TrialsComparison from './TrialsComparison/TrialsComparison';
import ProjectActionDropdown from './WorkspaceDetails/ProjectActionDropdown';

const { TabPane } = Tabs;

interface Params {
  projectId: string;
}

const ProjectDetails: React.FC = () => {
  const { auth: { user } } = useStore();
  const { projectId } = useParams<Params>();

  const [ project, setProject ] = useState<Project>();

  const [ pageError, setPageError ] = useState<Error>();
  const [ isLoading, setIsLoading ] = useState(true);
  const [ canceler ] = useState(new AbortController());
  const pageRef = useRef<HTMLElement>(null);

  const [ workspace, setWorkspace ] = useState<Workspace>();

  const fetchWorkspace = useCallback(async () => {
    const id = project?.workspaceId;
    if (!isNumber(id)) return;
    try {
      const response = await getWorkspace({ id });
      setWorkspace(response);
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch workspace.' });
    }
  }, [ project?.workspaceId ]);

  useEffect(() => {
    fetchWorkspace();
  }, [ fetchWorkspace ]);

  const id = parseInt(projectId);

  const fetchProject = useCallback(async () => {
    try {
      const response = await getProject({ id }, { signal: canceler.signal });
      setProject((prev) => {
        if (isEqual(prev, response)) return prev;
        return response;
      });
    } catch (e) {
      if (!pageError) setPageError(e as Error);
    } finally {
      setIsLoading(false);
    }
  }, [ canceler.signal, id, pageError ]);

  usePolling(fetchProject, { rerunOnNewFn: true });
  usePolling(fetchWorkspace, { rerunOnNewFn: true });

  if (project?.immutable) {
    return (
      <div className={css.base}>
        <PageHeader
          className={css.header}
          title="Uncategorized"
        />
        <ExperimentList project={project} />
      </div>
    );
  }

  if (isNaN(id)) {
    return <Message title={`Invalid Project ID ${projectId}`} />;
  } else if (pageError) {
    const message = isNotFound(pageError) ?
      `Unable to find Project ${projectId}` :
      `Unable to fetch Project ${projectId}`;
    return <Message title={message} type={MessageType.Warning} />;
  } else if (!project) {
    return (
      <Spinner
        tip={projectId === '1' ? 'Loading...' : `Loading project ${projectId} details...`}
      />
    );
  }
  return (
    <Page
      bodyNoPadding
      containerRef={pageRef}
      // for docTitle, when id is 1 that means Uncategorized from webui/react/src/routes/routes.ts
      docTitle={id === 1 ? 'Uncategorized Experiments' : 'Project Details'}
      id="projectDetails">
      <BreadcrumbBar
        extra={(
          <Space>
            {project.description && (
              <Tooltip title={project.description}>
                <InfoCircleOutlined style={{ color: 'var(--theme-colors-monochrome-8)' }} />

              </Tooltip>
            )}
            <ProjectActionDropdown
              curUser={user}
              project={project}
              showChildrenIfEmpty={false}
              trigger={[ 'click' ]}
              workspaceArchived={workspace?.archived}
              onComplete={fetchProject}>
              <div style={{ cursor: 'pointer' }}>
                <Icon name="arrow-down" size="tiny" />
              </div>
            </ProjectActionDropdown>
          </Space>
        )}
        id={project.id}
        project={project}
        type="project"
      />
      <DynamicTabs
        basePath={paths.projectDetails(id)}
        defaultActiveKey="trials"
        destroyInactiveTabPane
        tabBarStyle={{ height: 50, paddingLeft: 16 }}>
        <TabPane
          className={css.tabPane}
          key="experiments"
          tab="Experiments">
          <div className={css.base}>
            <ExperimentList project={project} />
          </div>
        </TabPane>
        <TabPane
          className={css.tabPane}
          key="trials"
          tab="Trials">
          <div className={css.base}>
            <TrialsComparison projectId={projectId} />
          </div>
        </TabPane>
        <TabPane
          className={css.tabPane}
          key="notes"
          tab="Notes">
          <div className={css.base}>
            <ProjectNotes fetchProject={fetchProject} project={project} />
          </div>
        </TabPane>
      </DynamicTabs>
    </Page>
  );
};

export default ProjectDetails;
