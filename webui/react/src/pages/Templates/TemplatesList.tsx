import {
  FilterDropdownProps,
  FilterValue,
  SorterResult,
  TablePaginationConfig,
} from 'antd/lib/table/interface';
import Button from 'hew/Button';
import Dropdown, { MenuItem } from 'hew/Dropdown';
import Icon from 'hew/Icon';
import Message from 'hew/Message';
import { useModal } from 'hew/Modal';
import { Loadable } from 'hew/utils/loadable';
import _ from 'lodash';
import { useObservable } from 'micro-observables';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import InteractiveTable, { ColumnDef } from 'components/Table/InteractiveTable';
import SkeletonTable from 'components/Table/SkeletonTable';
import {
  defaultRowClassName,
  getFullPaginationConfig,
  taskWorkspaceRenderer,
} from 'components/Table/Table';
import TableFilterDropdown from 'components/Table/TableFilterDropdown';
import TableFilterSearch from 'components/Table/TableFilterSearch';
import WorkspaceFilter from 'components/WorkspaceFilter';
import usePermissions from 'hooks/usePermissions';
import { useSettings } from 'hooks/useSettings';
import css from 'pages/WorkspaceDetails/WorkspaceMembers.module.scss';
import { getTaskTemplates } from 'services/api';
import { V1GetTemplatesRequestSortBy } from 'services/api-ts-sdk';
import workspaceStore from 'stores/workspaces';
import { Template } from 'types';
import handleError, { ErrorType } from 'utils/error';
import { validateDetApiEnum } from 'utils/service';

import TemplateCreateModalComponent from './TemplateCreateModal';
import TemplateDeleteModalComponent from './TemplateDeleteModal';
import settingsConfig, { DEFAULT_COLUMN_WIDTHS, Settings } from './TemplatesList.settings';
import TemplateViewModalComponent from './TemplateViewModal';

interface Props {
  workspaceId?: number;
}

const MenuKey = {
  DeleteTemplate: 'delete-template',
  EditTemplate: 'edit-template',
  ViewTemplate: 'view-template',
} as const;

