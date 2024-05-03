import {
  FilterDropdownProps,
  FilterValue,
  SorterResult,
  TablePaginationConfig,
} from 'antd/lib/table/interface';
import Button from 'hew/Button';
import Dropdown from 'hew/Dropdown';
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
import { alphaNumericSorter } from 'utils/sort';

import TemplateCreateModalComponent from './TemplateCreateModal';
import settingsConfig, { DEFAULT_COLUMN_WIDTHS, Settings } from './TemplatesList.settings';
import TemplateViewModalComponent from './TemplateViewModal';

interface Props {
  workspaceId?: number;
}

const MenuKey = {
  ViewTemplate: 'view-template',
} as const;

const DROPDOWN_MENU = [{ key: MenuKey.ViewTemplate, label: 'View Template' }];

const TemplateList: React.FC<Props> = ({ workspaceId }) => {
  const { settings, updateSettings } = useSettings<Settings>(
    settingsConfig(workspaceId ? workspaceId.toString() : 'global'),
  );
  const [selectedTemplate, setSelectedTemplate] = useState<Template>();
  const [templates, setTemplates] = useState<Template[]>([]);
  const [total, setTotal] = useState(0);
  const [isLoading, setIsLoading] = useState(true);
  const [canceler] = useState(new AbortController());
  const pageRef = useRef<HTMLElement>(null);
  const { canCreateTemplate, canCreateTemplateWorkspace } = usePermissions();
  const TemplateCreateModal = useModal(TemplateCreateModalComponent);

  const TemplateViewModal = useModal(TemplateViewModalComponent);

  const workspaces = Loadable.getOrElse([], useObservable(workspaceStore.workspaces));

  const fetchTemplates = useCallback(async () => {
    try {
      const res = await getTaskTemplates(
        {
          name: settings.name,
          orderBy: settings.sortDesc ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC',
          sortBy: validateDetApiEnum(V1GetTemplatesRequestSortBy, settings.sortKey),
        },
        { signal: canceler.signal },
      );
      setTotal(res.length);
      setTemplates((prev) => {
        let tmpls = res;
        if (workspaceId) tmpls = res.filter((t) => t.workspaceId === workspaceId);
        else if (settings.workspace?.length)
          tmpls = res.filter((t) => settings.workspace?.includes(t.workspaceId));
        if (_.isEqual(prev, tmpls)) return prev;
        return tmpls;
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
      switch (key) {
        case MenuKey.ViewTemplate:
          setSelectedTemplate(record);
          TemplateViewModal.open();
          break;
      }
    },
    [TemplateViewModal],
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
    const actionRenderer = (_: string, record: Template) => (
      <Dropdown menu={DROPDOWN_MENU} onClick={(key) => handleDropdown(key, record)}>
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
        key: 'name',
        sorter: (a: Template, b: Template): number => alphaNumericSorter(a.name, b.name),
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
        isFiltered: (settings: Settings) => !!settings.workspace,
        key: V1GetTemplatesRequestSortBy.NAME,
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
  }, [workspaceFilterDropdown, workspaces, handleDropdown, workspaceId, nameFilterSearch]);

  const handleTableChange = useCallback(
    (
      tablePagination: TablePaginationConfig,
      _tableFilters: Record<string, FilterValue | null>,
      tableSorter: SorterResult<Template> | SorterResult<Template>[],
    ) => {
      if (Array.isArray(tableSorter)) return;

      const { columnKey, order } = tableSorter as SorterResult<Template>;
      if (!columnKey || !columns.find((column) => column.key === columnKey)) return;

      const newSettings = {
        sortDesc: order === 'descend',
        sortKey: columnKey === 'name' ? columnKey : V1GetTemplatesRequestSortBy.UNSPECIFIED,
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

  return (
    <>
      <div className={css.headerButton}>
        {canCreate && <Button onClick={TemplateCreateModal.open}>New Template</Button>}
      </div>
      {(workspaceId ? templates.length === 0 : total === 0) && !isLoading ? (
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
      <TemplateCreateModal.Component workspaceId={workspaceId} onSuccess={fetchTemplates} />
      <TemplateViewModal.Component template={selectedTemplate} workspaces={workspaces} />
    </>
  );
};

export default TemplateList;
