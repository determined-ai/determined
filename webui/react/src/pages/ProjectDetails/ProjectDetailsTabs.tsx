import { Tabs } from 'antd';
import React, { useCallback, useEffect, useState } from 'react';

import BreadcrumbBar from 'components/BreadcrumbBar';
import PageHeader from 'components/PageHeader';
import { UpdateSettings } from 'hooks/useSettings';
import { ProjectDetailsSettings } from 'pages/ProjectDetails.settings';
import { Project } from 'types';
import { sentenceToCamelCase } from 'utils/string';

import css from './ProjectDetailsTabs.module.scss';

const { TabPane } = Tabs;

export interface TabInfo {
  body: React.ReactNode;
  options?: React.ReactNode;
  title: string;
}

interface Props {
  project: Project;
  settings: ProjectDetailsSettings;
  tabs: TabInfo[];
  updateSettings: UpdateSettings<ProjectDetailsSettings>
}

const ProjectDetailsTabs: React.FC<Props> = (
  { project, tabs, settings, updateSettings }: Props,
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
          className={css.noPadding}
          options={experimentsTab?.options}
          title="Uncategorized"
        />
        <div className={css.body}>
          {experimentsTab?.body}
        </div>
      </div>
    );
  }

  return (
    <div>
      <BreadcrumbBar id={project.id} project={project} type="project" />
      <Tabs
        activeKey={settings.tab}
        defaultActiveKey={settings.tab ?? sentenceToCamelCase(tabs[0].title)}
        tabBarExtraContent={activeTab.options}
        tabBarStyle={{ padding: 16, paddingBottom: 0 }}
        onChange={handleTabSwitch}>
        {tabs.map(tabInfo => {
          return (
            <TabPane key={sentenceToCamelCase(tabInfo.title)} tab={tabInfo.title}>
              <div className={css.base}>
                {tabInfo.body}
              </div>
            </TabPane>
          );
        })}
      </Tabs>
    </div>
  );
};

export default ProjectDetailsTabs;
