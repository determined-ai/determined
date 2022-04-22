import { Breadcrumb, Tabs } from 'antd';
import React from 'react';

import Link from 'components/Link';
import PageHeader from 'components/PageHeader';
import WorkspaceIcon from 'components/WorkspaceIcon';
import { paths } from 'routes/utils';
import { Project, Workspace } from 'types';

import css from './ProjectDetailsHeader.module.scss';

const { TabPane } = Tabs;

interface Props {
  experimentsTab: React.ReactNode;
  notesTab: React.ReactNode;
  options?: React.ReactNode;
  project: Project;
  workspace?: Workspace;
}

const ProjectDetailsHeader: React.FC<Props> = (
  { workspace, project, options, experimentsTab, notesTab }: Props,
) => {
  if (project.immutable) {
    return (
      <div className={css.base}>
        <PageHeader
          className={css.noPadding}
          options={options}
          title="Uncategorized"
        />
        <div className={css.body}>
          {experimentsTab}
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
      <Tabs defaultActiveKey="experiments" tabBarStyle={{ padding: 16, paddingBottom: 0 }}>
        <TabPane key="experiments" tab="Experiments">
          <div className={css.base}>
            <PageHeader
              className={css.noPadding}
              options={options}
              title="Experiments"
            />
            <div className={css.body}>
              {experimentsTab}
            </div>
          </div>
        </TabPane>
        <TabPane key="notes" tab="Notes">
          <PageHeader
            className={css.noPadding}
            title="Notes"
          />
          {notesTab}
        </TabPane>
      </Tabs>
    </div>
  );
};

export default ProjectDetailsHeader;
