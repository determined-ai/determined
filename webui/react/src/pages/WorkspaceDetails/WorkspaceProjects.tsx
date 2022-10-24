import { Select, Space } from 'antd';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Grid, { GridMode } from 'components/Grid';
import GridListRadioGroup, { GridListView } from 'components/GridListRadioGroup';
import InlineEditor from 'components/InlineEditor';
import Link from 'components/Link';
import SelectFilter from 'components/SelectFilter';
import InteractiveTable, {
  ColumnDef,
  InteractiveTableSettings,
  onRightClickableCell,
} from 'components/Table/InteractiveTable';
import {
  checkmarkRenderer,
  GenericRenderer,
  getFullPaginationConfig,
  relativeTimeRenderer,
  stateRenderer,
  userRenderer,
} from 'components/Table/Table';
import Toggle from 'components/Toggle';
import { useStore } from 'contexts/Store';
import usePermissions from 'hooks/usePermissions';
import useSettings, { UpdateSettings } from 'hooks/useSettings';
import { paths } from 'routes/utils';
import { getWorkspaceProjects, patchProject } from 'services/api';
import { V1GetWorkspaceProjectsRequestSortBy } from 'services/api-ts-sdk';
import Message, { MessageType } from 'shared/components/Message';
import Spinner from 'shared/components/Spinner';
import { isEqual } from 'shared/utils/data';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { validateDetApiEnum } from 'shared/utils/service';
import { ShirtSize } from 'themes';
import { Project, Workspace } from 'types';
import handleError from 'utils/error';

import ProjectActionDropdown from '../WorkspaceDetails/ProjectActionDropdown';
import ProjectCard from '../WorkspaceDetails/ProjectCard';

import css from './WorkspaceProjects.module.scss';
import settingsConfig, {
  DEFAULT_COLUMN_WIDTHS,
  ProjectColumnName,
  WhoseProjects,
  WorkspaceDetailsSettings,
} from './WorkspaceProjects.settings';

const { Option } = Select;

interface Props {
  id: number;
  pageRef: React.RefObject<HTMLElement>;
  workspace: Workspace;
}

