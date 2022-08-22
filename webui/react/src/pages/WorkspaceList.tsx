import { Button, Select, Space } from 'antd';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import Grid, { GridMode } from 'components/Grid';
import GridListRadioGroup, { GridListView } from 'components/GridListRadioGroup';
import InteractiveTable, { ColumnDef,
  InteractiveTableSettings,
  onRightClickableCell } from 'components/InteractiveTable';
import Link from 'components/Link';
import Page from 'components/Page';
import SelectFilter from 'components/SelectFilter';
import { checkmarkRenderer, GenericRenderer,
  getFullPaginationConfig, stateRenderer, userRenderer } from 'components/Table';
import Toggle from 'components/Toggle';
import { useStore } from 'contexts/Store';
import useModalWorkspaceCreate from 'hooks/useModal/Workspace/useModalWorkspaceCreate';
import usePolling from 'hooks/usePolling';
import useSettings, { UpdateSettings } from 'hooks/useSettings';
import { paths } from 'routes/utils';
import { getWorkspaces } from 'services/api';
import { V1GetWorkspacesRequestSortBy } from 'services/api-ts-sdk';
import Icon from 'shared/components/Icon';
import Message, { MessageType } from 'shared/components/Message';
import Spinner from 'shared/components/Spinner';
import { isEqual } from 'shared/utils/data';
import { validateDetApiEnum } from 'shared/utils/service';
import { ShirtSize } from 'themes';
import { Workspace } from 'types';

import css from './WorkspaceList.module.scss';
import settingsConfig, { DEFAULT_COLUMN_WIDTHS,
  WhoseWorkspaces, WorkspaceColumnName, WorkspaceListSettings } from './WorkspaceList.settings';
import WorkspaceActionDropdown from './WorkspaceList/WorkspaceActionDropdown';
import WorkspaceCard from './WorkspaceList/WorkspaceCard';

const { Option } = Select;

