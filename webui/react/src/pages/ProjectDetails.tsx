import { InfoCircleOutlined } from '@ant-design/icons';
import { Space, Tabs, Tooltip } from 'antd';
import React, { useCallback, useRef, useState } from 'react';
import { useParams } from 'react-router-dom';

import BreadcrumbBar from 'components/BreadcrumbBar';
import DynamicTabs from 'components/DynamicTabs';
import Page from 'components/Page';
import PageNotFound from 'components/PageNotFound';
import { useStore } from 'contexts/Store';
import usePermissions from 'hooks/usePermissions';
import { paths } from 'routes/utils';
import { getProject, getWorkspace } from 'services/api';
import Icon from 'shared/components/Icon/Icon';
import Message, { MessageType } from 'shared/components/Message';
import Spinner from 'shared/components/Spinner';
import usePolling from 'shared/hooks/usePolling';
import { isEqual, isNumber } from 'shared/utils/data';
import { isNotFound } from 'shared/utils/service';
import { Project, Workspace } from 'types';
import handleError from 'utils/error';

import ExperimentList from './ExperimentList';
import NoPermissions from './NoPermissions';
import css from './ProjectDetails.module.scss';
import ProjectNotes from './ProjectNotes';
import TrialsComparison from './TrialsComparison/TrialsComparison';
import ProjectActionDropdown from './WorkspaceDetails/ProjectActionDropdown';

const { TabPane } = Tabs;

type Params = {
  projectId: string;
};

