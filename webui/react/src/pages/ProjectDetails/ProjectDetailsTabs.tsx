import { InfoCircleOutlined } from '@ant-design/icons';
import { Space, Tabs } from 'antd';
import type { TabsProps } from 'antd';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import BreadcrumbBar from 'components/BreadcrumbBar';
import Tooltip from 'components/kit/Tooltip';
import PageHeader from 'components/PageHeader';
import ProjectActionDropdown from 'components/ProjectActionDropdown';
import { paths } from 'routes/utils';
import { getWorkspace } from 'services/api';
import Icon from 'shared/components/Icon/Icon';
import { routeToReactUrl } from 'shared/utils/routes';
import { sentenceToCamelCase } from 'shared/utils/string';
import { DetailedUser, Project, Workspace } from 'types';
import handleError from 'utils/error';

import css from './ProjectDetailsTabs.module.scss';

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

const ProjectDetailsTabs: React.FC<Props> = ({ project, tabs, fetchProject, curUser }: Props) => {
  const [workspace, setWorkspace] = useState<Workspace>();
  const [activeTab, setActiveTab] = useState<TabInfo>(tabs[0]);

  const fetchWorkspace = useCallback(async () => {
    try {
      const response = await getWorkspace({ id: project.workspaceId });
      setWorkspace(response);
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch workspace.' });
    }
  }, [project.workspaceId]);

  const handleTabSwitch = useCallback(
    (tabKey: string) => {
      setActiveTab(tabs.find((tab) => sentenceToCamelCase(tab.title) === tabKey) ?? tabs[0]);
    },
    [tabs],
  );

  const onProjectDelete = useCallback(() => {
    routeToReactUrl(paths.workspaceDetails(project.workspaceId));
  }, [project.workspaceId]);

  const tabItems: TabsProps['items'] = useMemo(() => {
    return tabs.map((tabInfo) => ({
      children: (
        <div className={css.tabPane}>
          <div className={css.base}>{tabInfo.body}</div>
        </div>
      ),
      key: sentenceToCamelCase(tabInfo.title),
      label: tabInfo.title,
    }));
  }, [tabs]);

  /**
   * prevents stable tab content, e.g. archived state
   */
  useEffect(() => {
    setActiveTab((curTab) => tabs.find((tab) => tab.title === curTab.title) ?? tabs[0]);
  }, [tabs]);

  useEffect(() => {
    fetchWorkspace();
  }, [fetchWorkspace]);

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
        extra={
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
              trigger={['click']}
              workspaceArchived={workspace?.archived}
              onComplete={fetchProject}
              onDelete={onProjectDelete}>
              <div style={{ cursor: 'pointer' }}>
                <Icon name="arrow-down" size="tiny" />
              </div>
            </ProjectActionDropdown>
          </Space>
        }
        id={project.id}
        project={project}
        type="project"
      />
      <Tabs
        activeKey={sentenceToCamelCase(activeTab.title)}
        defaultActiveKey={sentenceToCamelCase(tabs[0].title)}
        items={tabItems}
        tabBarExtraContent={activeTab.options}
        tabBarStyle={{ height: 50, paddingLeft: 16 }}
        onChange={handleTabSwitch}
      />
    </>
  );
};

export default ProjectDetailsTabs;
