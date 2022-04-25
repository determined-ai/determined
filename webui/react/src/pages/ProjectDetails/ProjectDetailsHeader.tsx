import { Breadcrumb, Tabs } from 'antd';
import React from 'react';

import Link from 'components/Link';
import PageHeader from 'components/PageHeader';
import WorkspaceIcon from 'components/WorkspaceIcon';
import { paths } from 'routes/utils';
import { Project, Workspace } from 'types';
import { sentenceToCamelCase } from 'utils/string';

import css from './ProjectDetailsHeader.module.scss';

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

const ProjectDetailsHeader: React.FC<Props> = (
  { workspace, project, tabs }: Props,
) => {
  if (project.immutable) {
    return (
      <div className={css.base}>
        <PageHeader
          className={css.noPadding}
          options={tabs[0].options}
          title="Uncategorized"
        />
        <div className={css.body}>
          {tabs[0].body}
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
        tabBarStyle={{ padding: 16, paddingBottom: 0 }}>
        {tabs.map(tabInfo => {
          return (
            <TabPane key={sentenceToCamelCase(tabInfo.title)} tab={tabInfo.title}>
              <div className={css.base}>
                <PageHeader
                  className={css.noPadding}
                  options={tabInfo.options}
                  title={tabInfo.title}
                />
                <div className={css.body}>
                  {tabInfo.body}
                </div>
              </div>
            </TabPane>
          );
        })}
      </Tabs>
    </div>
  );
};

export default ProjectDetailsHeader;
