import Button from 'hew/Button';
import Card from 'hew/Card';
import Column from 'hew/Column';
import Input from 'hew/Input';
import Message from 'hew/Message';
import { useModal } from 'hew/Modal';
import Row from 'hew/Row';
import Section from 'hew/Section';
import Select, { Option } from 'hew/Select';
import Spinner from 'hew/Spinner';
import Toggle from 'hew/Toggle';
import { Loadable } from 'hew/utils/loadable';
import { List } from 'immutable';
import { sortBy } from 'lodash';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import GridListRadioGroup, { GridListView } from 'components/GridListRadioGroup';
import Link from 'components/Link';
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
import useFeature from 'hooks/useFeature';
import usePermissions from 'hooks/usePermissions';
import usePrevious from 'hooks/usePrevious';
import { useSettings } from 'hooks/useSettings';
import { paths } from 'routes/utils';
import { patchProject } from 'services/api';
import { V1GetWorkspaceProjectsRequestSortBy } from 'services/api-ts-sdk';
import projectStore from 'stores/projects';
import userStore from 'stores/users';
import { Project, Workspace } from 'types';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';
import { useObservable } from 'utils/observable';

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
  const [canceler] = useState(new AbortController());
  const { canCreateProject } = usePermissions();
  const ProjectCreateModal = useModal(ProjectCreateModalComponent);
  const config = useMemo(() => configForWorkspace(id), [id]);
  const { settings, updateSettings } = useSettings<WorkspaceDetailsSettings>(config);
  const streamingUpdatesOn = useFeature().isOn('streaming_updates');
  const f_flat_runs = useFeature().isOn('flat_runs');

  const loadableProjects: Loadable<List<Project>> = useObservable(
    projectStore.getProjectsByWorkspace(id),
  );

  const sortProjects = useCallback(
    (arr: Project[]) => {
      switch (settings.sortKey) {
        case V1GetWorkspaceProjectsRequestSortBy.LASTEXPERIMENTSTARTTIME:
          return arr.sort((a, b) => {
            if (!a.lastExperimentStartedAt && !b.lastExperimentStartedAt) return b.id - a.id;
            if (a.lastExperimentStartedAt && b.lastExperimentStartedAt)
              return new Date(a.lastExperimentStartedAt) < new Date(b.lastExperimentStartedAt)
                ? 1
                : -1;
            return a.lastExperimentStartedAt ? -1 : 1;
          });
        case V1GetWorkspaceProjectsRequestSortBy.NAME:
          return sortBy(arr, 'name');
        case V1GetWorkspaceProjectsRequestSortBy.CREATIONTIME:
          return sortBy(arr, 'id').reverse();
        default:
          return arr;
      }
    },
    [settings.sortKey],
  );

  const [projects, isLoading] = useMemo(
    () =>
      loadableProjects
        .map((p): [Project[], boolean] => [
          sortProjects(p.toJSON().filter((p) => (settings.archived ? p : !p.archived))),
          false,
        ])
        .getOrElse([[], true]),
    [loadableProjects, settings.archived, sortProjects],
  );

  const handleProjectCreateClick = useCallback(() => {
    ProjectCreateModal.open();
  }, [ProjectCreateModal]);

  const handleViewSelect = useCallback(
    (value: unknown) => {
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
    if (settings.whose === prevWhose || !settings.whose || Loadable.isNotLoaded(loadableUsers))
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

  const onEdit = useCallback(
    (projectId: number, name: string, archived: boolean, description?: string) => {
      if (!streamingUpdatesOn) {
        const project = projects.find((p) => p.id === projectId);
        project &&
          projectStore.upsertProject({
            ...project,
            archived,
            description: description || project?.description,
            name,
          } as Project);
      }
    },
    [streamingUpdatesOn, projects],
  );

  const onRemove = useCallback(
    (projectId: number) => {
      if (!streamingUpdatesOn) {
        projectStore.delete(projectId);
      }
    },
    [streamingUpdatesOn],
  );

  const saveProjectDescription = useCallback(
    async (newDescription: string, projectId: number) => {
      try {
        await patchProject({ description: newDescription, id: projectId });
        const project = projects.find((p) => p.id === projectId);
        project && projectStore.upsertProject({ ...project, description: newDescription });
      } catch (e) {
        handleError(e, {
          level: ErrorLevel.Error,
          publicMessage: 'Please try again later.',
          publicSubject: 'Unable to edit project.',
          silent: false,
          type: ErrorType.Server,
        });
      }
    },
    [projects],
  );

  const columns = useMemo(() => {
    const projectNameRenderer = (value: string, record: Project) => (
      <Link path={paths.projectDetails(record.id)}>{value}</Link>
    );

    const actionRenderer: GenericRenderer<Project> = (_, record) => (
      <ProjectActionDropdown
        project={record}
        workspaceArchived={workspace?.archived}
        onDelete={() => onRemove(record.id)}
        onEdit={(name: string, archived: boolean, description?: string) =>
          onEdit(record.id, name, archived, description)
        }
        onMove={() => onRemove(record.id)}
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
        title: `Last ${f_flat_runs ? 'Run' : 'Experiment'} Started`,
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
  }, [f_flat_runs, workspace?.archived, onRemove, onEdit, saveProjectDescription, users]);

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
    ({ record, children }: { children: React.ReactNode; record: Project }) => (
      <ProjectActionDropdown
        isContextMenu
        project={record}
        workspaceArchived={workspace?.archived}
        onDelete={() => onRemove(record.id)}
        onEdit={(name: string, archived: boolean, description?: string) =>
          onEdit(record.id, name, archived, description)
        }
        onMove={() => onRemove(record.id)}>
        {children}
      </ProjectActionDropdown>
    ),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [workspace?.archived, onEdit, onRemove],
  );

  const projectsList = useMemo(() => {
    if (!settings) return <Spinner spinning />;

    switch (settings.view) {
      case GridListView.Grid:
        return (
          <Card.Group size="small">
            {projects.map((project) => (
              <ProjectCard
                key={project.id}
                project={project}
                workspaceArchived={workspace?.archived}
                onEdit={(name: string, archived: boolean, description?: string) =>
                  onEdit(project.id, name, archived, description)
                }
                onRemove={() => onRemove(project.id)}
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
            loading={isLoading || Loadable.isNotLoaded(loadableUsers)}
            pagination={getFullPaginationConfig(
              {
                limit: settings.tableLimit,
                offset: settings.tableOffset,
              },
              projects.length,
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
    isLoading,
    loadableUsers,
    pageRef,
    projects,
    settings,
    updateSettings,
    workspace?.archived,
    onEdit,
    onRemove,
  ]);

  useEffect(() => {
    projectStore.fetch(id, canceler.signal, true);
  }, [id, canceler]);

  useEffect(() => {
    return () => canceler.abort();
  }, [canceler]);

  return (
    <>
      <Section>
        <Row wrap>
          <Column>
            <Select value={settings.whose} width={160} onSelect={handleViewSelect}>
              <Option value={WhoseProjects.All}>All Projects</Option>
              <Option value={WhoseProjects.Mine}>My Projects</Option>
              <Option value={WhoseProjects.Others}>Others&apos; Projects</Option>
            </Select>
          </Column>
          <Column align="right">
            <Row wrap>
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
                    <Button data-testid="newProject" onClick={handleProjectCreateClick}>
                      New Project
                    </Button>
                  )}
              </div>
            </Row>
          </Column>
        </Row>
      </Section>
      <Spinner conditionalRender spinning={isLoading}>
        {projects.length !== 0 ? (
          projectsList
        ) : workspace.numProjects === 0 ? (
          <Message
            description={
              canCreateProject({ workspace: { id } })
                ? 'Create a project with the "New Project" button or in the CLI.'
                : 'User cannot create a project in this workspace.'
            }
            icon="warning"
            title="Workspace contains no projects. "
          />
        ) : (
          <Message icon="warning" title="No projects matching the current filters" />
        )}
      </Spinner>
      <ProjectCreateModal.Component workspaceId={workspace.id} />
    </>
  );
};

export default WorkspaceProjects;
