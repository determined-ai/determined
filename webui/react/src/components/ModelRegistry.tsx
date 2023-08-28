import { Space, Typography } from 'antd';
import {
  FilterDropdownProps,
  FilterValue,
  SorterResult,
  TablePaginationConfig,
} from 'antd/lib/table/interface';
import _ from 'lodash';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import DeleteModelModal from 'components/DeleteModelModal';
import DynamicIcon from 'components/DynamicIcon';
import FilterCounter from 'components/FilterCounter';
import Button from 'components/kit/Button';
import Dropdown, { MenuItem } from 'components/kit/Dropdown';
import Empty from 'components/kit/Empty';
import Icon from 'components/kit/Icon';
import Input from 'components/kit/Input';
import { useModal } from 'components/kit/Modal';
import Tags, { tagsActionHelper } from 'components/kit/Tags';
import Toggle from 'components/kit/Toggle';
import Tooltip from 'components/kit/Tooltip';
import Link from 'components/Link';
import ModelCreateModal from 'components/ModelCreateModal';
import ModelMoveModal from 'components/ModelMoveModal';
import InteractiveTable, {
  ColumnDef,
  onRightClickableCell,
} from 'components/Table/InteractiveTable';
import {
  checkmarkRenderer,
  defaultRowClassName,
  getFullPaginationConfig,
  modelNameRenderer,
  relativeTimeRenderer,
  userRenderer,
} from 'components/Table/Table';
import TableFilterDropdown from 'components/Table/TableFilterDropdown';
import TableFilterSearch from 'components/Table/TableFilterSearch';
import WorkspaceFilter from 'components/WorkspaceFilter';
import usePermissions from 'hooks/usePermissions';
import usePolling from 'hooks/usePolling';
import { useSettings } from 'hooks/useSettings';
import { paths } from 'routes/utils';
import { archiveModel, getModelLabels, getModels, patchModel, unarchiveModel } from 'services/api';
import { V1GetModelsRequestSortBy } from 'services/api-ts-sdk';
import userStore from 'stores/users';
import workspaceStore from 'stores/workspaces';
import { ModelItem, Workspace } from 'types';
import handleError, { ErrorType } from 'utils/error';
import { Loadable } from 'utils/loadable';
import { useObservable } from 'utils/observable';
import { validateDetApiEnum } from 'utils/service';
import { alphaNumericSorter } from 'utils/sort';
import { getDisplayName } from 'utils/user';

import css from './ModelRegistry.module.scss';
import settingsConfig, {
  DEFAULT_COLUMN_WIDTHS,
  isOfSortKey,
  ModelColumnName,
  Settings,
} from './ModelRegistry.settings';

const filterKeys: Array<keyof Settings> = ['tags', 'name', 'users', 'description', 'workspace'];

interface Props {
  workspace?: Workspace;
}

const MenuKey = {
  DeleteModel: 'delete-model',
  MoveModel: 'move-model',
  SwitchArchived: 'switch-archived',
} as const;

