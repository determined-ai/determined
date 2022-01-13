import { Button, Dropdown, Menu, Modal } from 'antd';
import { ColumnsType } from 'antd/lib/table';
import { FilterDropdownProps, SorterResult } from 'antd/lib/table/interface';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Icon from 'components/Icon';
import InlineEditor from 'components/InlineEditor';
import Link from 'components/Link';
import Page from 'components/Page';
import ResponsiveTable from 'components/ResponsiveTable';
import tableCss from 'components/ResponsiveTable.module.scss';
import Section from 'components/Section';
import { checkmarkRenderer, getFullPaginationConfig, modelNameRenderer,
  relativeTimeRenderer, userRenderer } from 'components/Table';
import TableFilterDropdown from 'components/TableFilterDropdown';
import TableFilterSearch from 'components/TableFilterSearch';
import TagList from 'components/TagList';
import { useStore } from 'contexts/Store';
import useCreateModelModal from 'hooks/useCreateModelModal';
import { useFetchUsers } from 'hooks/useFetch';
import usePolling from 'hooks/usePolling';
import useSettings from 'hooks/useSettings';
import { paths } from 'routes/utils';
import { archiveModel, deleteModel, getModelLabels,
  getModels, patchModel, unarchiveModel } from 'services/api';
import { V1GetModelsRequestSortBy } from 'services/api-ts-sdk';
import { validateDetApiEnum } from 'services/utils';
import { ArchiveFilter, ModelItem } from 'types';
import { isBoolean, isEqual } from 'utils/data';
import handleError, { ErrorType } from 'utils/error';
import { alphaNumericSorter } from 'utils/sort';
import { capitalize } from 'utils/string';

import css from './ModelRegistry.module.scss';
import settingsConfig, { Settings } from './ModelRegistry.settings';