const TemplateList: React.FC<Props> = ({ workspaceId }) => {
  const { settings, updateSettings } = useSettings<Settings>(
    settingsConfig(workspaceId ? workspaceId.toString() : 'global'),
  );
  const [selectedTemplate, setSelectedTemplate] = useState<Template>();
  const [templates, setTemplates] = useState<Template[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [canceler] = useState(new AbortController());
  const pageRef = useRef<HTMLElement>(null);
  const { canCreateTemplate, canCreateTemplateWorkspace, canDeleteTemplate, canModifyTemplate } =
    usePermissions();

  const TemplateCreateModal = useModal(TemplateCreateModalComponent);
  const TemplateViewModal = useModal(TemplateViewModalComponent);
  const TemplateDeleteModal = useModal(TemplateDeleteModalComponent);

  const workspaces = Loadable.getOrElse([], useObservable(workspaceStore.workspaces));

  const fetchTemplates = useCallback(async () => {
    try {
      const res = await getTaskTemplates(
        {
          name: settings.name,
          orderBy: settings.sortDesc ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC',
          sortBy: validateDetApiEnum(V1GetTemplatesRequestSortBy, settings.sortKey),
          workspaceIds: workspaceId
            ? [workspaceId]
            : settings.workspace?.length
              ? settings.workspace
              : undefined,
        },
        { signal: canceler.signal },
      );
      setTemplates((prev) => {
        if (_.isEqual(prev, res)) return prev;
        return res;
      });
    } catch (e) {
      handleError(e, {
        publicSubject: 'Unable to fetch templates.',
        silent: true,
        type: ErrorType.Api,
      });
    } finally {
      setIsLoading(false);
    }
  }, [canceler.signal, settings, workspaceId]);

  useEffect(() => {
    setIsLoading(true);
    fetchTemplates();
  }, [fetchTemplates]);

  useEffect(() => {
    return () => canceler.abort();
  }, [canceler]);

  const handleWorkspaceFilterApply = useCallback(
    (workspaces: string[]) => {
      updateSettings({
        tableOffset: 0,
        workspace:
          workspaces.length !== 0 ? workspaces.map((workspace) => Number(workspace)) : undefined,
      });
    },
    [updateSettings],
  );

  const handleWorkspaceFilterReset = useCallback(() => {
    updateSettings({ workspace: undefined });
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

  const handleDropdown = useCallback(
    (key: string, record: Template) => {
      setSelectedTemplate(record);
      switch (key) {
        case MenuKey.ViewTemplate:
          TemplateViewModal.open();
          break;
        case MenuKey.EditTemplate:
          TemplateCreateModal.open();
          break;
        case MenuKey.DeleteTemplate:
          TemplateDeleteModal.open();
          break;
      }
    },
    [TemplateViewModal, TemplateCreateModal, TemplateDeleteModal],
  );

  const handleNameSearchReset = useCallback(() => {
    updateSettings({ name: undefined });
  }, [updateSettings]);

  const handleNameSearchApply = useCallback(
    (newSearch: string) => {
      updateSettings({ name: newSearch || undefined, tableOffset: 0 });
    },
    [updateSettings],
  );

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

  const columns = useMemo(() => {
    const actionMenu = (record: Template) => {
      const menu: MenuItem[] = [{ key: MenuKey.ViewTemplate, label: 'View Template' }];
      if (canModifyTemplate({ template: record })) {
        menu.push({ key: MenuKey.EditTemplate, label: 'Edit Template' });
      }
      if (canDeleteTemplate({ template: record })) {
        menu.push({ danger: true, key: MenuKey.DeleteTemplate, label: 'Delete Template' });
      }
      return menu;
    };

    const actionRenderer = (_: string, record: Template) => (
      <Dropdown menu={actionMenu(record)} onClick={(key) => handleDropdown(key, record)}>
        <Button icon={<Icon name="overflow-vertical" title="Action menu" />} type="text" />
      </Dropdown>
    );

    return [
      {
        dataIndex: 'name',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['name'],
        filterDropdown: nameFilterSearch,
        filterIcon: <Icon name="search" size="tiny" title="Search" />,
        isFiltered: (settings: Settings) => !!settings.name,
        key: V1GetTemplatesRequestSortBy.NAME,
        sorter: true,
        title: 'Name',
      },
      {
        align: 'center',
        dataIndex: 'workspace',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['workspace'],
        filterDropdown: workspaceId ? undefined : workspaceFilterDropdown,
        filters: workspaceId
          ? undefined
          : workspaces.map((ws) => ({
              text: <WorkspaceFilter workspace={ws} />,
              value: ws.id,
            })),
        isFiltered: (settings: Settings) => !!settings.workspace?.length,
        key: 'workspace',
        render: (_v: string, record: Template) => taskWorkspaceRenderer(record, workspaces),
        title: 'Workspace',
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
    ] as ColumnDef<Template>[];
  }, [
    workspaceFilterDropdown,
    workspaces,
    handleDropdown,
    workspaceId,
    nameFilterSearch,
    canDeleteTemplate,
    canModifyTemplate,
  ]);

  const handleTableChange = useCallback(
    (
      tablePagination: TablePaginationConfig,
      _tableFilters: Record<string, FilterValue | null>,
      tableSorter: SorterResult<Template> | SorterResult<Template>[],
    ) => {
      if (Array.isArray(tableSorter)) return;

      const { columnKey, order, field } = tableSorter as SorterResult<Template>;
      if (!columnKey || !columns.find((column) => column.key === columnKey)) return;

      const newSettings = {
        sortDesc: order === 'descend',
        sortKey:
          field === 'name'
            ? V1GetTemplatesRequestSortBy.NAME
            : V1GetTemplatesRequestSortBy.UNSPECIFIED,
        tableLimit: tablePagination.pageSize,
        tableOffset: ((tablePagination.current ?? 1) - 1) * (tablePagination.pageSize ?? 0),
      };
      updateSettings(newSettings);
    },
    [columns, updateSettings],
  );

  const canCreate = useMemo(() => {
    return workspaceId
      ? canCreateTemplateWorkspace({ workspace: { id: workspaceId } })
      : canCreateTemplate;
  }, [workspaceId, canCreateTemplate, canCreateTemplateWorkspace]);

  const onClickCreate = useCallback(() => {
    setSelectedTemplate(undefined);
    TemplateCreateModal.open();
  }, [TemplateCreateModal]);

  return (
    <>
      <div className={css.headerButton}>
        {canCreate && <Button onClick={onClickCreate}>New Template</Button>}
      </div>
      {!(settings.name || settings.workspace?.length) && templates.length === 0 && !isLoading ? (
        <Message
          description="Move settings that are shared by many tasks into a single YAML file, that can then be referenced by configurations that require those settings."
          icon="columns"
          title="No Template Configured"
        />
      ) : settings ? (
        <InteractiveTable<Template, Settings>
          columns={columns}
          containerRef={pageRef}
          dataSource={templates}
          interactiveColumns={false}
          loading={isLoading}
          pagination={getFullPaginationConfig(
            {
              limit: settings.tableLimit,
              offset: settings.tableOffset,
            },
            templates.length,
          )}
          rowClassName={defaultRowClassName({ clickable: false })}
          rowKey="id"
          settings={settings}
          showSorterTooltip={false}
          size="small"
          updateSettings={updateSettings}
          onChange={handleTableChange}
        />
      ) : (
        <SkeletonTable columns={columns.length} />
      )}
      <TemplateCreateModal.Component
        template={selectedTemplate}
        workspaceId={workspaceId}
        onSuccess={fetchTemplates}
      />
      <TemplateViewModal.Component template={selectedTemplate} workspaces={workspaces} />
      <TemplateDeleteModal.Component template={selectedTemplate!} onSuccess={fetchTemplates} />
    </>
  );
};

export default TemplateList;
