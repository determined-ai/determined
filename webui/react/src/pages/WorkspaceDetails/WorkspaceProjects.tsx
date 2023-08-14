import { Space } from 'antd';
import _ from 'lodash';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import GridListRadioGroup, { GridListView } from 'components/GridListRadioGroup';
import Button from 'components/kit/Button';
import Card from 'components/kit/Card';
import { Column, Columns } from 'components/kit/Columns';
import Input from 'components/kit/Input';
import { useModal } from 'components/kit/Modal';
import Select, { Option } from 'components/kit/Select';
import Spinner from 'components/kit/Spinner';
import Toggle from 'components/kit/Toggle';
import Link from 'components/Link';
import Message, { MessageType } from 'components/Message';
import ProjectActionDropdown from 'components/ProjectActionDropdown';
import ProjectCard from 'components/ProjectCard';
import ProjectCreateModalComponent from 'components/ProjectCreateModal';
import InteractiveTable, {
  ColumnDef,
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
import usePermissions from 'hooks/usePermissions';
import usePrevious from 'hooks/usePrevious';
import { useSettings } from 'hooks/useSettings';
import { paths } from 'routes/utils';
import { getWorkspaceProjects, patchProject } from 'services/api';
import { V1GetWorkspaceProjectsRequestSortBy } from 'services/api-ts-sdk';
import userStore from 'stores/users';
import { Project, Workspace } from 'types';
import { ErrorLevel, ErrorType } from 'utils/error';
import handleError from 'utils/error';
import { Loadable } from 'utils/loadable';
import { useObservable } from 'utils/observable';
import { validateDetApiEnum } from 'utils/service';

import css from './WorkspaceProjects.module.scss';
import {
  configForWorkspace,
  DEFAULT_COLUMN_WIDTHS,
  ProjectColumnName,
  WhoseProjects,
  WorkspaceDetailsSettings,
} from './WorkspaceProjects.settings';

interface Props {
  id: number;
  pageRef: React.RefObject<HTMLElement>;
  workspace: Workspace;
}

const WorkspaceProjects: React.FC<Props> = ({ workspace, id, pageRef }) => {
  const loadableUsers = useObservable(userStore.getUsers());
  const users = Loadable.getOrElse([], useObservable(userStore.getUsers()));
  const currentUser = Loadable.getOrElse(undefined, useObservable(userStore.currentUser));
  const [projects, setProjects] = useState<Project[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [canceler] = useState(new AbortController());
  const { canCreateProject } = usePermissions();
  const ProjectCreateModal = useModal(ProjectCreateModalComponent);
  const config = useMemo(() => configForWorkspace(id), [id]);
  const { settings, updateSettings } = useSettings<WorkspaceDetailsSettings>(config);

  const fetchProjects = useCallback(async () => {
    if (!settings) return;

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
        if (_.isEqual(prev, response.projects)) return prev;
        return response.projects;
      });
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch projects.' });
    } finally {
      setIsLoading(false);
    }
  }, [canceler.signal, id, workspace, settings]);

  useEffect(() => {
    setIsLoading(true);
    fetchProjects().then(() => setIsLoading(false));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleProjectCreateClick = useCallback(() => {
    ProjectCreateModal.open();
  }, [ProjectCreateModal]);

  const handleViewSelect = useCallback(
    (value: unknown) => {
      setIsLoading(true);
      updateSettings({ whose: value as WhoseProjects | undefined });
    },
    [updateSettings],
  );

  const handleSortSelect = useCallback(
    (value: unknown) => {
      updateSettings({
        sortDesc:
          value === V1GetWorkspaceProjectsRequestSortBy.NAME ||
          value === V1GetWorkspaceProjectsRequestSortBy.LASTEXPERIMENTSTARTTIME
            ? false
            : true,
        sortKey: value as V1GetWorkspaceProjectsRequestSortBy | undefined,
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

  const prevWhose = usePrevious(settings.whose, undefined);
  useEffect(() => {
    if (settings.whose === prevWhose || !settings.whose || Loadable.isLoading(loadableUsers))
      return;

    switch (settings.whose) {
      case WhoseProjects.All:
        updateSettings({ user: undefined });
        break;
      case WhoseProjects.Mine:
        updateSettings({ user: currentUser ? [currentUser.id.toString()] : undefined });
        break;
      case WhoseProjects.Others:
        updateSettings({
          user: users.filter((u) => u.id !== currentUser?.id).map((u) => u.id.toString()),
        });
        break;
    }
  }, [currentUser, loadableUsers, prevWhose, settings.whose, updateSettings, users]);

  const saveProjectDescription = useCallback(async (newDescription: string, projectId: number) => {
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
        project={record}
        workspaceArchived={workspace?.archived}
        onComplete={fetchProjects}
      />
    );

    const descriptionRenderer = (value: string, record: Project) => (
      <Input
        className={css.descriptionRenderer}
        defaultValue={value}
        disabled={record.archived}
        placeholder={record.archived ? 'Archived' : 'Add description...'}
        title={record.archived ? 'Archived description' : 'Edit description'}
        onBlur={(e) => {
          const newDesc = e.currentTarget.value;
          saveProjectDescription(newDesc, record.id);
        }}
        onPressEnter={(e) => {
          // when enter is pressed,
          // input box gets blurred and then value will be saved in onBlur
          e.currentTarget.blur();
        }}
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
        render: (_, r) => userRenderer(users.find((u) => u.id === r.userId)),
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
  }, [fetchProjects, saveProjectDescription, workspace?.archived, users]);

  const switchShowArchived = useCallback(
    (showArchived: boolean) => {
      if (!settings) return;

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
    ({
      record,
      onVisibleChange,
      children,
    }: {
      children: React.ReactNode;
      onVisibleChange?: (visible: boolean) => void;
      record: Project;
    }) => (
      <ProjectActionDropdown
        isContextMenu
        project={record}
        workspaceArchived={workspace?.archived}
        onComplete={fetchProjects}
        onVisibleChange={onVisibleChange}>
        {children}
      </ProjectActionDropdown>
    ),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [workspace?.archived],
  );

  const projectsList = useMemo(() => {
    if (!settings) return <Spinner spinning />;

    switch (settings.view) {
      case GridListView.Grid:
        return (
          <Card.Group size="small">
            {projects.map((project) => (
              <ProjectCard
                fetchProjects={fetchProjects}
                key={project.id}
                project={project}
                workspaceArchived={workspace?.archived}
              />
            ))}
          </Card.Group>
        );
      case GridListView.List:
        return (
          <InteractiveTable<Project, WorkspaceDetailsSettings>
            columns={columns}
            containerRef={pageRef}
            ContextMenu={actionDropdown}
            dataSource={projects}
            loading={isLoading || Loadable.isLoading(loadableUsers)}
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
            updateSettings={updateSettings}
          />
        );
    }
  }, [
    actionDropdown,
    columns,
    fetchProjects,
    isLoading,
    loadableUsers,
    pageRef,
    projects,
    settings,
    total,
    updateSettings,
    workspace?.archived,
  ]);

  useEffect(() => {
    fetchProjects();
  }, [fetchProjects]);

  useEffect(() => {
    return () => canceler.abort();
  }, [canceler]);

  return (
    <>
      <Columns page>
        <Column>
          <Select value={settings.whose} width={160} onSelect={handleViewSelect}>
            <Option value={WhoseProjects.All}>All Projects</Option>
            <Option value={WhoseProjects.Mine}>My Projects</Option>
            <Option value={WhoseProjects.Others}>Others&apos; Projects</Option>
          </Select>
        </Column>
        <Column align="right">
          <Space wrap>
            {!workspace.archived && (
              <Toggle
                checked={settings.archived}
                label="Show Archived"
                onChange={switchShowArchived}
              />
            )}
            <Select value={settings.sortKey} width={170} onSelect={handleSortSelect}>
              <Option value={V1GetWorkspaceProjectsRequestSortBy.NAME}>Alphabetical</Option>
              <Option value={V1GetWorkspaceProjectsRequestSortBy.LASTEXPERIMENTSTARTTIME}>
                Last Updated
              </Option>
              <Option value={V1GetWorkspaceProjectsRequestSortBy.CREATIONTIME}>
                Newest to Oldest
              </Option>
            </Select>
            {settings && <GridListRadioGroup value={settings.view} onChange={handleViewChange} />}
            <div className={css.headerButton}>
              {!workspace.immutable &&
                !workspace.archived &&
                canCreateProject({ workspace: workspace }) && (
                  <Button onClick={handleProjectCreateClick}>New Project</Button>
                )}
            </div>
          </Space>
        </Column>
      </Columns>
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
      <ProjectCreateModal.Component workspaceId={workspace.id} />
    </>
  );
};

export default WorkspaceProjects;