const ModelRegistry: React.FC = () => {
  const { users, auth: { user } } = useStore();
  const [ models, setModels ] = useState<ModelItem[]>([]);
  const [ tags, setTags ] = useState<string[]>([]);
  const [ isLoading, setIsLoading ] = useState(true);
  const [ canceler ] = useState(new AbortController());
  const [ total, setTotal ] = useState(0);
  const { showModal } = useCreateModelModal();

  const {
    settings,
    updateSettings,
  } = useSettings<Settings>(settingsConfig);

  const fetchUsers = useFetchUsers(canceler);

  const fetchModels = useCallback(async () => {
    try {
      const response = await getModels({
        archived: settings.archived,
        description: settings.description,
        labels: settings.tags,
        limit: settings.tableLimit,
        name: settings.name,
        offset: settings.tableOffset,
        orderBy: settings.sortDesc ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC',
        sortBy: validateDetApiEnum(V1GetModelsRequestSortBy, settings.sortKey),
        users: settings.users,
      }, { signal: canceler.signal });
      setTotal(response.pagination.total || 0);
      setModels(prev => {
        if (isEqual(prev, response.models)) return prev;
        return response.models;
      });
      setIsLoading(false);
    } catch(e) {
      handleError(e, {
        publicSubject: 'Unable to fetch models.',
        silent: true,
        type: ErrorType.Api,
      });
      setIsLoading(false);
    }
  }, [ settings, canceler.signal ]);

  const fetchTags = useCallback(async () => {
    try {
      const tags = await getModelLabels({ signal: canceler.signal });
      tags.sort((a, b) => alphaNumericSorter(a, b));
      setTags(tags);
    } catch (e) {}
  }, [ canceler.signal ]);

  const fetchAll = useCallback(() => {
    fetchModels();
    fetchTags();
    fetchUsers();
  }, [ fetchModels, fetchTags, fetchUsers ]);

  usePolling(fetchAll);

  /*
   * Get new models based on changes to the
   * pagination and sorter.
   */
  useEffect(() => {
    setIsLoading(true);
    fetchModels();
  }, [
    fetchModels,
    settings,
  ]);

  const deleteCurrentModel = useCallback((model: ModelItem) => {
    deleteModel({ modelId: model.id });
    fetchModels();
  }, [ fetchModels ]);

  const switchArchived = useCallback(async (model: ModelItem) => {
    try {
      setIsLoading(true);
      if (model.archived) {
        await unarchiveModel({ modelId: model.id });
      } else {
        await archiveModel({ modelId: model.id });
      }
      await fetchModels();
    } catch (e) {
      handleError(e, {
        publicSubject: `Unable to switch model ${model.id} archive status.`,
        silent: true,
        type: ErrorType.Api,
      });
      setIsLoading(false);
    }
  }, [ fetchModels ]);

  const setModelTags = useCallback(async (modelId, tags) => {
    try {
      setIsLoading(true);
      await patchModel({ body: { id: modelId, labels: tags }, modelId });
      await fetchModels();
    } catch (e) {
      handleError(e, {
        publicSubject: `Unable to update model ${modelId} tags.`,
        silent: true,
        type: ErrorType.Api,
      });
      setIsLoading(false);
    }
  }, [ fetchModels ]);

  const handleArchiveFilterApply = useCallback((archived: string[]) => {
    const archivedFilter = archived.length === 1
      ? archived[0] === ArchiveFilter.Archived : undefined;
    updateSettings({ archived: archivedFilter });
  }, [ updateSettings ]);

  const handleArchiveFilterReset = useCallback(() => {
    updateSettings({ archived: undefined });
  }, [ updateSettings ]);

  const archiveFilterDropdown = useCallback((filterProps: FilterDropdownProps) => (
    <TableFilterDropdown
      {...filterProps}
      values={isBoolean(settings.archived)
        ? [ settings.archived ? ArchiveFilter.Archived : ArchiveFilter.Unarchived ]
        : undefined}
      onFilter={handleArchiveFilterApply}
      onReset={handleArchiveFilterReset}
    />
  ), [ handleArchiveFilterApply, handleArchiveFilterReset, settings.archived ]);

  const handleUserFilterApply = useCallback((users: string[]) => {
    updateSettings({ users: users.length !== 0 ? users : undefined });
  }, [ updateSettings ]);

  const handleUserFilterReset = useCallback(() => {
    updateSettings({ users: undefined });
  }, [ updateSettings ]);

  const userFilterDropdown = useCallback((filterProps: FilterDropdownProps) => (
    <TableFilterDropdown
      {...filterProps}
      multiple
      searchable
      values={settings.users}
      onFilter={handleUserFilterApply}
      onReset={handleUserFilterReset}
    />
  ), [ handleUserFilterApply, handleUserFilterReset, settings.users ]);

  const tableSearchIcon = useCallback(() => <Icon name="search" size="tiny" />, []);

  const handleNameSearchApply = useCallback((newSearch: string) => {
    updateSettings({ name: newSearch || undefined });
  }, [ updateSettings ]);

  const handleNameSearchReset = useCallback(() => {
    updateSettings({ name: undefined });
  }, [ updateSettings ]);

  const nameFilterSearch = useCallback((filterProps: FilterDropdownProps) => (
    <TableFilterSearch
      {...filterProps}
      value={settings.name || ''}
      onReset={handleNameSearchReset}
      onSearch={handleNameSearchApply}
    />
  ), [ handleNameSearchApply, handleNameSearchReset, settings.name ]);

  const handleDescriptionSearchApply = useCallback((newSearch: string) => {
    updateSettings({ description: newSearch || undefined });
  }, [ updateSettings ]);

  const handleDescriptionSearchReset = useCallback(() => {
    updateSettings({ description: undefined });
  }, [ updateSettings ]);

  const descriptionFilterSearch = useCallback((filterProps: FilterDropdownProps) => (
    <TableFilterSearch
      {...filterProps}
      value={settings.description || ''}
      onReset={handleDescriptionSearchReset}
      onSearch={handleDescriptionSearchApply}
    />
  ), [ handleDescriptionSearchApply, handleDescriptionSearchReset, settings.description ]);

  const handleLabelFilterApply = useCallback((tags: string[]) => {
    updateSettings({ tags: tags.length !== 0 ? tags : undefined });
  }, [ updateSettings ]);

  const handleLabelFilterReset = useCallback(() => {
    updateSettings({ tags: undefined });
  }, [ updateSettings ]);

  const labelFilterDropdown = useCallback((filterProps: FilterDropdownProps) => (
    <TableFilterDropdown
      {...filterProps}
      multiple
      searchable
      values={settings.tags}
      onFilter={handleLabelFilterApply}
      onReset={handleLabelFilterReset}
    />
  ), [ handleLabelFilterApply, handleLabelFilterReset, settings.tags ]);

  const showConfirmDelete = useCallback((model: ModelItem) => {
    Modal.confirm({
      closable: true,
      content: `Are you sure you want to delete this model "${model.name}" and all 
      of its versions from the model registry?`,
      icon: null,
      maskClosable: true,
      okText: 'Delete Model',
      okType: 'danger',
      onOk: () => deleteCurrentModel(model),
      title: 'Confirm Delete',
    });
  }, [ deleteCurrentModel ]);

  const saveModelDescription = useCallback(async (editedDescription: string, id: number) => {
    try {
      await patchModel({
        body: { description: editedDescription, id },
        modelId: id,
      });
    } catch (e) {
      handleError(e, {
        publicSubject: 'Unable to save model description.',
        silent: true,
        type: ErrorType.Api,
      });
      setIsLoading(false);
    }
  }, [ ]);

  const columns = useMemo(() => {
    const labelsRenderer = (value: string, record: ModelItem) => (
      <TagList
        compact
        disabled={record.archived}
        tags={record.labels ?? []}
        onChange={(tags) => setModelTags(record.id, tags)}
      />
    );

    const overflowRenderer = (_:string, record: ModelItem) => {
      const isDeletable = user?.isAdmin || user?.username === record.username;
      return (
        <Dropdown
          overlay={(
            <Menu>
              <Menu.Item
                key="switch-archived"
                onClick={() => switchArchived(record)}>
                {record.archived ? 'Unarchive' : 'Archive'}
              </Menu.Item>
              <Menu.Item
                danger
                disabled={!isDeletable}
                key="delete-model"
                onClick={() => showConfirmDelete(record)}>
                Delete Model
              </Menu.Item>
            </Menu>
          )}
          trigger={[ 'click' ]}>
          <Button className={css.overflow} type="text">
            <Icon name="overflow-vertical" size="tiny" />
          </Button>
        </Dropdown>
      );
    };

    const descriptionRenderer = (value:string, record: ModelItem) => (
      <InlineEditor
        disabled={record.archived}
        placeholder="Add description..."
        value={value}
        onSave={(newDescription: string) => saveModelDescription(newDescription, record.id)}
      />
    );

    const tableColumns: ColumnsType<ModelItem> = [
      {
        dataIndex: 'name',
        filterDropdown: nameFilterSearch,
        filterIcon: tableSearchIcon,
        key: V1GetModelsRequestSortBy.NAME,
        onHeaderCell: () => settings.name ? { className: tableCss.headerFilterOn } : {},
        render: modelNameRenderer,
        sorter: true,
        title: 'Name',
        width: 250,
      },
      {
        dataIndex: 'description',
        filterDropdown: descriptionFilterSearch,
        filterIcon: tableSearchIcon,
        key: V1GetModelsRequestSortBy.DESCRIPTION,
        onHeaderCell: () => settings.description ? { className: tableCss.headerFilterOn } : {},
        render: descriptionRenderer,
        sorter: true,
        title: 'Description',
      },
      {
        dataIndex: 'numVersions',
        key: V1GetModelsRequestSortBy.NUMVERSIONS,
        sorter: true,
        title: 'Versions',
        width: 100,
      },
      {
        dataIndex: 'lastUpdatedTime',
        key: V1GetModelsRequestSortBy.LASTUPDATEDTIME,
        render: (date) => relativeTimeRenderer(new Date(date)),
        sorter: true,
        title: 'Last updated',
        width: 150,
      },
      {
        dataIndex: 'labels',
        filterDropdown: labelFilterDropdown,
        filters: tags.map(tag => ({ text: tag, value: tag })),
        onHeaderCell: () => settings.tags ? { className: tableCss.headerFilterOn } : {},
        render: labelsRenderer,
        title: 'Tags',
        width: 120,
      },
      {
        dataIndex: 'archived',
        filterDropdown: archiveFilterDropdown,
        filters: [
          { text: capitalize(ArchiveFilter.Archived), value: ArchiveFilter.Archived },
          { text: capitalize(ArchiveFilter.Unarchived), value: ArchiveFilter.Unarchived },
        ],
        key: 'archived',
        onHeaderCell: () => settings.archived != null ? { className: tableCss.headerFilterOn } : {},
        render: checkmarkRenderer,
        title: 'Archived',
        width: 120,
      },
      {
        dataIndex: 'username',
        filterDropdown: userFilterDropdown,
        filters: users.map(user => ({ text: user.username, value: user.username })),
        onHeaderCell: () => settings.archived != null ? { className: tableCss.headerFilterOn } : {},
        render: userRenderer,
        title: 'User',
        width: 100,
      },
      { fixed: 'right', render: overflowRenderer, title: '', width: 40 },
    ];

    return tableColumns.map(column => {
      column.sortOrder = null;
      if (column.key === settings.sortKey) {
        column.sortOrder = settings.sortDesc ? 'descend' : 'ascend';
      }
      return column;
    });
  }, [ nameFilterSearch,
    tableSearchIcon,
    descriptionFilterSearch,
    labelFilterDropdown,
    archiveFilterDropdown,
    userFilterDropdown,
    users,
    setModelTags,
    user,
    switchArchived,
    showConfirmDelete,
    settings,
    tags,
    saveModelDescription ]);

  const handleTableChange = useCallback((tablePagination, tableFilters, tableSorter) => {
    if (Array.isArray(tableSorter)) return;

    const { columnKey, order } = tableSorter as SorterResult<ModelItem>;
    if (!columnKey || !columns.find(column => column.key === columnKey)) return;

    const newSettings = {
      sortDesc: order === 'descend',
      /* eslint-disable-next-line @typescript-eslint/no-explicit-any */
      sortKey: columnKey as any,
      tableLimit: tablePagination.pageSize,
      tableOffset: (tablePagination.current - 1) * tablePagination.pageSize,
    };
    const shouldPush = settings.tableOffset !== newSettings.tableOffset;
    updateSettings(newSettings, shouldPush);
  }, [ columns, settings.tableOffset, updateSettings ]);

  useEffect(() => {
    return () => canceler.abort();
  }, [ canceler ]);

  const showCreateModelModal = useCallback(() => {
    showModal({});
  }, [ showModal ]);

  return (
    <Page docTitle="Model Registry" id="models" loading={isLoading}>
      <Section
        options={<Button onClick={showCreateModelModal}>New Model</Button>}
        title="Model Registry">
        {(models.length === 0 && !isLoading) ?
          (
            <div className={css.emptyBase}>
              <div className={css.icon}>
                <Icon name="model" size="mega" />
              </div>
              <h4>No Models Registered</h4>
              <p className={css.description}>
                Track important checkpoints and versions from your experiments.&nbsp;
                <Link external path={paths.docs('/post-training/model-registry.html')}>
                  Learn more
                </Link>
              </p>
            </div>
          ) : (
            <ResponsiveTable
              columns={columns}
              dataSource={models}
              loading={isLoading}
              pagination={getFullPaginationConfig({
                limit: settings.tableLimit,
                offset: settings.tableOffset,
              }, total)}
              showSorterTooltip={false}
              size="small"
              onChange={handleTableChange}
            />
          )}
      </Section>
    </Page>
  );
};

export default ModelRegistry;
