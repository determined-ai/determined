import { InfoCircleOutlined } from '@ant-design/icons';
import { Space, Tabs, Tooltip } from 'antd';
import React, { useCallback, useState } from 'react';

import BreadcrumbBar from 'components/BreadcrumbBar';
import Icon from 'components/Icon';
import PageHeader from 'components/PageHeader';
import ProjectActionDropdown from 'pages/WorkspaceDetails/ProjectActionDropdown';
import { DetailedUser, Project } from 'types';
import { sentenceToCamelCase } from 'utils/string';

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
  const [ activeTab, setActiveTab ] = useState<TabInfo>(tabs[0]);

  const handleTabSwitch = useCallback((tabKey: string) => {
    setActiveTab(tabs.find(tab => sentenceToCamelCase(tab.title) === tabKey) ?? tabs[0]);
  }, [ tabs ]);

  if (project.immutable) {
    const experimentsTab = tabs.find(tab => tab.title === 'Experiments');
    return (
      <div className={css.base}>
        <PageHeader
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
              trigger={[ 'click' ]}
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
        {tabs.map(tabInfo => {
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
