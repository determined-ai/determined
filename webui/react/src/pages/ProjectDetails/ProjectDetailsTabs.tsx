import { Tabs } from 'antd';
import React, { useCallback, useEffect, useState } from 'react';

import BreadcrumbBar from 'components/BreadcrumbBar';
import Icon from 'components/Icon';
import PageHeader from 'components/PageHeader';
import { UpdateSettings } from 'hooks/useSettings';
import { ProjectDetailsSettings } from 'pages/ProjectDetails.settings';
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
  settings: ProjectDetailsSettings;
  tabs: TabInfo[];
  updateSettings: UpdateSettings<ProjectDetailsSettings>
}

const ProjectDetailsTabs: React.FC<Props> = (
  { project, tabs, settings, updateSettings, fetchProject, curUser }: Props,
) => {
  const [ activeTab, setActiveTab ] = useState<TabInfo>(
    settings.tab ?
      tabs.find(tab => sentenceToCamelCase(tab.title) === settings.tab) ?? tabs[0] :
      tabs[0],
  );

  const handleTabSwitch = useCallback((tabKey: string) => {
    setActiveTab(tabs.find(tab => sentenceToCamelCase(tab.title) === tabKey) ?? tabs[0]);
  }, [ tabs ]);

  useEffect(() => {
    handleTabSwitch(sentenceToCamelCase(activeTab.title));
  }, [ activeTab.title, handleTabSwitch, updateSettings ]);

  useEffect(() => {
    updateSettings({ tab: sentenceToCamelCase(activeTab.title) });
  }, [ activeTab.title, updateSettings ]);

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
          <ProjectActionDropdown curUser={curUser} project={project} onComplete={fetchProject}>
            <div style={{ cursor: 'pointer' }}>
              <Icon name="arrow-down" size="tiny" />
            </div>
          </ProjectActionDropdown>
        )}
        id={project.id}
        project={project}
        type="project"
      />
      <Tabs
        activeKey={settings.tab}
        defaultActiveKey={settings.tab ?? sentenceToCamelCase(tabs[0].title)}
        tabBarExtraContent={activeTab.options}
        tabBarStyle={{ padding: 16, paddingBottom: 0, paddingRight: 0 }}
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
