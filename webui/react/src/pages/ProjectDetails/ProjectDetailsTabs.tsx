import { Breadcrumb, Tabs } from 'antd';
import React, { useCallback, useEffect, useState } from 'react';

import Link from 'components/Link';
import PageHeader from 'components/PageHeader';
import WorkspaceIcon from 'components/WorkspaceIcon';
import { paths } from 'routes/utils';
import { Project, Workspace } from 'types';
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
  tabs: TabInfo[];
  workspace?: Workspace;
}

const ProjectDetailsTabs: React.FC<Props> = (
  { workspace, project, tabs }: Props,
) => {
  const [ activeTab, setActiveTab ] = useState<TabInfo>(tabs[0]);

  const handleTabSwitch = useCallback((tabKey) => {
    setActiveTab(tabs.find(tab => sentenceToCamelCase(tab.title) === tabKey) ?? tabs[0]);
  }, [ tabs ]);

  useEffect(() => {
    handleTabSwitch(sentenceToCamelCase(activeTab.title));
  }, [ activeTab.title, handleTabSwitch ]);

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
      <div className={css.breadcrumbsRow}>
        <Breadcrumb separator="">
          <Breadcrumb.Item>
            <Link path={paths.workspaceDetails(project.workspaceId)}>
              <WorkspaceIcon name={workspace?.name} size={24} style={{ marginRight: 10 }} />
            </Link>
          </Breadcrumb.Item>
          <Breadcrumb.Item>
            <Link path={paths.workspaceDetails(project.workspaceId)}>
              {workspace?.name ?? '...'}
            </Link>
          </Breadcrumb.Item>
          <Breadcrumb.Separator />
          <Breadcrumb.Item>
            {project.name}
          </Breadcrumb.Item>
        </Breadcrumb>
      </div>
      <Tabs
        defaultActiveKey={sentenceToCamelCase(tabs[0].title)}
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
