import Button from 'hew/Button';
import Card from 'hew/Card';
import Column from 'hew/Column';
import Message from 'hew/Message';
import { useModal } from 'hew/Modal';
import Row from 'hew/Row';
import Section from 'hew/Section';
import Select, { Option } from 'hew/Select';
import Spinner from 'hew/Spinner';
import Toggle from 'hew/Toggle';
import { Loadable } from 'hew/utils/loadable';
import _ from 'lodash';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import GridListRadioGroup, { GridListView } from 'components/GridListRadioGroup';
import Link from 'components/Link';
import Page from 'components/Page';
import InteractiveTable, {
  ColumnDef,
  onRightClickableCell,
} from 'components/Table/InteractiveTable';
import {
  checkmarkRenderer,
  GenericRenderer,
  getFullPaginationConfig,
  stateRenderer,
  userRenderer,
} from 'components/Table/Table';
import WorkspaceCreateModalComponent from 'components/WorkspaceCreateModal';
import useFeature from 'hooks/useFeature';
import usePermissions from 'hooks/usePermissions';
import usePolling from 'hooks/usePolling';
import usePrevious from 'hooks/usePrevious';
import { useSettings } from 'hooks/useSettings';
import { paths } from 'routes/utils';
import { getWorkspaces } from 'services/api';
import { V1GetWorkspacesRequestSortBy } from 'services/api-ts-sdk';
import clusterStore from 'stores/cluster';
import userStore from 'stores/users';
import { Workspace } from 'types';
import { useObservable } from 'utils/observable';
import { validateDetApiEnum } from 'utils/service';

import WorkspaceActionDropdown from './WorkspaceList/WorkspaceActionDropdown';
import WorkspaceCard from './WorkspaceList/WorkspaceCard';
import settingsConfig, {
  DEFAULT_COLUMN_WIDTHS,
  WhoseWorkspaces,
  WorkspaceColumnName,
  WorkspaceListSettings,
} from './WorkspaceList.settings';