const WorkspaceList: React.FC = () => {
  const { users, auth: { user } } = useStore();
  const [ workspaces, setWorkspaces ] = useState<Workspace[]>([]);
  const [ total, setTotal ] = useState(0);
  const [ pageError, setPageError ] = useState<Error>();
  const [ isLoading, setIsLoading ] = useState(true);
  const pageRef = useRef<HTMLElement>(null);
  const [ canceler ] = useState(new AbortController());

  const { contextHolder, modalOpen } = useModalWorkspaceCreate();

  const {
    settings,
    updateSettings,
  } = useSettings<WorkspaceListSettings>(settingsConfig);

  const handleWorkspaceCreateClick = useCallback(() => modalOpen(), [ modalOpen ]);

  const fetchWorkspaces = useCallback(async () => {
    try {
      const response = await getWorkspaces({
        archived: settings.archived ? undefined : false,
        limit: settings.view === GridListView.Grid ? 0 : settings.tableLimit,
        name: settings.name,
        offset: settings.view === GridListView.Grid ? 0 : settings.tableOffset,
        orderBy: settings.sortDesc ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC',
        sortBy: validateDetApiEnum(V1GetWorkspacesRequestSortBy, settings.sortKey),
        users: settings.user,
      }, { signal: canceler.signal });
      setTotal((response.pagination.total ?? 1) - 1); // -1 because we do not display immutable ws
      setWorkspaces((prev) => {
        const withoutDefault = response.workspaces.filter((w) => !w.immutable);
        if (isEqual(prev, withoutDefault)) return prev;
        return withoutDefault;
      });
    } catch (e) {
      if (!pageError) setPageError(e as Error);
    } finally {
      setIsLoading(false);
    }
  }, [ canceler.signal,
    pageError,
    settings.archived,
    settings.name,
    settings.sortDesc,
    settings.sortKey,
    settings.tableLimit,
    settings.tableOffset,
    settings.user,
    settings.view ]);

  usePolling(fetchWorkspaces);

  const handleViewSelect = useCallback((value) => {
    updateSettings({ whose: value });

  }, [ updateSettings ]);

  const handleSortSelect = useCallback((value) => {
    updateSettings({
      sortDesc: value === V1GetWorkspacesRequestSortBy.NAME ? false : true,
      sortKey: value,
    });
  }, [ updateSettings ]);

  const handleViewChange = useCallback((value: GridListView) => {
    updateSettings({ view: value });
  }, [ updateSettings ]);

  useEffect(() => {
    switch (settings.whose) {
      case WhoseWorkspaces.All:
        updateSettings({ user: undefined });
        break;
      case WhoseWorkspaces.Mine:
        updateSettings({ user: user ? [ user.username ] : undefined });
        break;
      case WhoseWorkspaces.Others:
        updateSettings({ user: users.filter((u) => u.id !== user?.id).map((u) => u.username) });
        break;
    }
  }, [ updateSettings, user, users, settings.whose ]);

  const columns = useMemo(() => {
    const workspaceNameRenderer = (value: string, record: Workspace) => (
      <Link path={paths.workspaceDetails(record.id)}>{value}</Link>
    );

    const actionRenderer: GenericRenderer<Workspace> = (_, record) => (
      <WorkspaceActionDropdown
        curUser={user}
        workspace={record}
        onComplete={fetchWorkspaces}
      />
    );

    return [
      {
        dataIndex: 'name',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['name'],
        key: V1GetWorkspacesRequestSortBy.NAME,
        onCell: onRightClickableCell,
        render: workspaceNameRenderer,
        title: 'Name',
      },
      {
        dataIndex: 'numProjects',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['numProjects'],
        key: 'numProjects',
        title: 'Projects',
      },
      {
        dataIndex: 'userId',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['userId'],
        key: 'user',
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
    ] as ColumnDef<Workspace>[];
  }, [ fetchWorkspaces, user ]);

  const switchShowArchived = useCallback((showArchived: boolean) => {
    let newColumns: WorkspaceColumnName[];
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
      <WorkspaceActionDropdown
        curUser={user}
        workspace={record}
        onComplete={fetchWorkspaces}
        onVisibleChange={onVisibleChange}>
        {children}
      </WorkspaceActionDropdown>
    ),
    [ fetchWorkspaces, user ],
  );

  const workspacesList = useMemo(() => {
    switch (settings.view) {
      case GridListView.Grid:
        return (
          <Grid
            gap={ShirtSize.medium}
            minItemWidth={300}
            mode={GridMode.AutoFill}>
            {workspaces.map((workspace) => (
              <WorkspaceCard
                curUser={user}
                fetchWorkspaces={fetchWorkspaces}
                key={workspace.id}
                workspace={workspace}
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
            dataSource={workspaces}
            loading={isLoading}
            pagination={getFullPaginationConfig({
              limit: settings.tableLimit,
              offset: settings.tableOffset,
            }, total)}
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
    fetchWorkspaces,
    isLoading,
    settings,
    total,
    updateSettings,
    user,
    workspaces,
  ]);

  useEffect(() => {
    setIsLoading(true);
    fetchWorkspaces().then(() => setIsLoading(false));
  }, [ fetchWorkspaces ]);

  useEffect(() => {
    return () => canceler.abort();
  }, [ canceler ]);

  if (pageError) {
    return <Message title="Unable to fetch workspaces" type={MessageType.Warning} />;
  }

  return (
    <Page
      className={css.base}
      containerRef={pageRef}
      id="workspaces"
      options={<Button onClick={handleWorkspaceCreateClick}>New Workspace</Button>}
      title="Workspaces">
      <div className={css.controls}>
        <SelectFilter
          dropdownMatchSelectWidth={160}
          showSearch={false}
          value={settings.whose}
          onSelect={handleViewSelect}>
          <Option value={WhoseWorkspaces.All}>All Workspaces</Option>
          <Option value={WhoseWorkspaces.Mine}>My Workspaces</Option>
          <Option value={WhoseWorkspaces.Others}>Others&apos; Workspaces</Option>
        </SelectFilter>
        <Space wrap>
          <Toggle
            checked={settings.archived}
            prefixLabel="Show Archived"
            onChange={switchShowArchived}
          />
          <SelectFilter
            dropdownMatchSelectWidth={150}
            showSearch={false}
            value={settings.sortKey}
            onSelect={handleSortSelect}>
            <Option value={V1GetWorkspacesRequestSortBy.NAME}>Alphabetical</Option>
            <Option value={V1GetWorkspacesRequestSortBy.ID}>
              Newest to Oldest
            </Option>
          </SelectFilter>
          <GridListRadioGroup value={settings.view} onChange={handleViewChange} />
        </Space>
      </div>
      <Spinner spinning={isLoading}>
        {workspaces.length !== 0 ? (
          workspacesList
        ) :
          (settings.whose === WhoseWorkspaces.All && settings.archived && !isLoading) ?
            (
              <div className={css.emptyBase}>
                <div className={css.icon}>
                  <Icon name="workspaces" size="mega" />
                </div>
                <h4>No Workspaces</h4>
                <p className={css.description}>
                  Create a workspace to keep track of related projects and experiments.
                </p>
              </div>
            ) : (
              <Message
                title="No workspaces matching the current filters"
                type={MessageType.Empty}
              />
            )}
      </Spinner>
      {contextHolder}
    </Page>
  );
};

export default WorkspaceList;
