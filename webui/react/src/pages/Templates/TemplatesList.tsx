import {
  FilterDropdownProps,
  FilterValue,
  SorterResult,
  TablePaginationConfig,
} from 'antd/lib/table/interface';
import { Loadable } from 'hew/utils/loadable';
import { useObservable } from 'micro-observables';
import React, { useCallback, useMemo } from 'react';

import InteractiveTable, { ColumnDef } from 'components/Table/InteractiveTable';
import SkeletonTable from 'components/Table/SkeletonTable';
import {
  defaultRowClassName,
  getFullPaginationConfig,
  taskWorkspaceRenderer,
} from 'components/Table/Table';
import TableFilterDropdown from 'components/Table/TableFilterDropdown';
import WorkspaceFilter from 'components/WorkspaceFilter';
import { useSettings } from 'hooks/useSettings';
import { V1GetTemplatesRequestSortBy } from 'services/api-ts-sdk';
import workspaceStore from 'stores/workspaces';
import { Template } from 'types';
import { alphaNumericSorter } from 'utils/sort';

import settingsConfig, { DEFAULT_COLUMN_WIDTHS, Settings } from './TemplatesList.settings';

interface Props {
  templates: Template[];
  isLoading: boolean;
  pageRef: React.RefObject<HTMLElement>;
}

const TemplateList: React.FC<Props> = ({ templates, isLoading, pageRef }) => {
  const { settings, updateSettings } = useSettings<Settings>(settingsConfig);

  const workspaces = Loadable.getOrElse([], useObservable(workspaceStore.workspaces));

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

  const columns = useMemo(() => {
    return [
      {
        dataIndex: 'name',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['name'],
        key: 'name',
        sorter: (a: Template, b: Template): number => alphaNumericSorter(a.name, b.name),
        title: 'Name',
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
        key: V1GetTemplatesRequestSortBy.NAME,
        render: (_v: string, record: Template) => taskWorkspaceRenderer(record, workspaces),
        sorter: true,
        title: 'Workspace',
      },
    ] as ColumnDef<Template>[];
  }, [workspaceFilterDropdown, workspaces]);

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

  return settings ? (
    <InteractiveTable<Template, Settings>
      columns={columns}
      containerRef={pageRef}
      dataSource={templates}
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
  );
};

export default TemplateList;
