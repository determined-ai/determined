import { Select } from 'antd';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useParams } from 'react-router';

import Grid, { GridMode } from 'components/Grid';
import GridListRadioGroup, { GridListView } from 'components/GridListRadioGroup';
import InteractiveTable, { ColumnDef } from 'components/InteractiveTable';
import Link from 'components/Link';
import Message, { MessageType } from 'components/Message';
import Page from 'components/Page';
import SelectFilter from 'components/SelectFilter';
import Spinner from 'components/Spinner';
import { GenericRenderer, getFullPaginationConfig,
  relativeTimeRenderer, userRenderer } from 'components/Table';
import { useStore } from 'contexts/Store';
import { useFetchUsers } from 'hooks/useFetch';
import usePolling from 'hooks/usePolling';
import useSettings from 'hooks/useSettings';
import { paths } from 'routes/utils';
import { getWorkspace, getWorkspaceProjects, isNotFound } from 'services/api';
import { V1GetWorkspaceProjectsRequestSortBy } from 'services/api-ts-sdk';
import { validateDetApiEnum } from 'services/utils';
import { ShirtSize } from 'themes';
import { Project, Workspace } from 'types';
import handleError from 'utils/error';

import css from './WorkspaceDetails.module.scss';
import settingsConfig,
{ DEFAULT_COLUMN_WIDTHS, WorkspaceDetailsSettings } from './WorkspaceDetails.settings';
import ProjectActionDropdown from './WorkspaceDetails/ProjectActionDropdown';
import ProjectCard from './WorkspaceDetails/ProjectCard';
import WorkspaceDetailsHeader from './WorkspaceDetails/WorkspaceDetailsHeader';

const { Option } = Select;

interface Params {
  workspaceId: string;
}

enum ProjectFilters {
  All = 'ALL_PROJECTS',
  Mine = 'MY_PROJECTS',
  Others = 'OTHERS_PROJECTS'
}

/*
 * This indicates that the cell contents are rightClickable
 * and we should disable custom context menu on cell context hover
 */
const onRightClickableCell = () =>
  ({ isCellRightClickable: true } as React.HTMLAttributes<HTMLElement>);

