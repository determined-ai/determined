import { InfoCircleOutlined } from '@ant-design/icons';
import { Space, Tabs, Tooltip } from 'antd';
import React, { useCallback, useEffect, useState } from 'react';
import { useHistory } from 'react-router';

import BreadcrumbBar from 'components/BreadcrumbBar';
import PageHeader from 'components/PageHeader';
import ProjectActionDropdown from 'pages/WorkspaceDetails/ProjectActionDropdown';
import { paths } from 'routes/utils';
import { getWorkspace } from 'services/api';
import Icon from 'shared/components/Icon/Icon';
import { sentenceToCamelCase } from 'shared/utils/string';
import { DetailedUser, Project, Workspace } from 'types';
import handleError from 'utils/error';

import css from './ProjectDetailsTabs.module.scss';

const { TabPane } = Tabs;

export interface TabInfo {
  body: React.ReactNode;
  options?: React.ReactNode;
  title: string;
}

interface Props {
  curUser?: DetailedUser;
  fetchProject: () => void;
  project: Project;
  tabs: TabInfo[];
}

const ProjectDetailsTabs: React.FC<Props> = (
  { project, tabs, fetchProject, curUser }: Props,
) => {
  const history = useHistory();
  const [ workspace, setWorkspace ] = useState<Workspace>();
  const [ activeTab, setActiveTab ] = useState<TabInfo>(tabs[0]);

  const basePath = paths.projectDetails(project.id);

  const fetchWorkspace = useCallback(async () => {
    try {
      const response = await getWorkspace({ id: project.workspaceId });
      setWorkspace(response);
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch workspace.' });
    }
  }, [ project.workspaceId ]);

  const handleTabSwitch = useCallback((tabKey: string) => {
    setActiveTab(tabs.find((tab) => sentenceToCamelCase(tab.title) === tabKey) ?? tabs[0]);
  }, [ tabs ]);

  useEffect(() => {
    history.replace(`${basePath}/${sentenceToCamelCase(activeTab.title)}`);
  }, [ activeTab.title, basePath, history ]);

  /**
   * prevents stale tab content, e.g. archived state
   */
  useEffect(() =>
    setActiveTab(
      (curTab) => tabs.find((tab) => tab.title === curTab.title) ?? tabs[0],
    ), [ tabs ]);

  useEffect(() => {
    fetchWorkspace();
  }, [ fetchWorkspace ]);

  if (project.immutable) {
    const experimentsTab = tabs.find((tab) => tab.title === 'Experiments');
    return (
      <div className={css.base}>
        <PageHeader
          className={css.header}
          options={experimentsTab?.options}
          title="Uncategorized"
        />
        {experimentsTab?.body}
      </div>
    );
  }

  return (
    <>
      <BreadcrumbBar
        extra={(
          <Space>
            {project.description && (
              <Tooltip title={project.description}>
                <InfoCircleOutlined style={{ color: 'var(--theme-colors-monochrome-8)' }} />
              </Tooltip>
            )}
            <ProjectActionDropdown
              curUser={curUser}
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
      <Tabs
        defaultActiveKey={sentenceToCamelCase(tabs[0].title)}
        tabBarExtraContent={activeTab.options}
        tabBarStyle={{ height: 50, paddingLeft: 16 }}
        onChange={handleTabSwitch}>
        {tabs.map((tabInfo) => {
          return (
            <TabPane
              className={css.tabPane}
              key={sentenceToCamelCase(tabInfo.title)}
              tab={tabInfo.title}>
              <div className={css.base}>
                {tabInfo.body}
              </div>
            </TabPane>
          );
        })}
      </Tabs>
    </>
  );
};

export default ProjectDetailsTabs;