const WorkspaceProjects: React.FC<Props> = ({ workspace, id, pageRef }) => {
  const {
    users,
    auth: { user },
  } = useStore();
  const [projects, setProjects] = useState<Project[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [canceler] = useState(new AbortController());
  const { canCreateProject } = usePermissions();

  const { settings, updateSettings } = useSettings<WorkspaceDetailsSettings>(settingsConfig);

  const fetchProjects = useCallback(async () => {
    try {
      const response = await getWorkspaceProjects(
        {
          archived: workspace?.archived ? undefined : settings.archived ? undefined : false,
          id,
          limit: settings.view === GridListView.Grid ? 0 : settings.tableLimit,
          name: settings.name,
          offset: settings.view === GridListView.Grid ? 0 : settings.tableOffset,
          orderBy: settings.sortDesc ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC',
          sortBy: validateDetApiEnum(V1GetWorkspaceProjectsRequestSortBy, settings.sortKey),
          users: settings.user,
        },
        { signal: canceler.signal },
      );
      setTotal(response.pagination.total ?? 0);
      setProjects((prev) => {
        if (isEqual(prev, response.projects)) return prev;
        return response.projects;
      });
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch projects.' });
    } finally {
      setIsLoading(false);
    }
  }, [
    canceler.signal,
    id,
    workspace,
    settings.archived,
    settings.name,
    settings.sortDesc,
    settings.sortKey,
    settings.tableLimit,
    settings.tableOffset,
    settings.user,
    settings.view,
  ]);

  useEffect(() => {
    fetchProjects();
  }, [fetchProjects]);

  const handleViewSelect = useCallback(
    (value) => {
      updateSettings({ whose: value });
    },
    [updateSettings],
  );

  const handleSortSelect = useCallback(
    (value) => {
      updateSettings({
        sortDesc:
          value === V1GetWorkspaceProjectsRequestSortBy.NAME ||
          value === V1GetWorkspaceProjectsRequestSortBy.LASTEXPERIMENTSTARTTIME
            ? false
            : true,
        sortKey: value,
      });
    },
    [updateSettings],
  );

  const handleViewChange = useCallback(
    (value: GridListView) => {
      updateSettings({ view: value });
    },
    [updateSettings],
  );

  useEffect(() => {
    switch (settings.whose) {
      case WhoseProjects.All:
        updateSettings({ user: undefined });
        break;
      case WhoseProjects.Mine:
        updateSettings({ user: user ? [user.username] : undefined });
        break;
      case WhoseProjects.Others:
        updateSettings({ user: users.filter((u) => u.id !== user?.id).map((u) => u.username) });
        break;
    }
  }, [settings.whose, updateSettings, user, users]);

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

    const descriptionRenderer = (value: string, record: Project) => (
      <InlineEditor
        disabled={record.archived}
        placeholder={record.archived ? 'Archived' : 'Add description...'}
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
          record.lastExperimentStartedAt
            ? relativeTimeRenderer(new Date(record.lastExperimentStartedAt))
            : null,
        title: 'Last Experiment Started',
      },
      {
        dataIndex: 'userId',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['userId'],
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
        dataIndex: 'state',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['state'],
        key: 'state',
        render: stateRenderer,
        title: 'State',
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
  }, [fetchProjects, saveProjectDescription, user, workspace?.archived]);

  const switchShowArchived = useCallback(
    (showArchived: boolean) => {
      let newColumns: ProjectColumnName[];
      let newColumnWidths: number[];

      if (showArchived) {
        if (settings.columns?.includes('archived')) {
          // just some defensive coding: don't add archived twice
          newColumns = settings.columns;
          newColumnWidths = settings.columnWidths;
        } else {
          newColumns = [...settings.columns, 'archived'];
          newColumnWidths = [...settings.columnWidths, DEFAULT_COLUMN_WIDTHS['archived']];
        }
      } else {
        const archivedIndex = settings.columns.indexOf('archived');
        if (archivedIndex !== -1) {
          newColumns = [...settings.columns];
          newColumnWidths = [...settings.columnWidths];
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
    },
    [settings, updateSettings],
  );

  const actionDropdown = useCallback(
    ({ record, onVisibleChange, children }) => (
      <ProjectActionDropdown
        curUser={user}
        project={record}
        trigger={['contextMenu']}
        workspaceArchived={workspace?.archived}
        onComplete={fetchProjects}
        onVisibleChange={onVisibleChange}>
        {children}
      </ProjectActionDropdown>
    ),
    [fetchProjects, user, workspace?.archived],
  );

  const projectsList = useMemo(() => {
    switch (settings.view) {
      case GridListView.Grid:
        return (
          <Grid gap={ShirtSize.Medium} minItemWidth={250} mode={GridMode.AutoFill}>
            {projects.map((project) => (
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
            pagination={getFullPaginationConfig(
              {
                limit: settings.tableLimit,
                offset: settings.tableOffset,
              },
              total,
            )}
            rowKey="id"
            settings={settings}
            size="small"
            updateSettings={updateSettings as UpdateSettings<InteractiveTableSettings>}
          />
        );
    }
  }, [
    actionDropdown,
    columns,
    fetchProjects,
    isLoading,
    pageRef,
    projects,
    settings,
    total,
    updateSettings,
    user,
    workspace?.archived,
  ]);

  useEffect(() => {
    fetchProjects();
  }, [fetchProjects]);

  useEffect(() => {
    return () => canceler.abort();
  }, [canceler]);

  return (
    <div className={css.base}>
      <div className={css.controls}>
        <SelectFilter
          dropdownMatchSelectWidth={140}
          showSearch={false}
          value={settings.whose}
          onSelect={handleViewSelect}>
          <Option value={WhoseProjects.All}>All Projects</Option>
          <Option value={WhoseProjects.Mine}>My Projects</Option>
          <Option value={WhoseProjects.Others}>Others&apos; Projects</Option>
        </SelectFilter>
        <Space wrap>
          {!workspace.archived && (
            <Toggle
              checked={settings.archived}
              prefixLabel="Show Archived"
              onChange={switchShowArchived}
            />
          )}
          <SelectFilter
            dropdownMatchSelectWidth={150}
            showSearch={false}
            value={settings.sortKey}
            onSelect={handleSortSelect}>
            <Option value={V1GetWorkspaceProjectsRequestSortBy.NAME}>Alphabetical</Option>
            <Option value={V1GetWorkspaceProjectsRequestSortBy.LASTEXPERIMENTSTARTTIME}>
              Last Updated
            </Option>
            <Option value={V1GetWorkspaceProjectsRequestSortBy.CREATIONTIME}>
              Newest to Oldest
            </Option>
          </SelectFilter>
          <GridListRadioGroup value={settings.view} onChange={handleViewChange} />
        </Space>
      </div>
      <Spinner spinning={isLoading}>
        {projects.length !== 0 ? (
          projectsList
        ) : workspace.numProjects === 0 ? (
          <Message
            message={
              canCreateProject({ workspace: { id } })
                ? 'Create a project with the "New Project" button or in the CLI.'
                : 'User cannot create a project in this workspace.'
            }
            title="Workspace contains no projects. "
            type={MessageType.Empty}
          />
        ) : (
          <Message title="No projects matching the current filters" type={MessageType.Empty} />
        )}
      </Spinner>
    </div>
  );
};

export default WorkspaceProjects;