const ModelRegistry: React.FC<Props> = ({ workspace }: Props) => {
  const canceler = useRef(new AbortController());
  const users = Loadable.getOrElse([], useObservable(userStore.getUsers()));
  const [models, setModels] = useState<ModelItem[]>([]);
  const [model, setModel] = useState<ModelItem | null>(null);
  const [tags, setTags] = useState<string[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const pageRef = useRef<HTMLElement>(null);
  const { canCreateModels, canDeleteModel, canModifyModel } = usePermissions();
  const workspaces = Loadable.getOrElse([], useObservable(workspaceStore.workspaces));

  const modelCreateModal = useModal(ModelCreateModal);
  const deleteModelModal = useModal(DeleteModelModal);
  const modelMoveModal = useModal(ModelMoveModal);

  const settingConfig = useMemo(() => {
    return settingsConfig(workspace?.id.toString() ?? 'global');
  }, [workspace?.id]);
  const {
    activeSettings,
    isLoading: isLoadingSettings,
    settings,
    updateSettings,
    resetSettings,
  } = useSettings<Settings>(settingConfig);

  const filterCount = useMemo(() => activeSettings(filterKeys).length, [activeSettings]);
  const isTableLoading = useMemo(
    () => isLoading || isLoadingSettings,
    [isLoading, isLoadingSettings],
  );

  const fetchModels = useCallback(async () => {
    if (!settings) return;

    try {
      const response = await getModels(
        {
          archived: settings.archived ? undefined : false,
          description: settings.description,
          labels: settings.tags,
          limit: settings.tableLimit,
          name: settings.name,
          offset: settings.tableOffset,
          orderBy: settings.sortDesc ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC',
          sortBy: validateDetApiEnum(V1GetModelsRequestSortBy, settings.sortKey),
          users: settings.users,
          workspaceIds: workspace?.id ? [workspace.id] : settings.workspace,
        },
        { signal: canceler.current.signal },
      );
      setTotal(response.pagination.total || 0);
      setModels((prev) => {
        response.models.map((m) => m.labels?.sort?.());
        if (_.isEqual(prev, response.models)) return prev;
        return response.models;
      });
    } catch (e) {
      handleError(e, {
        publicSubject: 'Unable to fetch models.',
        silent: true,
        type: ErrorType.Api,
      });
    } finally {
      setIsLoading(false);
    }
  }, [settings, workspace?.id]);

  const fetchTags = useCallback(async () => {
    try {
      const tags = await getModelLabels(
        { workspaceId: workspace?.id },
        { signal: canceler.current.signal },
      );
      tags.sort((a, b) => alphaNumericSorter(a, b));
      setTags(tags);
    } catch (e) {
      handleError(e);
    }
  }, [workspace?.id]);

  const fetchAll = useCallback(async () => {
    await Promise.allSettled([fetchModels(), fetchTags()]);
  }, [fetchModels, fetchTags]);

  usePolling(fetchAll, { rerunOnNewFn: true });

  /**
   * Get new models based on changes to the pagination and sorter.
   */
  useEffect(() => {
    setIsLoading(true);
    fetchModels();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const switchArchived = useCallback(
    async (model: ModelItem) => {
      try {
        setIsLoading(true);
        if (model.archived) {
          await unarchiveModel({ modelName: model.name });
        } else {
          await archiveModel({ modelName: model.name });
        }
        await fetchModels();
      } catch (e) {
        handleError(e, {
          publicSubject: `Unable to switch model ${model.id} archive status.`,
          silent: true,
          type: ErrorType.Api,
        });
      } finally {
        setIsLoading(false);
      }
    },
    [fetchModels],
  );

  const setModelTags = useCallback(
    async (modelName: string, tags: string[]) => {
      try {
        await patchModel({ body: { labels: tags, name: modelName }, modelName });
        await fetchModels();
      } catch (e) {
        handleError(e, {
          publicSubject: `Unable to update model ${modelName} tags.`,
          silent: true,
          type: ErrorType.Api,
        });
      }
    },
    [fetchModels],
  );

  const handleUserFilterApply = useCallback(
    (users: string[]) => {
      updateSettings({ users: users.length !== 0 ? users : undefined });
    },
    [updateSettings],
  );

  const handleUserFilterReset = useCallback(() => {
    updateSettings({ users: undefined });
  }, [updateSettings]);

  const userFilterDropdown = useCallback(
    (filterProps: FilterDropdownProps) => (
      <TableFilterDropdown
        {...filterProps}
        multiple
        searchable
        values={settings.users}
        onFilter={handleUserFilterApply}
        onReset={handleUserFilterReset}
      />
    ),
    [handleUserFilterApply, handleUserFilterReset, settings.users],
  );

  const tableSearchIcon = useCallback(() => <Icon name="search" size="tiny" title="Search" />, []);

  const handleNameSearchApply = useCallback(
    (newSearch: string) => {
      updateSettings({ name: newSearch || undefined });
    },
    [updateSettings],
  );

  const handleNameSearchReset = useCallback(() => {
    updateSettings({ name: undefined });
  }, [updateSettings]);

  const nameFilterSearch = useCallback(
    (filterProps: FilterDropdownProps) => (
      <TableFilterSearch
        {...filterProps}
        value={settings.name || ''}
        onReset={handleNameSearchReset}
        onSearch={handleNameSearchApply}
      />
    ),
    [handleNameSearchApply, handleNameSearchReset, settings.name],
  );

  const handleDescriptionSearchApply = useCallback(
    (newSearch: string) => {
      updateSettings({ description: newSearch || undefined });
    },
    [updateSettings],
  );

  const handleDescriptionSearchReset = useCallback(() => {
    updateSettings({ description: undefined });
  }, [updateSettings]);

  const descriptionFilterSearch = useCallback(
    (filterProps: FilterDropdownProps) => (
      <TableFilterSearch
        {...filterProps}
        value={settings.description || ''}
        onReset={handleDescriptionSearchReset}
        onSearch={handleDescriptionSearchApply}
      />
    ),
    [handleDescriptionSearchApply, handleDescriptionSearchReset, settings.description],
  );

  const handleLabelFilterApply = useCallback(
    (tags: string[]) => {
      updateSettings({ tags: tags.length !== 0 ? tags : undefined });
    },
    [updateSettings],
  );

  const handleLabelFilterReset = useCallback(() => {
    updateSettings({ tags: undefined });
  }, [updateSettings]);

  const handleWorkspaceFilterApply = useCallback(
    (workspaces: string[]) => {
      updateSettings({
        row: undefined,
        workspace:
          workspaces.length !== 0 ? workspaces.map((workspace) => Number(workspace)) : undefined,
      });
    },
    [updateSettings],
  );

  const handleWorkspaceFilterReset = useCallback(() => {
    updateSettings({ row: undefined, workspace: undefined });
  }, [updateSettings]);

  const workspaceFilterDropdown = useCallback(
    (filterProps: FilterDropdownProps) => (
      <TableFilterDropdown
        {...filterProps}
        multiple
        values={settings.workspace?.map((ws) => ws.toString())}
        width={220}
        onFilter={handleWorkspaceFilterApply}
        onReset={handleWorkspaceFilterReset}
      />
    ),
    [handleWorkspaceFilterApply, handleWorkspaceFilterReset, settings.workspace],
  );

  const workspaceRenderer = useCallback(
    (record: ModelItem): React.ReactNode => {
      const workspace = workspaces.find((u) => u.id === record.workspaceId);
      if (!workspace) {
        return (
          <Link disabled>
            <DynamicIcon name="-" size={24} />
          </Link>
        );
      }
      const workspaceId = record.workspaceId;
      return (
        <Tooltip content={workspace?.name} placement="top">
          <div className={`${css.centerVertically} ${css.centerHorizontally}`}>
            <Link
              path={
                workspaceId === 1
                  ? paths.projectDetails(workspaceId)
                  : paths.workspaceDetails(workspaceId)
              }>
              <DynamicIcon name={workspace?.name} size={24} />
            </Link>
          </div>
        </Tooltip>
      );
    },
    [workspaces],
  );

  const labelFilterDropdown = useCallback(
    (filterProps: FilterDropdownProps) => (
      <TableFilterDropdown
        {...filterProps}
        multiple
        searchable
        values={settings.tags}
        onFilter={handleLabelFilterApply}
        onReset={handleLabelFilterReset}
      />
    ),
    [handleLabelFilterApply, handleLabelFilterReset, settings.tags],
  );

  const saveModelDescription = useCallback(async (modelName: string, editedDescription: string) => {
    try {
      await patchModel({
        body: { description: editedDescription, name: modelName },
        modelName,
      });
    } catch (e) {
      handleError(e, {
        publicSubject: 'Unable to save model description.',
        silent: false,
        type: ErrorType.Api,
      });
    }
  }, []);

  const resetFilters = useCallback(() => {
    resetSettings([...filterKeys, 'tableOffset']);
  }, [resetSettings]);

  const ModelActionMenu = useCallback(
    (record: ModelItem) => {
      const canDeleteModelFlag = canDeleteModel({ model: record });
      const canModifyModelFlag = canModifyModel({ model: record });
      const menuItems: MenuItem[] = [];
      if (canModifyModelFlag) {
        menuItems.push({
          key: MenuKey.SwitchArchived,
          label: record.archived ? 'Unarchive' : 'Archive',
        });
        if (!record.archived) {
          menuItems.push({ key: MenuKey.MoveModel, label: 'Move' });
        }
      }
      if (canDeleteModelFlag) {
        menuItems.push({ danger: true, key: MenuKey.DeleteModel, label: 'Delete Model' });
      }

      return menuItems;
    },
    [], // canDeleteModel, canModifyModel
  );

  const handleDropdown = useCallback(
    (key: string, record: ModelItem) => {
      switch (key) {
        case MenuKey.DeleteModel:
          setModel(record);
          deleteModelModal.open();
          break;
        case MenuKey.MoveModel:
          setModel(record);
          modelMoveModal.open();
          break;
        case MenuKey.SwitchArchived:
          switchArchived(record);
          break;
      }
    },
    [deleteModelModal, modelMoveModal, switchArchived],
  );

  const columns = useMemo(() => {
    const tagsRenderer = (value: string, record: ModelItem) => (
      <div className={css.tagsRenderer}>
        <Typography.Text
          ellipsis={{
            tooltip: <Tags disabled tags={record.labels ?? []} />,
          }}>
          <div>
            <Tags
              compact
              disabled={record.archived || !canModifyModel({ model: record })}
              tags={record.labels ?? []}
              onAction={tagsActionHelper(record.labels ?? [], (tags) =>
                setModelTags(record.name, tags),
              )}
            />
          </div>
        </Typography.Text>
      </div>
    );

    const actionRenderer = (_: string, record: ModelItem) => {
      const canDeleteModelFlag = canDeleteModel({ model: record });
      const canModifyModelFlag = canModifyModel({ model: record });
      return (
        <Dropdown
          disabled={!canDeleteModelFlag && !canModifyModelFlag}
          menu={ModelActionMenu(record)}
          placement="bottomRight"
          onClick={(key) => handleDropdown(key, record)}>
          <Button icon={<Icon name="overflow-vertical" title="Action menu" />} type="text" />
        </Dropdown>
      );
    };

    const descriptionRenderer = (value: string, record: ModelItem) => (
      <Input
        className={css.descriptionRenderer}
        defaultValue={value}
        disabled={record.archived || !canModifyModel({ model: record })}
        placeholder={record.archived ? 'Archived' : 'Add description...'}
        title={record.archived ? 'Archived description' : 'Edit description'}
        onBlur={(e) => {
          const newDesc = e.currentTarget.value;
          saveModelDescription(record.name, newDesc);
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
        filterDropdown: nameFilterSearch,
        filterIcon: tableSearchIcon,
        isFiltered: (settings: Settings) => !!settings.name,
        key: V1GetModelsRequestSortBy.NAME,
        onCell: onRightClickableCell,
        render: modelNameRenderer,
        sorter: true,
        title: 'Name',
      },
      {
        dataIndex: 'description',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['description'],
        filterDropdown: descriptionFilterSearch,
        filterIcon: tableSearchIcon,
        isFiltered: (settings: Settings) => !!settings.description,
        key: V1GetModelsRequestSortBy.DESCRIPTION,
        render: descriptionRenderer,
        sorter: true,
        title: 'Description',
      },
      {
        align: 'center',
        dataIndex: 'workspace',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['workspace'],
        filterDropdown: workspaceFilterDropdown,
        filters: workspaces.map((ws) => ({
          text: <WorkspaceFilter workspace={ws} />,
          value: ws.id,
        })),
        isFiltered: (settings: Settings) => !!settings.workspace,
        key: V1GetModelsRequestSortBy.WORKSPACE,
        render: (v: string, record: ModelItem) => workspaceRenderer(record),
        sorter: true,
        title: 'Workspace',
      },
      {
        align: 'right',
        dataIndex: 'numVersions',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['numVersions'],
        key: V1GetModelsRequestSortBy.NUMVERSIONS,
        onCell: onRightClickableCell,
        sorter: true,
        title: 'Versions',
      },
      {
        align: 'right',
        dataIndex: 'lastUpdatedTime',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['lastUpdatedTime'],
        key: V1GetModelsRequestSortBy.LASTUPDATEDTIME,
        render: (date: string) => relativeTimeRenderer(new Date(date)),
        sorter: true,
        title: 'Last updated',
      },
      {
        dataIndex: 'tags',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['tags'],
        filterDropdown: labelFilterDropdown,
        filters: tags.map((tag) => ({ text: tag, value: tag })),
        isFiltered: (settings: Settings) => !!settings.tags,
        key: 'tags',
        render: tagsRenderer,
        title: 'Tags',
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
        dataIndex: 'user',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['user'],
        filterDropdown: userFilterDropdown,
        filters: users.map((user) => ({ text: getDisplayName(user), value: user.id })),
        isFiltered: (settings: Settings) => !!settings.users,
        key: 'user',
        render: (_, r) => userRenderer(users.find((u) => u.id === r.userId)),
        title: 'User',
      },
      {
        align: 'right',
        className: 'fullCell',
        dataIndex: 'action',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['action'],
        fixed: 'right',
        key: 'action',
        render: actionRenderer,
        title: '',
        width: DEFAULT_COLUMN_WIDTHS['action'],
      },
    ] as ColumnDef<ModelItem>[];
  }, [
    users,
    nameFilterSearch,
    tableSearchIcon,
    descriptionFilterSearch,
    workspaceFilterDropdown,
    workspaces,
    labelFilterDropdown,
    tags,
    userFilterDropdown,
    // canModifyModel,
    setModelTags,
    // canDeleteModel,
    ModelActionMenu,
    // handleDropdown,
    saveModelDescription,
    workspaceRenderer,
  ]);

  const handleTableChange = useCallback(
    (
      tablePagination: TablePaginationConfig,
      tableFilters: Record<string, FilterValue | null>,
      tableSorter: SorterResult<ModelItem> | SorterResult<ModelItem>[],
    ) => {
      if (Array.isArray(tableSorter)) return;

      const { columnKey, order } = tableSorter as SorterResult<ModelItem>;
      if (!columnKey || !columns.find((column) => column.key === columnKey)) return;

      const newSettings = {
        sortDesc: order === 'descend',
        sortKey: isOfSortKey(columnKey) ? columnKey : V1GetModelsRequestSortBy.UNSPECIFIED,
        tableLimit: tablePagination.pageSize,
        tableOffset: ((tablePagination.current ?? 1) - 1) * (tablePagination.pageSize ?? 0),
      };
      updateSettings(newSettings);
    },
    [columns, updateSettings],
  );

  useEffect(() => {
    const currentCanceler = canceler.current;
    return () => currentCanceler.abort();
  }, []);

  const showCreateModelModal = useCallback(() => modelCreateModal.open(), [modelCreateModal]);

  const switchShowArchived = useCallback(
    (showArchived: boolean) => {
      let newColumns: ModelColumnName[];
      let newColumnWidths: number[];
      const settingsColumns = settings.columns ?? [];
      const settingsColumnsWidths = settings.columnWidths ?? [];

      if (showArchived) {
        if (settings.columns?.includes('archived')) {
          // just some defensive coding: don't add archived twice
          newColumns = settings.columns;
          newColumnWidths = settings.columnWidths;
        } else {
          newColumns = [...settingsColumns, 'archived'];
          newColumnWidths = [...settingsColumnsWidths, DEFAULT_COLUMN_WIDTHS['archived']];
        }
      } else {
        const archivedIndex = settings.columns.indexOf('archived') ?? 0;
        if (archivedIndex !== -1) {
          newColumns = [...settingsColumns];
          newColumnWidths = [...settingsColumnsWidths];
          newColumns.splice(archivedIndex, 1);
          newColumnWidths.splice(archivedIndex, 1);
        } else {
          newColumns = settingsColumns;
          newColumnWidths = settingsColumnsWidths;
        }
      }
      updateSettings({
        archived: showArchived,
        columns: newColumns,
        columnWidths: newColumnWidths,
        row: undefined,
      });
    },
    [settings, updateSettings],
  );

  const ModelActionDropdown = useCallback(
    ({ record, children }: { children: React.ReactNode; record: ModelItem }) => {
      return (
        <Dropdown
          isContextMenu
          menu={ModelActionMenu(record)}
          onClick={(key) => handleDropdown(key, record)}>
          {children}
        </Dropdown>
      );
    },
    [ModelActionMenu], // handleDropdown
  );

  return (
    <>
      <div className={css.options}>
        <Space>
          <Toggle checked={settings.archived} label="Show Archived" onChange={switchShowArchived} />
          {filterCount > 0 && (
            <FilterCounter activeFilterCount={filterCount} onReset={resetFilters} />
          )}
          {canCreateModels ? (
            <Button onClick={showCreateModelModal}>New Model</Button>
          ) : (
            <Tooltip content="User lacks permission to create models" placement="leftBottom">
              <div>
                <Button disabled onClick={showCreateModelModal}>
                  New Model
                </Button>
              </div>
            </Tooltip>
          )}
        </Space>
      </div>
      {models.length === 0 && !isLoading && filterCount === 0 ? (
        <Empty
          description={
            <>
              Track important checkpoints and versions from your experiments.{' '}
              <Link external path={paths.docs('/post-training/model-registry.html')}>
                Learn more
              </Link>
            </>
          }
          icon="model"
          title="No Models Registered"
        />
      ) : (
        <InteractiveTable<ModelItem, Settings>
          columns={columns}
          containerRef={pageRef}
          ContextMenu={ModelActionDropdown}
          dataSource={models}
          loading={isTableLoading}
          pagination={getFullPaginationConfig(
            {
              limit: settings.tableLimit,
              offset: settings.tableOffset,
            },
            total,
          )}
          rowClassName={defaultRowClassName({ clickable: false })}
          rowKey="name"
          settings={settings}
          showSorterTooltip={false}
          size="small"
          updateSettings={updateSettings}
          onChange={handleTableChange}
        />
      )}
      <modelCreateModal.Component workspaceId={workspace?.id} />
      {model && <deleteModelModal.Component model={model} />}
      {model && <modelMoveModal.Component model={model} />}
    </>
  );
};

export default ModelRegistry;
