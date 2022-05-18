import { Select, Space, Switch } from 'antd';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useParams } from 'react-router';

import Grid, { GridMode } from 'components/Grid';
import GridListRadioGroup, { GridListView } from 'components/GridListRadioGroup';
import InlineEditor from 'components/InlineEditor';
import InteractiveTable, { ColumnDef, InteractiveTableSettings } from 'components/InteractiveTable';
import Label, { LabelTypes } from 'components/Label';
import Link from 'components/Link';
import Page from 'components/Page';
import SelectFilter from 'components/SelectFilter';
import Spinner from 'components/Spinner';
import { checkmarkRenderer, GenericRenderer, getFullPaginationConfig,
  relativeTimeRenderer, userRenderer } from 'components/Table';
import { useStore } from 'contexts/Store';
import { useFetchUsers } from 'hooks/useFetch';
import usePolling from 'hooks/usePolling';
import useResize from 'hooks/useResize';
import useSettings, { UpdateSettings } from 'hooks/useSettings';
import { paths } from 'routes/utils';
import { getWorkspace, getWorkspaceProjects, isNotFound, patchProject } from 'services/api';
import { V1GetWorkspaceProjectsRequestSortBy } from 'services/api-ts-sdk';
import { validateDetApiEnum } from 'services/utils';
import Message, { MessageType } from 'shared/components/message';
import { ShirtSize } from 'themes';
import { Project, Workspace } from 'types';
import { isEqual } from 'utils/data';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';