const WorkspaceList: React.FC = () => {
  const currentUser = Loadable.getOrElse(undefined, useObservable(userStore.currentUser));
  const loadableUsers = useObservable(userStore.getUsers());
  const users = Loadable.getOrElse([], loadableUsers);
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [total, setTotal] = useState(0);
  const [pageError, setPageError] = useState<Error>();
  const [isLoading, setIsLoading] = useState(true);
  const pageRef = useRef<HTMLElement>(null);
  const canceler = useRef<AbortController>();
  const f_flat_runs = useFeature().isOn('flat_runs');

  const { canCreateWorkspace } = usePermissions();

  const WorkspaceCreateModal = useModal(WorkspaceCreateModalComponent);

  const { settings, updateSettings } = useSettings<WorkspaceListSettings>(settingsConfig);

  const openWorkspaceCreateModal = () => {
    clusterStore.fetchKubernetesResourceManagers();
    WorkspaceCreateModal.open();
  };

  const fetchWorkspaces = useCallback(async () => {
    if (!settings) return;

    const abortController = new AbortController();
    canceler.current = abortController;
    try {
      const response = await getWorkspaces(
        {
          archived: settings.archived ? undefined : false,
          limit: settings.view === GridListView.Grid ? 0 : settings.tableLimit,
          name: settings.name,
          offset: settings.view === GridListView.Grid ? 0 : settings.tableOffset,
          orderBy: settings.sortDesc ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC',
          sortBy: validateDetApiEnum(V1GetWorkspacesRequestSortBy, settings.sortKey),
          users: settings.user,
        },
        { signal: abortController.signal },
      );
      setTotal((response.pagination.total ?? 1) - 1); // -1 because we do not display immutable ws
      setWorkspaces((prev) => {
        const withoutDefault = response.workspaces.filter((w) => !w.immutable);
        if (_.isEqual(prev, withoutDefault)) return prev;
        return withoutDefault;
      });
    } catch (e) {
      if (!abortController.signal.aborted) {
        setPageError((p) => p || (e as Error));
      }
    } finally {
      if (!abortController.signal.aborted) {
        setIsLoading(false);
      }
    }
  }, [settings]);

  useEffect(() => {
    canceler.current?.abort();
  }, [settings]);

  usePolling(fetchWorkspaces, { rerunOnNewFn: true, runImmediately: true });

  const handleWhoseSelect = useCallback(
    (value: unknown) => {
      setIsLoading(true);
      updateSettings({
        tableOffset: settingsConfig.settings.tableOffset.defaultValue,
        whose: value as WhoseWorkspaces,
      });
    },
    [updateSettings],
  );

  const handleSortSelect = useCallback(
    (value: unknown) => {
      updateSettings({
        sortDesc: value === V1GetWorkspacesRequestSortBy.NAME ? false : true,
        sortKey: value as V1GetWorkspacesRequestSortBy | undefined,
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
      case WhoseWorkspaces.All:
        updateSettings({ user: undefined });
        break;
      case WhoseWorkspaces.Mine:
        updateSettings({ user: currentUser ? [currentUser.id.toString()] : undefined });
        break;
      case WhoseWorkspaces.Others:
        updateSettings({
          user: users.filter((u) => u.id !== currentUser?.id).map((u) => u.id.toString()),
        });
        break;
    }
  }, [currentUser, loadableUsers, prevWhose, settings.whose, updateSettings, users]);

  const columns = useMemo(() => {
    if (Loadable.isNotLoaded(loadableUsers)) return [];

    const workspaceNameRenderer = (value: string, record: Workspace) => (
      <Link path={paths.workspaceDetails(record.id)}>{value}</Link>
    );

    const actionRenderer: GenericRenderer<Workspace> = (_, record) => (
      <WorkspaceActionDropdown workspace={record} onComplete={fetchWorkspaces} />
    );

    return [
      {
        dataIndex: 'name',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['name'],
        key: V1GetWorkspacesRequestSortBy.NAME,
        onCell: () => ({ ...onRightClickableCell(), 'data-testid': 'name' }),
        render: workspaceNameRenderer,
        title: 'Name',
      },
      {
        align: 'right',
        dataIndex: 'numProjects',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['numProjects'],
        key: 'numProjects',
        title: 'Projects',
      },
      {
        align: 'center',
        dataIndex: 'userId',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['userId'],
        key: 'user',
        render: (_, r) => userRenderer(users.find((u) => u.id === r.userId)),
        title: 'User',
      },
      {
        align: 'center',
        dataIndex: 'archived',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['archived'],
        key: 'archived',
        render: checkmarkRenderer,
        title: 'Archived',
      },
      {
        align: 'center',
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
    ] as ColumnDef<Workspace>[];
  }, [fetchWorkspaces, loadableUsers, users]);

  const switchShowArchived = useCallback(
    (showArchived: boolean) => {
      if (!settings) return;

      let newColumns: WorkspaceColumnName[];
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
      children,
    }: {
      children: React.ReactNode;
      onVisibleChange?: (visible: boolean) => void;
      record: Workspace;
    }) => (
      <WorkspaceActionDropdown isContextMenu workspace={record} onComplete={fetchWorkspaces}>
        {children}
      </WorkspaceActionDropdown>
    ),
    [fetchWorkspaces],
  );

  const workspacesList = useMemo(() => {
    if (!settings) return <Spinner spinning />;

    switch (settings.view) {
      case GridListView.Grid:
        return (
          <Card.Group size="medium">
            {workspaces.map((workspace) => (
              <WorkspaceCard
                fetchWorkspaces={fetchWorkspaces}
                key={workspace.id}
                workspace={workspace}
              />
            ))}
          </Card.Group>
        );
      case GridListView.List:
        return (
          <InteractiveTable<Workspace, WorkspaceListSettings>
            columns={columns}
            containerRef={pageRef}
            ContextMenu={actionDropdown}
            dataSource={workspaces}
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
    fetchWorkspaces,
    isLoading,
    loadableUsers,
    settings,
    total,
    updateSettings,
    workspaces,
  ]);

  useEffect(() => {
    return () => canceler.current?.abort();
  }, []);

  if (pageError) {
    return <Message icon="warning" title="Unable to fetch workspaces" />;
  }

  return (
    <Page
      breadcrumb={[
        {
          breadcrumbName: 'Workspaces',
          path: paths.workspaceList(),
        },
      ]}
      containerRef={pageRef}
      id="workspaces"
      options={
        <Button
          data-testid="newWorkspace"
          disabled={!canCreateWorkspace}
          onClick={openWorkspaceCreateModal}>
          New Workspace
        </Button>
      }
      title="Workspaces">
      <Section>
        <Row wrap>
          <Column>
            <Select
              data-testid="whose"
              value={settings.whose}
              width={180}
              onSelect={handleWhoseSelect}>
              <Option value={WhoseWorkspaces.All}>All Workspaces</Option>
              <Option value={WhoseWorkspaces.Mine}>My Workspaces</Option>
              <Option value={WhoseWorkspaces.Others}>Others&apos; Workspaces</Option>
            </Select>
          </Column>
          <Column align="right">
            <Row wrap>
              <Toggle
                checked={settings.archived}
                label="Show Archived"
                onChange={switchShowArchived}
              />
              <Select
                data-testid="sort"
                value={settings.sortKey}
                width={170}
                onSelect={handleSortSelect}>
                <Option value={V1GetWorkspacesRequestSortBy.NAME}>Alphabetical</Option>
                <Option value={V1GetWorkspacesRequestSortBy.ID}>Newest to Oldest</Option>
              </Select>
              {settings && <GridListRadioGroup value={settings.view} onChange={handleViewChange} />}
            </Row>
          </Column>
        </Row>
      </Section>
      <Spinner conditionalRender spinning={isLoading}>
        {workspaces.length !== 0 ? (
          workspacesList
        ) : settings.whose === WhoseWorkspaces.All && settings.archived && !isLoading ? (
          <Message
            description={`Create a workspace to keep track of related projects and ${f_flat_runs ? 'searches' : 'experiments'}.`}
            icon="workspaces"
            testId="noWorkspaces"
            title="No Workspaces"
          />
        ) : (
          <Message
            icon="warning"
            testId="noMatchingWorkspaces"
            title="No workspaces matching the current filters"
          />
        )}
      </Spinner>
      <WorkspaceCreateModal.Component />
    </Page>
  );
};

export default WorkspaceList;
