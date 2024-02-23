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
import usePermissions from 'hooks/usePermissions';
import usePrevious from 'hooks/usePrevious';
import { useSettings } from 'hooks/useSettings';
import { paths } from 'routes/utils';
import { patchProject } from 'services/api';
import { V1GetWorkspaceProjectsRequestSortBy } from 'services/api-ts-sdk';
import { ProjectSpec } from 'services/stream/projects';
import projectStore from 'stores/projects';
import streamStore from 'stores/stream';
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
  const [projects, setProjects] = useState<List<Project>>(List());
  const [isLoading, setIsLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [canceler] = useState(new AbortController());
  const { canCreateProject } = usePermissions();
  const ProjectCreateModal = useModal(ProjectCreateModalComponent);
  const config = useMemo(() => configForWorkspace(id), [id]);
  const { settings, updateSettings } = useSettings<WorkspaceDetailsSettings>(config);

  const loadableProjects: Loadable<List<Project>> = useObservable(
    projectStore.getProjectsByWorkspace(id),
  );

  useEffect(() => {
    Loadable.match(loadableProjects, {
      _: () => {},
      Loaded: (data) => {
        setProjects(data);
        setIsLoading(false);
        streamStore.emit(new ProjectSpec([id]));
      },
    });
  }, [loadableProjects, id]);

  useEffect(() => {
    setTotal(projects.size);
  }, [projects]);

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

  const onProjectRemove = useCallback(
    (id: number) => {
      setProjects((prev) => prev.filter((p) => p.id !== id));
    },
    [setProjects],
  );

  const onProjectEdit = useCallback(
    (id: number, name: string, archived: boolean) => {
      setProjects((prev) => prev.map((p) => (p.id === id ? { ...p, archived, name } : p)));
    },
    [setProjects],
  );

  const columns = useMemo(() => {
    const projectNameRenderer = (value: string, record: Project) => (
      <Link path={paths.projectDetails(record.id)}>{value}</Link>
    );

    const actionRenderer: GenericRenderer<Project> = (_, record) => (
      <ProjectActionDropdown
        project={record}
        workspaceArchived={workspace?.archived}
        onDelete={() => onProjectRemove(record.id)}
        onEdit={(name: string, archived: boolean) => onProjectEdit(record.id, name, archived)}
        onMove={() => onProjectRemove(record.id)}
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
  }, [saveProjectDescription, workspace?.archived, users, onProjectEdit, onProjectRemove]);

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
        onDelete={() => onProjectRemove(record.id)}
        onEdit={(name: string, archived: boolean) => onProjectEdit(record.id, name, archived)}
        onMove={() => onProjectRemove(record.id)}>
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
                key={project.id}
                project={project}
                workspaceArchived={workspace?.archived}
                onEdit={(name: string, archived: boolean) =>
                  onProjectEdit(project.id, name, archived)
                }
                onRemove={() => onProjectRemove(project.id)}
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
            dataSource={projects.toArray()}
            loading={isLoading || Loadable.isNotLoaded(loadableUsers)}
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
    isLoading,
    loadableUsers,
    onProjectEdit,
    onProjectRemove,
    pageRef,
    projects,
    settings,
    total,
    updateSettings,
    workspace?.archived,
  ]);

  useEffect(() => {
    projectStore.fetch(id);
  }, [id]);

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
                    <Button onClick={handleProjectCreateClick}>New Project</Button>
                  )}
              </div>
            </Row>
          </Column>
        </Row>
      </Section>
      <Spinner conditionalRender spinning={isLoading}>
        {projects.size !== 0 ? (
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