import css from './WorkspaceDetails.module.scss';
import settingsConfig, { DEFAULT_COLUMN_WIDTHS,
  ProjectColumnName, WorkspaceDetailsSettings } from './WorkspaceDetails.settings';
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
  const [ pageError, setPageError ] = useState<Error>();
  const [ isLoading, setIsLoading ] = useState(true);
  const [ total, setTotal ] = useState(0);
  const [ canceler ] = useState(new AbortController());
  const pageRef = useRef<HTMLElement>(null);
  const size = useResize();

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
        archived: workspace?.archived ? undefined : settings.archived ? undefined : false,
        id,
        limit: settings.tableLimit,
        name: settings.name,
        offset: settings.tableOffset,
        orderBy: settings.sortDesc ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC',
        sortBy: validateDetApiEnum(V1GetWorkspaceProjectsRequestSortBy, settings.sortKey),
        users: settings.user,
      }, { signal: canceler.signal });
      setTotal(response.pagination.total ?? 0);
      setProjects(prev => {
        if (isEqual(prev, response.projects)) return prev;
        return response.projects;
      });
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
    settings.user,
    workspace?.archived ]);

  const fetchUsers = useFetchUsers(canceler);

  const fetchAll = useCallback(async () => {
    await Promise.allSettled([ fetchWorkspace(), fetchProjects(), fetchUsers() ]);
  }, [ fetchWorkspace, fetchProjects, fetchUsers ]);

  usePolling(fetchAll);

  const handleViewSelect = useCallback((value) => {
    setProjectFilter(value as ProjectFilters);
  }, []);

  const handleSortSelect = useCallback((value) => {
    updateSettings({
      sortDesc: (value === V1GetWorkspaceProjectsRequestSortBy.NAME ||
        value === V1GetWorkspaceProjectsRequestSortBy.LASTEXPERIMENTSTARTTIME) ? false : true,
      sortKey: value,
    });
  }, [ updateSettings ]);

  const handleViewChange = useCallback((value: GridListView) => {
    updateSettings({ view: value });
  }, [ updateSettings ]);

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

  const saveProjectDescription = useCallback(async (newDescription, projectId: number) => {
    try {
      await patchProject({ description: newDescription, id: projectId });
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to edit project.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, []);

  const columns = useMemo(() => {
    const projectNameRenderer = (value: string, record: Project) => (
      <Link path={paths.projectDetails(record.id)}>{value}</Link>
    );

    const actionRenderer: GenericRenderer<Project> = (_, record) => (
      <ProjectActionDropdown
        curUser={user}
        project={record}
        workspaceArchived={workspace?.archived}
        onComplete={fetchProjects}
      />
    );

    const descriptionRenderer = (value:string, record: Project) => (
      <InlineEditor
        disabled={record.archived}
        placeholder="Add description..."
        value={value}
        onSave={(newDescription: string) => saveProjectDescription(newDescription, record.id)}
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
        render: descriptionRenderer,
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
        title: 'Last Experiment Started',
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
        key: 'archived',
        render: checkmarkRenderer,
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
  }, [ fetchProjects, saveProjectDescription, user, workspace?.archived ]);

  const switchShowArchived = useCallback((showArchived: boolean) => {
    let newColumns: ProjectColumnName[];
    let newColumnWidths: number[];

    if (showArchived) {
      if (settings.columns?.includes('archived')) {
        // just some defensive coding: don't add archived twice
        newColumns = settings.columns;
        newColumnWidths = settings.columnWidths;
      } else {
        newColumns = [ ...settings.columns, 'archived' ];
        newColumnWidths = [ ...settings.columnWidths, DEFAULT_COLUMN_WIDTHS['archived'] ];
      }
    } else {
      const archivedIndex = settings.columns.indexOf('archived');
      if (archivedIndex !== -1) {
        newColumns = [ ...settings.columns ];
        newColumnWidths = [ ...settings.columnWidths ];
        newColumns.splice(archivedIndex, 1);
        newColumnWidths.splice(archivedIndex, 1);
      } else {
        newColumns = settings.columns;
        newColumnWidths = settings.columnWidths;
      }
    }
    updateSettings({
      archived: showArchived,
      columns: newColumns,
      columnWidths: newColumnWidths,
    });

  }, [ settings, updateSettings ]);

  const actionDropdown = useCallback(
    ({ record, onVisibleChange, children }) => (
      <ProjectActionDropdown
        curUser={user}
        project={record}
        trigger={[ 'contextMenu' ]}
        workspaceArchived={workspace?.archived}
        onComplete={fetchProjects}
        onVisibleChange={onVisibleChange}>
        {children}
      </ProjectActionDropdown>
    ),
    [ fetchProjects, user, workspace?.archived ],
  );

  const projectsList = useMemo(() => {
    switch (settings.view) {
      case GridListView.Grid:
        return (
          <Grid
            gap={ShirtSize.medium}
            minItemWidth={size.width <= 480 ? 165 : 300}
            mode={GridMode.AutoFill}>
            {projects.map(project => (
              <ProjectCard
                curUser={user}
                fetchProjects={fetchProjects}
                key={project.id}
                project={project}
                workspaceArchived={workspace?.archived}
              />
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
            updateSettings={updateSettings as UpdateSettings<InteractiveTableSettings>}
          />
        );
    }
  }, [ actionDropdown,
    columns,
    fetchProjects,
    isLoading,
    projects,
    settings,
    size.width,
    total,
    updateSettings,
    user,
    workspace?.archived ]);

  useEffect(() => {
    fetchWorkspace();
  }, [ fetchWorkspace ]);

  useEffect(() => {
    fetchProjects();
  }, [ fetchProjects ]);

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
      headerComponent={(
        <WorkspaceDetailsHeader
          curUser={user}
          fetchWorkspace={fetchAll}
          workspace={workspace}
        />
      )}
      id="workspaceDetails">
      <div className={css.controls}>
        <SelectFilter
          bordered={false}
          dropdownMatchSelectWidth={140}
          label="View:"
          showSearch={false}
          value={projectFilter}
          onSelect={handleViewSelect}>
          <Option value={ProjectFilters.All}>All projects</Option>
          <Option value={ProjectFilters.Mine}>My projects</Option>
          <Option value={ProjectFilters.Others}>Others&apos; projects</Option>
        </SelectFilter>
        <Space wrap>
          {!workspace.archived && (
            <>
              <Switch checked={settings.archived} onChange={switchShowArchived} />
              <Label type={LabelTypes.TextOnly}>Show Archived</Label>
            </>
          )}
          <SelectFilter
            bordered={false}
            dropdownMatchSelectWidth={150}
            label="Sort:"
            showSearch={false}
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
          <GridListRadioGroup value={settings.view} onChange={handleViewChange} />
        </Space>
      </div>
      <Spinner spinning={isLoading}>
        {projects.length !== 0 ? (
          projectsList
        ) : (
          workspace.numProjects === 0 ? (
            <Message
              message='Create a project with the "New Project" button or in the CLI.'
              title="Workspace contains no projects. "
              type={MessageType.Empty}
            />
          ) : (
            <Message
              title="No projects matching the current filters"
              type={MessageType.Empty}
            />
          )
        )}
      </Spinner>
    </Page>
  );
};

export default WorkspaceDetails;