const WorkspaceDetails: React.FC = () => {
  const { users, auth: { user } } = useStore();
  const { workspaceId } = useParams<Params>();
  const [ workspace, setWorkspace ] = useState<Workspace>();
  const [ projects, setProjects ] = useState<Project[]>([]);
  const [ projectFilter, setProjectFilter ] = useState<ProjectFilters>(ProjectFilters.All);
  const [ selectedView, setSelectedView ] = useState(GridListView.Grid);
  const [ pageError, setPageError ] = useState<Error>();
  const [ isLoading, setIsLoading ] = useState(true);
  const [ total, setTotal ] = useState(0);
  const [ canceler ] = useState(new AbortController());
  const pageRef = useRef<HTMLElement>(null);

  const id = parseInt(workspaceId);

  const {
    settings,
    updateSettings,
  } = useSettings<WorkspaceDetailsSettings>(settingsConfig);

  const fetchWorkspace = useCallback(async () => {
    try {
      const response = await getWorkspace({ id }, { signal: canceler.signal });
      setWorkspace(response);
    } catch (e) {
      if (!pageError) setPageError(e as Error);
    }
  }, [ canceler.signal, id, pageError ]);

  const fetchProjects = useCallback(async () => {
    try {
      const response = await getWorkspaceProjects({
        archived: settings.archived,
        id,
        limit: settings.tableLimit,
        name: settings.name,
        offset: settings.tableOffset,
        orderBy: settings.sortDesc ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC',
        sortBy: validateDetApiEnum(V1GetWorkspaceProjectsRequestSortBy, settings.sortKey),
        users: settings.user,
      }, { signal: canceler.signal });
      setTotal(response.pagination.total ?? 0);
      setProjects(response.projects);
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch projects.' });
    } finally {
      setIsLoading(false);
    }
  }, [ canceler.signal,
    id,
    settings.archived,
    settings.name,
    settings.sortDesc,
    settings.sortKey,
    settings.tableLimit,
    settings.tableOffset,
    settings.user ]);

  const fetchUsers = useFetchUsers(canceler);

  const fetchAll = useCallback(async () => {
    await Promise.allSettled([ fetchWorkspace(), fetchProjects(), fetchUsers() ]);
  }, [ fetchWorkspace, fetchProjects, fetchUsers ]);

  usePolling(fetchAll);

  const handleViewSelect = useCallback((value) => {
    setProjectFilter(value as ProjectFilters);
  }, []);

  const handleSortSelect = useCallback((value) => {
    updateSettings({ sortKey: value });
  }, [ updateSettings ]);

  const handleViewChange = useCallback((value: GridListView) => {
    setSelectedView(value);
  }, []);

  useEffect(() => {
    switch (projectFilter) {
      case ProjectFilters.All:
        updateSettings({ user: undefined });
        break;
      case ProjectFilters.Mine:
        updateSettings({ user: user ? [ user.username ] : undefined });
        break;
      case ProjectFilters.Others:
        updateSettings({ user: users.filter(u => u.id !== user?.id).map(u => u.username) });
        break;
    }
  }, [ projectFilter, updateSettings, user, users ]);

  const columns = useMemo(() => {
    const projectNameRenderer = (value: string, record: Project) => (
      <Link path={paths.projectDetails(record.id)}>{value}</Link>
    );

    const actionRenderer: GenericRenderer<Project> = (_, record) => (
      <ProjectActionDropdown
        curUser={user}
        project={record}
      />
    );

    return [
      {
        dataIndex: 'name',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['name'],
        key: V1GetWorkspaceProjectsRequestSortBy.NAME,
        onCell: onRightClickableCell,
        render: projectNameRenderer,
        title: 'Name',
      },
      {
        dataIndex: 'description',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['description'],
        key: V1GetWorkspaceProjectsRequestSortBy.DESCRIPTION,
        onCell: onRightClickableCell,
        title: 'Description',
      },
      {
        dataIndex: 'numExperiments',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['numExperiments'],
        title: 'Experiments',
      },
      {
        dataIndex: 'lastUpdated',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['lastUpdated'],
        render: (_: number, record: Project): React.ReactNode =>
          record.lastExperimentStartedAt ?
            relativeTimeRenderer(new Date(record.lastExperimentStartedAt)) :
            null,
        title: 'Last Updated',
      },
      {
        dataIndex: 'user',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['user'],
        render: userRenderer,
        title: 'User',
      },
      {
        dataIndex: 'archived',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['archived'],
        title: 'Archived',
      },
      {
        align: 'right',
        dataIndex: 'action',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['action'],
        fixed: 'right',
        key: 'action',
        onCell: onRightClickableCell,
        render: actionRenderer,
        title: '',
      },
    ] as ColumnDef<Project>[];
  }, [ user ]);

  const actionDropdown = useCallback(
    ({ record, onVisibleChange, children }) => (
      <ProjectActionDropdown
        curUser={user}
        project={record}
        onVisibleChange={onVisibleChange}>
        {children}
      </ProjectActionDropdown>
    ),
    [ user ],
  );

  const projectsList = useMemo(() => {
    switch (selectedView) {
      case GridListView.Grid:
        return (
          <Grid gap={ShirtSize.medium} mode={GridMode.AutoFill}>
            {projects.map(project => (
              <ProjectCard curUser={user} key={project.id} project={project} />
            ))}
          </Grid>
        );
      case GridListView.List:
        return (
          <InteractiveTable
            columns={columns}
            containerRef={pageRef}
            ContextMenu={actionDropdown}
            dataSource={projects}
            loading={isLoading}
            pagination={getFullPaginationConfig({
              limit: settings.tableLimit,
              offset: settings.tableOffset,
            }, total)}
            settings={settings}
            size="small"
            updateSettings={updateSettings}
          />
        );
    }
  }, [ actionDropdown,
    columns,
    isLoading,
    projects,
    selectedView,
    settings,
    total,
    updateSettings,
    user ]);

  useEffect(() => {
    fetchAll();
  }, [ fetchAll ]);

  useEffect(() => {
    return () => canceler.abort();
  }, [ canceler ]);

  if (isNaN(id)) {
    return <Message title={`Invalid Workspace ID ${workspaceId}`} />;
  } else if (pageError) {
    const message = isNotFound(pageError) ?
      `Unable to find Workspace ${workspaceId}` :
      `Unable to fetch Workspace ${workspaceId}`;
    return <Message title={message} type={MessageType.Warning} />;
  } else if (!workspace) {
    return <Spinner tip={`Loading workspace ${workspaceId} details...`} />;
  }

  return (
    <Page
      className={css.base}
      containerRef={pageRef}
      headerComponent={<WorkspaceDetailsHeader workspace={workspace} />}
      id="workspaceDetails">
      <div className={css.controls}>
        <SelectFilter
          bordered={false}
          label="View:"
          value={projectFilter}
          onSelect={handleViewSelect}>
          <Option value={ProjectFilters.All}>All projects</Option>
          <Option value={ProjectFilters.Mine}>My projects</Option>
          <Option value={ProjectFilters.Others}>Others projects</Option>
        </SelectFilter>
        <div className={css.options}>
          <SelectFilter
            bordered={false}
            label="Sort:"
            value={settings.sortKey}
            onSelect={handleSortSelect}>
            <Option value={V1GetWorkspaceProjectsRequestSortBy.NAME}>Alphabetical</Option>
            <Option value={V1GetWorkspaceProjectsRequestSortBy.LASTEXPERIMENTSTARTTIME}>
              Last updated
            </Option>
            <Option value={V1GetWorkspaceProjectsRequestSortBy.CREATIONTIME}>
              Newest to oldest
            </Option>
          </SelectFilter>
          <GridListRadioGroup value={selectedView} onChange={handleViewChange} />
        </div>
      </div>
      <Spinner spinning={isLoading}>
        {projects.length !== 0 ? (
          projectsList
        ) : (
          <Message
            title="No projects matching the current filters"
            type={MessageType.Empty}
          />
        )}
      </Spinner>
    </Page>
  );
};

export default WorkspaceDetails;
