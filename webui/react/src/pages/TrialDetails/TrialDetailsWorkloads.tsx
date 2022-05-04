import { Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import { SorterResult } from 'antd/es/table/interface';
import React, { useCallback, useMemo } from 'react';

import CheckpointModalTrigger from 'components/CheckpointModalTrigger';
import HumanReadableNumber from 'components/HumanReadableNumber';
import MetricBadgeTag from 'components/MetricBadgeTag';
import ResponsiveFilters from 'components/ResponsiveFilters';
import ResponsiveTable from 'components/ResponsiveTable';
import Section from 'components/Section';
import SelectFilter from 'components/SelectFilter';
import { defaultRowClassName, getFullPaginationConfig } from 'components/Table';
import {
  CommandTask, ExperimentBase, MetricName,
  Step, TrialDetails,
} from 'types';
import { isEqual } from 'utils/data';
import { extractMetricValue } from 'utils/metric';
import { numericSorter } from 'utils/sort';
import { hasCheckpoint, hasCheckpointStep, workloadsToSteps } from 'utils/workload';

import { Settings, TrialWorkloadFilter } from './TrialDetailsOverview.settings';
import { columns as defaultColumns } from './TrialDetailsWorkloads.table';

const { Option } = Select;

export interface Props {
  defaultMetrics: MetricName[];
  experiment: ExperimentBase;
  metricNames: MetricName[];
  metrics: MetricName[];
  settings: Settings;
  trial?: TrialDetails;
  updateSettings: (newSettings: Partial<Settings>) => void;
}

const TrialDetailsWorkloads: React.FC<Props> = ({
  defaultMetrics,
  experiment,
  metrics,
  settings,
  trial,
  updateSettings,
}: Props) => {
  const hasFiltersApplied = useMemo(() => {
    const metricsApplied = !isEqual(metrics, defaultMetrics);
    const checkpointValidationFilterApplied = settings.filter !== TrialWorkloadFilter.All;
    return metricsApplied || checkpointValidationFilterApplied;
  }, [ defaultMetrics, metrics, settings.filter ]);

  const columns = useMemo(() => {
    const checkpointRenderer = (_: string, record: Step) => {
      if (trial && record.checkpoint && hasCheckpointStep(record)) {
        const checkpoint = {
          ...record.checkpoint,
          experimentId: trial.experimentId,
          trialId: trial.id,
        };
        return (
          <CheckpointModalTrigger
            checkpoint={checkpoint}
            experiment={experiment}
            title={`Checkpoint for Batch ${checkpoint.totalBatches}`}
          />
        );
      }
      return null;
    };

    const metricRenderer = (metricName: MetricName) => {
      const metricCol = (_: string, record: Step) => {
        const value = extractMetricValue(record, metricName);
        return <HumanReadableNumber num={value} />;
      };
      return metricCol;
    };

    const { metric, smallerIsBetter } = experiment?.config?.searcher || {};
    const newColumns = [ ...defaultColumns ].map(column => {
      if (column.key === 'checkpoint') column.render = checkpointRenderer;
      return column;
    });

    metrics.forEach(metricName => {
      const stateIndex = newColumns.findIndex(column => column.key === 'state');
      newColumns.splice(stateIndex, 0, {
        defaultSortOrder: metric && metric === metricName.name ?
          (smallerIsBetter ? 'ascend' : 'descend') : undefined,
        key: metricName.name,
        render: metricRenderer(metricName),
        sorter: (a, b) => numericSorter(
          extractMetricValue(a, metricName),
          extractMetricValue(b, metricName),
        ),
        title: <MetricBadgeTag metric={metricName} />,
      });
    });

    return newColumns.map(column => {
      column.sortOrder = null;
      if (column.key === settings.sortKey) {
        column.sortOrder = settings.sortDesc ? 'descend' : 'ascend';
      }
      return column;
    });
  }, [ metrics, settings, trial, experiment ]);

  const workloadSteps = useMemo(() => {
    const data = trial?.workloads || [];
    const workloadSteps = workloadsToSteps(data);
    return settings.filter === TrialWorkloadFilter.All
      ? workloadSteps
      : workloadSteps.filter(wlStep => {
        if (settings.filter === TrialWorkloadFilter.Checkpoint) {
          return hasCheckpoint(wlStep);
        } else if (settings.filter === TrialWorkloadFilter.Validation) {
          return !!wlStep.validation;
        } else if (settings.filter === TrialWorkloadFilter.CheckpointOrValidation) {
          return !!wlStep.checkpoint || !!wlStep.validation;
        }
        return false;
      });
  }, [ settings.filter, trial?.workloads ]);

  const handleHasCheckpointOrValidationSelect = useCallback((value: SelectValue): void => {
    const newFilter = value as TrialWorkloadFilter;
    const isValidFilter = Object.values(TrialWorkloadFilter).includes(newFilter);
    const filter = isValidFilter ? newFilter : undefined;
    updateSettings({ filter, tableOffset: 0 });
  }, [ updateSettings ]);

  const handleTableChange = useCallback((tablePagination, tableFilters, tableSorter) => {
    if (Array.isArray(tableSorter)) return;

    const { columnKey, order } = tableSorter as SorterResult<CommandTask>;
    if (!columnKey || !columns.find(column => column.key === columnKey)) return;

    updateSettings({
      sortDesc: order === 'descend',
      sortKey: columnKey as string,
      tableLimit: tablePagination.pageSize,
      tableOffset: (tablePagination.current - 1) * tablePagination.pageSize,
    });
  }, [ columns, updateSettings ]);

  const options = (
    <ResponsiveFilters hasFiltersApplied={hasFiltersApplied}>
      <SelectFilter
        dropdownMatchSelectWidth={300}
        label="Show"
        value={settings.filter}
        onSelect={handleHasCheckpointOrValidationSelect}>
        {Object.values(TrialWorkloadFilter).map(key => (
          <Option key={key} value={key}>{key}</Option>
        ))}
      </SelectFilter>
    </ResponsiveFilters>
  );

  return (
    <>
      <Section options={options} title="Workloads">
        <ResponsiveTable<Step>
          columns={columns}
          dataSource={workloadSteps}
          pagination={getFullPaginationConfig({
            limit: settings.tableLimit,
            offset: settings.tableOffset,
          }, workloadSteps.length)}
          rowClassName={defaultRowClassName({ clickable: false })}
          rowKey="batchNum"
          scroll={{ x: 1000 }}
          showSorterTooltip={false}
          size="small"
          onChange={handleTableChange}
        />
      </Section>
    </>
  );
};

export default TrialDetailsWorkloads;