const ProjectDetails: React.FC = () => {
  const {
    auth: { user },
  } = useStore();
  const { projectId } = useParams<Params>();

  const [project, setProject] = useState<Project>();

  const permissions = usePermissions();
  const [pageError, setPageError] = useState<Error>();
  const [canceler] = useState(new AbortController());
  const pageRef = useRef<HTMLElement>(null);

  const [workspace, setWorkspace] = useState<Workspace>();

  const id = parseInt(projectId ?? '1');

  const fetchWorkspace = useCallback(async () => {
    const workspaceId = project?.workspaceId;
    if (!isNumber(workspaceId)) return;
    try {
      const response = await getWorkspace({ id: workspaceId });
      setWorkspace(response);
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch workspace.' });
    }
  }, [project?.workspaceId]);

  const fetchProject = useCallback(async () => {
    try {
      const response = await getProject({ id }, { signal: canceler.signal });
      setProject((prev) => {
        if (isEqual(prev, response)) return prev;
        return response;
      });
      setPageError(undefined);
    } catch (e) {
      if (!pageError) setPageError(e as Error);
    }
  }, [canceler.signal, id, pageError]);

  usePolling(fetchProject, { rerunOnNewFn: true });
  usePolling(fetchWorkspace, { rerunOnNewFn: true });

  // const fetchExperimentGroups = useCallback(async () => {
  //   try {
  //     const groups = await getExperimentGroups({ projectId: id }, { signal: canceler.signal });
  //     groups.sort((a, b) => alphaNumericSorter(a.name, b.name));
  //     setExperimentGroups(groups);
  //   } catch (e) {
  //     handleError(e);
  //   }
  // }, [canceler.signal, id]);

  // const columns = useMemo(() => {
  //   const tagsRenderer = (value: string, record: ExperimentItem) => (
  //     <TagList
  //       disabled={record.archived || project?.archived || !canEditExperiment}
  //       tags={record.labels}
  //       onChange={experimentTags.handleTagListChange(record.id)}
  //     />
  //   );

  //   const groupRenderer = (value: string, record: ExperimentItem) => {
  //     const handleGroupSelect = async (option?: OptionType | string) => {
  //       try {
  //         if (option === undefined || option === '') {
  //           if (record.groupId === undefined || record.groupId === null) return;
  //           await patchExperiment({
  //             body: { groupId: null },
  //             experimentId: record.id,
  //           });
  //         } else if (isString(option)) {
  //           const response = await createExperimentGroup({
  //             name: option,
  //             projectId: record.projectId,
  //           });
  //           await patchExperiment({ body: { groupId: response.id }, experimentId: record.id });
  //           await fetchExperimentGroups();
  //         } else {
  //           if (record.groupId === option.value) return;
  //           await patchExperiment({
  //             body: { groupId: isNumber(option.value) ? option.value : parseInt(option.value) },
  //             experimentId: record.id,
  //           });
  //         }
  //       } catch (e) {
  //         handleError(e, {
  //           isUserTriggered: true,
  //           publicMessage: 'Unable to save experiment group.',
  //           silent: false,
  //         });
  //         return e as Error;
  //       }
  //     };
  //     return (
  //       <AutoComplete
  //         allowClear={true}
  //         initialValue={record.groupName}
  //         options={experimentGroups.map((group) => ({ label: group.name, value: group.id }))}
  //         placeholder="Add experiment group..."
  //         onSave={handleGroupSelect}
  //       />
  //     );
  //   };

  //   const actionRenderer: ExperimentRenderer = (_, record) => {
  //     return <ContextMenu record={record} />;
  //   };

  //   const descriptionRenderer = (value: string, record: ExperimentItem) => (
  //     <TextEditorModal
  //       disabled={record.archived || !canEditExperiment}
  //       placeholder={record.archived ? 'Archived' : canEditExperiment ? 'Add description...' : ''}
  //       title="Edit description"
  //       value={value}
  //       onSave={(newDescription: string) => saveExperimentDescription(newDescription, record.id)}
  //     />
  //   );

  //   const forkedFromRenderer = (value: string | number | undefined): React.ReactNode =>
  //     value ? <Link path={paths.experimentDetails(value)}>{value}</Link> : null;

  //   return [
  //     {
  //       align: 'right',
  //       dataIndex: 'id',
  //       defaultWidth: DEFAULT_COLUMN_WIDTHS['id'],
  //       key: V1GetExperimentsRequestSortBy.ID,
  //       onCell: onRightClickableCell,
  //       render: experimentNameRenderer,
  //       sorter: true,
  //       title: 'ID',
  //     },
  //     {
  //       dataIndex: 'name',
  //       defaultWidth: DEFAULT_COLUMN_WIDTHS['name'],
  //       filterDropdown: nameFilterSearch,
  //       filterIcon: tableSearchIcon,
  //       isFiltered: (settings: ProjectDetailsSettings) => !!settings.search,
  //       key: V1GetExperimentsRequestSortBy.NAME,
  //       onCell: onRightClickableCell,
  //       render: experimentNameRenderer,
  //       sorter: true,
  //       title: 'Name',
  //     },
  //     {
  //       dataIndex: 'description',
  //       defaultWidth: DEFAULT_COLUMN_WIDTHS['description'],
  //       onCell: onRightClickableCell,
  //       render: descriptionRenderer,
  //       title: 'Description',
  //     },
  //     {
  //       dataIndex: 'tags',
  //       defaultWidth: DEFAULT_COLUMN_WIDTHS['tags'],
  //       filterDropdown: labelFilterDropdown,
  //       filters: labels.map((label) => ({ text: label, value: label })),
  //       isFiltered: (settings: ProjectDetailsSettings) => !!settings.label,
  //       key: 'labels',
  //       onCell: onRightClickableCell,
  //       render: tagsRenderer,
  //       title: 'Tags',
  //     },
  //     {
  //       dataIndex: 'groupName',
  //       defaultWidth: DEFAULT_COLUMN_WIDTHS['groupName'],
  //       key: V1GetExperimentsRequestSortBy.GROUP,
  //       onCell: onRightClickableCell,
  //       sorter: true,
  //       title: 'Group',
  //     },
  //     {
  //       align: 'right',
  //       dataIndex: 'forkedFrom',
  //       defaultWidth: DEFAULT_COLUMN_WIDTHS['forkedFrom'],
  //       key: V1GetExperimentsRequestSortBy.FORKEDFROM,
  //       onCell: onRightClickableCell,
  //       render: forkedFromRenderer,
  //       sorter: true,
  //       title: 'Forked From',
  //     },
  //     {
  //       align: 'right',
  //       dataIndex: 'startTime',
  //       defaultWidth: DEFAULT_COLUMN_WIDTHS['startTime'],
  //       key: V1GetExperimentsRequestSortBy.STARTTIME,
  //       onCell: onRightClickableCell,
  //       render: (_: number, record: ExperimentItem): React.ReactNode =>
  //         relativeTimeRenderer(new Date(record.startTime)),
  //       sorter: true,
  //       title: 'Start Time',
  //     },
  //     {
  //       align: 'right',
  //       dataIndex: 'duration',
  //       defaultWidth: DEFAULT_COLUMN_WIDTHS['duration'],
  //       key: 'duration',
  //       onCell: onRightClickableCell,
  //       render: experimentDurationRenderer,
  //       title: 'Duration',
  //     },
  //     {
  //       align: 'right',
  //       dataIndex: 'numTrials',
  //       defaultWidth: DEFAULT_COLUMN_WIDTHS['numTrials'],
  //       key: V1GetExperimentsRequestSortBy.NUMTRIALS,
  //       onCell: onRightClickableCell,
  //       sorter: true,
  //       title: 'Trials',
  //     },
  //     {
  //       dataIndex: 'state',
  //       defaultWidth: DEFAULT_COLUMN_WIDTHS['state'],
  //       filterDropdown: stateFilterDropdown,
  //       filters: [
  //         RunState.Active,
  //         RunState.Paused,
  //         RunState.Canceled,
  //         RunState.Completed,
  //         RunState.Error,
  //       ].map((value) => ({
  //         text: <Badge state={value} type={BadgeType.State} />,
  //         value,
  //       })),
  //       isFiltered: () => !!settings.state,
  //       key: V1GetExperimentsRequestSortBy.STATE,
  //       render: stateRenderer,
  //       sorter: true,
  //       title: 'State',
  //     },
  //     {
  //       dataIndex: 'searcherType',
  //       defaultWidth: DEFAULT_COLUMN_WIDTHS['searcherType'],
  //       key: 'searcherType',
  //       onCell: onRightClickableCell,
  //       title: 'Searcher Type',
  //     },
  //     {
  //       dataIndex: 'resourcePool',
  //       defaultWidth: DEFAULT_COLUMN_WIDTHS['resourcePool'],
  //       key: V1GetExperimentsRequestSortBy.RESOURCEPOOL,
  //       onCell: onRightClickableCell,
  //       sorter: true,
  //       title: 'Resource Pool',
  //     },
  //     {
  //       align: 'center',
  //       dataIndex: 'progress',
  //       defaultWidth: DEFAULT_COLUMN_WIDTHS['progress'],
  //       key: V1GetExperimentsRequestSortBy.PROGRESS,
  //       render: experimentProgressRenderer,
  //       sorter: true,
  //       title: 'Progress',
  //     },
  //     {
  //       align: 'center',
  //       dataIndex: 'archived',
  //       defaultWidth: DEFAULT_COLUMN_WIDTHS['archived'],
  //       key: 'archived',
  //       render: checkmarkRenderer,
  //       title: 'Archived',
  //     },
  //     {
  //       dataIndex: 'user',
  //       defaultWidth: DEFAULT_COLUMN_WIDTHS['user'],
  //       filterDropdown: userFilterDropdown,
  //       filters: users.map((user) => ({ text: getDisplayName(user), value: user.id })),
  //       isFiltered: (settings: ProjectDetailsSettings) => !!settings.user,
  //       key: V1GetExperimentsRequestSortBy.USER,
  //       render: userRenderer,
  //       sorter: true,
  //       title: 'User',
  //     },
  //     {
  //       align: 'right',
  //       className: 'fullCell',
  //       dataIndex: 'action',
  //       defaultWidth: DEFAULT_COLUMN_WIDTHS['action'],
  //       fixed: 'right',
  //       key: 'action',
  //       onCell: onRightClickableCell,
  //       render: actionRenderer,
  //       title: '',
  //       width: DEFAULT_COLUMN_WIDTHS['action'],
  //     },
  //   ] as ColumnDef<ExperimentItem>[];
  // }, [
  //   nameFilterSearch,
  //   tableSearchIcon,
  //   labelFilterDropdown,
  //   labels,
  //   stateFilterDropdown,
  //   userFilterDropdown,
  //   users,
  //   project,
  //   experimentTags,
  //   canEditExperiment,
  //   fetchExperimentGroups,
  //   settings,
  //   saveExperimentDescription,
  //   ContextMenu,
  // ]);

  // const onGroupAction = useCallback(() => {
  //   fetchExperiments();
  // }, [fetchExperiments]);

  // const { contextHolder: modalExperimentGroupsContextHolder, modalOpen: openManageGroups } =
  //   useModalExperimentGroups({ onAction: onGroupAction, projectId: id });

  // const handleManageGroupsClick = useCallback(() => {
  //   openManageGroups({});
  // }, [openManageGroups]);

  // const ExperimentTabOptions = useMemo(() => {
  //   const getMenuProps = (): { items: MenuProps['items']; onClick: MenuProps['onClick'] } => {
  //     enum MenuKey {
  //       SWITCH_ARCHIVED = 'switchArchive',
  //       COLUMNS = 'columns',
  //       RESULT_FILTER = 'resetFilters',
  //       GROUPS = 'groups',
  //     }

  //     const funcs = {
  //       [MenuKey.SWITCH_ARCHIVED]: () => {
  //         switchShowArchived(!settings.archived);
  //       },
  //       [MenuKey.COLUMNS]: () => {
  //         handleCustomizeColumnsClick();
  //       },
  //       [MenuKey.RESULT_FILTER]: () => {
  //         resetFilters();
  //       },
  //       [MenuKey.GROUPS]: () => {
  //         handleManageGroupsClick();
  //       },
  //     };

  //     const onItemClick: MenuProps['onClick'] = (e) => {
  //       funcs[e.key as MenuKey]();
  //     };

  //     const menuItems: MenuProps['items'] = [
  //       {
  //         key: MenuKey.SWITCH_ARCHIVED,
  //         label: settings.archived ? 'Hide Archived' : 'Show Archived',
  //       },
  //       { key: MenuKey.COLUMNS, label: 'Columns' },
  //       { key: MenuKey.GROUPS, label: 'Groups' },
  //     ];
  //     if (filterCount > 0) {
  //       menuItems.push({ key: MenuKey.RESULT_FILTER, label: `Clear Filters (${filterCount})` });
  //     }
  //     return { items: menuItems, onClick: onItemClick };
  //   };

  //   return (
  //     <div className={css.tabOptions}>
  //       <Space className={css.actionList}>
  //         <Toggle
  //           checked={settings.archived}
  //           prefixLabel="Show Archived"
  //           onChange={switchShowArchived}
  //         />
  //         <Button onClick={handleCustomizeColumnsClick}>Columns</Button>
  //         <Button onClick={handleManageGroupsClick}>Groups</Button>
  //         <FilterCounter activeFilterCount={filterCount} onReset={resetFilters} />
  //       </Space>
  //       <div className={css.actionOverflow} title="Open actions menu">
  //         <Dropdown
  //           overlay={<Menu {...getMenuProps()} />}
  //           placement="bottomRight"
  //           trigger={['click']}>
  //           <div>
  //             <Icon name="overflow-vertical" />
  //           </div>
  //         </Dropdown>
  //       </div>
  //     </div>
  //   );
  // }, [
  //   filterCount,
  //   handleCustomizeColumnsClick,
  //   handleManageGroupsClick,
  //   resetFilters,
  //   settings.archived,
  //   switchShowArchived,
  // ]);

  // const tabs: TabInfo[] = useMemo(() => {
  //   return [
  //     {
  //       body: (
  //         <div className={css.experimentTab}>
  //           <TableBatch
  //             actions={batchActions.map((action) => ({
  //               disabled: !availableBatchActions.includes(action),
  //               label: action,
  //               value: action,
  //             }))}
  //             selectedRowCount={(settings.row ?? []).length}
  //             onAction={handleBatchAction}
  //             onClear={clearSelected}
  //           />
  //           <InteractiveTable
  //             areRowsSelected={!!settings.row}
  //             columns={columns}
  //             containerRef={pageRef}
  //             ContextMenu={ContextMenu}
  //             dataSource={experiments}
  //             loading={isLoading}
  //             numOfPinned={(settings.pinned[id] ?? []).length}
  //             pagination={getFullPaginationConfig(
  //               {
  //                 limit: settings.tableLimit,
  //                 offset: settings.tableOffset,
  //               },
  //               total,
  //             )}
  //             rowClassName={defaultRowClassName({ clickable: false })}
  //             rowKey="id"
  //             rowSelection={{
  //               onChange: handleTableRowSelect,
  //               preserveSelectedRowKeys: true,
  //               selectedRowKeys: settings.row ?? [],
  //             }}
  //             scroll={{
  //               y: `calc(100vh - ${availableBatchActions.length === 0 ? '230' : '280'}px)`,
  //             }}
  //             settings={settings as InteractiveTableSettings}
  //             showSorterTooltip={false}
  //             size="small"
  //             updateSettings={updateSettings as UpdateSettings<InteractiveTableSettings>}
  //           />
  //         </div>
  //       ),
  //       options: ExperimentTabOptions,
  //       title: 'Experiments',
  //     },
  //     {
  //       body: (
  //         <PaginatedNotesCard
  //           disabled={project?.archived}
  //           notes={project?.notes ?? []}
  //           onDelete={handleDeleteNote}
  //           onNewPage={handleNewNotesPage}
  //           onSave={handleSaveNotes}
  //         />
  //       ),
  //       options: (
  //         <div className={css.tabOptions}>
  //           <Button type="text" onClick={handleNewNotesPage}>
  //             + New Page
  //           </Button>
  //         </div>
  //       ),
  //       title: 'Notes',
  //     },
  //   ];
  // }, [
  //   settings,
  //   handleBatchAction,
  //   clearSelected,
  //   columns,
  //   ContextMenu,
  //   experiments,
  //   isLoading,
  //   id,
  //   total,
  //   handleTableRowSelect,
  //   availableBatchActions,
  //   updateSettings,
  //   ExperimentTabOptions,
  //   project?.archived,
  //   project?.notes,
  //   handleDeleteNote,
  //   handleNewNotesPage,
  //   handleSaveNotes,
  // ]);

  if (isNaN(id)) {
    return <Message title={`Invalid Project ID ${projectId}`} />;
  } else if (!permissions.canViewWorkspaces) {
    return <NoPermissions />;
  } else if (pageError) {
    if (isNotFound(pageError)) return <PageNotFound />;
    const message = `Unable to fetch Project ${projectId}`;
    return <Message title={message} type={MessageType.Warning} />;
  } else if (!project) {
    return <Spinner tip={id === 1 ? 'Loading...' : `Loading project ${id} details...`} />;
  }
  return (
    <Page
      bodyNoPadding
      containerRef={pageRef}
      // for docTitle, when id is 1 that means Uncategorized from webui/react/src/routes/routes.ts
      docTitle={id === 1 ? 'Uncategorized Experiments' : 'Project Details'}
      id="projectDetails">
      <BreadcrumbBar
        extra={
          <Space>
            {project.description && (
              <Tooltip title={project.description}>
                <InfoCircleOutlined style={{ color: 'var(--theme-float-on)' }} />
              </Tooltip>
            )}
            {id !== 1 && (
              <ProjectActionDropdown
                curUser={user}
                project={project}
                showChildrenIfEmpty={false}
                trigger={['click']}
                workspaceArchived={workspace?.archived}
                onComplete={fetchProject}>
                <div style={{ cursor: 'pointer' }}>
                  <Icon name="arrow-down" size="tiny" />
                </div>
              </ProjectActionDropdown>
            )}
          </Space>
        }
        id={project.id}
        project={project}
        type="project"
      />
      <DynamicTabs
        basePath={paths.projectDetailsBasePath(id)}
        destroyInactiveTabPane
        tabBarStyle={{ height: 50, paddingLeft: 16 }}>
        <TabPane className={css.tabPane} key="experiments" tab={id === 1 ? '' : 'Experiments'}>
          <div className={css.base}>
            <ExperimentList project={project} />
          </div>
        </TabPane>
        {!project.immutable && projectId && (
          <>
            <TabPane className={css.tabPane} key="notes" tab="Notes">
              <div className={css.base}>
                <ProjectNotes fetchProject={fetchProject} project={project} />
              </div>
            </TabPane>
            <TabPane className={css.tabPane} key="trials" tab="Trials">
              <div className={css.base}>
                <TrialsComparison projectId={projectId} />
              </div>
            </TabPane>
          </>
        )}
      </DynamicTabs>
    </Page>
  );
};

export default ProjectDetails;
