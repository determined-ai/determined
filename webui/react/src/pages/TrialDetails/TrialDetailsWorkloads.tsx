// import { SelectValue } from 'antd/es/select';
import { FilterValue, SorterResult, TablePaginationConfig } from 'antd/es/table/interface';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import CheckpointModalTrigger from 'components/CheckpointModalTrigger';
import HumanReadableNumber from 'components/HumanReadableNumber';
import Select, { Option, SelectValue } from 'components/kit/Select';
import MetricBadgeTag from 'components/MetricBadgeTag';
import ResponsiveFilters from 'components/ResponsiveFilters';
import Section from 'components/Section';
import ResponsiveTable from 'components/Table/ResponsiveTable';
import { defaultRowClassName, getFullPaginationConfig } from 'components/Table/Table';
import usePolling from 'hooks/usePolling';
import { getTrialWorkloads } from 'services/api';
import {
  ExperimentBase,
  Metric,
  Step,
  TrialDetails,
  TrialWorkloadFilter,
  WorkloadGroup,
} from 'types';
import { isEqual } from 'utils/data';
import { ErrorType } from 'utils/error';
import handleError from 'utils/error';
import {
  extractMetricSortValue,
  extractMetricValue,
  metricKeyToMetric,
  metricToKey,
} from 'utils/metric';
import { numericSorter } from 'utils/sort';
import { hasCheckpoint, hasCheckpointStep, workloadsToSteps } from 'utils/workload';

import { Settings } from './TrialDetailsOverview.settings';
import { columns as defaultColumns } from './TrialDetailsWorkloads.table';

export interface Props {
  defaultMetrics: Metric[];
  experiment: ExperimentBase;
  metricNames: Metric[];
  metrics: Metric[];
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
  }, [defaultMetrics, metrics, settings.filter]);

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

    const metricRenderer = (metricName: Metric) => {
      const metricCol = (_: string, record: Step) => {
        const value = extractMetricValue(record, metricName);
        return <HumanReadableNumber num={value} />;
      };
      return metricCol;
    };

    const { metric, smallerIsBetter } = experiment?.config?.searcher || {};
    const newColumns = [...defaultColumns].map((column) => {
      if (column.key === 'checkpoint') column.render = checkpointRenderer;
      return column;
    });

    metrics.forEach((metricName) => {
      const stateIndex = newColumns.findIndex((column) => column.key === 'state');
      newColumns.splice(stateIndex, 0, {
        defaultSortOrder:
          metric && metric === metricName.name
            ? smallerIsBetter
              ? 'ascend'
              : 'descend'
            : undefined,
        key: metricToKey(metricName),
        render: metricRenderer(metricName),
        sorter: (a, b) => {
          const aVal = extractMetricSortValue(a, metricName),
            bVal = extractMetricSortValue(b, metricName);
          if (aVal === undefined && bVal !== undefined) {
            return settings.sortDesc ? -1 : 1;
          } else if (aVal !== undefined && bVal === undefined) {
            return settings.sortDesc ? 1 : -1;
          }
          return numericSorter(aVal, bVal);
        },
        title: <MetricBadgeTag metric={metricName} />,
      });
    });

    return newColumns.map((column) => {
      column.sortOrder = null;
      if (column.key === settings.sortKey) {
        column.sortOrder = settings.sortDesc ? 'descend' : 'ascend';
      }
      return column;
    });
  }, [metrics, settings, trial, experiment]);

  const [workloads, setWorkloads] = useState<WorkloadGroup[]>([]);
  const [workloadCount, setWorkloadCount] = useState<number>(0);

  const fetchWorkloads = useCallback(async () => {
    try {
      if (trial?.id) {
        const wl = await getTrialWorkloads({
          filter: settings.filter,
          id: trial.id,
          limit: settings.tableLimit,
          metricType: metricKeyToMetric(settings.sortKey)?.type || undefined,
          offset: settings.tableOffset,
          orderBy: settings.sortDesc ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC',
          sortKey: metricKeyToMetric(settings.sortKey)?.name || undefined,
        });
        setWorkloads(wl.workloads);
        setWorkloadCount(wl.pagination.total || 0);
      } else {
        setWorkloads([]);
        setWorkloadCount(0);
      }
    } catch (e) {
      handleError(e, {
        publicMessage: 'Failed to load recent trial workloads.',
        publicSubject: 'Unable to fetch Trial Workloads.',
        silent: false,
        type: ErrorType.Api,
      });
    }
  }, [
    trial?.id,
    settings.sortDesc,
    settings.sortKey,
    settings.tableLimit,
    settings.tableOffset,
    settings.filter,
  ]);

  const { stopPolling } = usePolling(fetchWorkloads, { rerunOnNewFn: true });

  const workloadSteps = useMemo(() => {
    const data = workloads ?? [];
    const workloadSteps = workloadsToSteps(data);
    return settings.filter === TrialWorkloadFilter.All
      ? workloadSteps
      : workloadSteps.filter((wlStep) => {
          if (settings.filter === TrialWorkloadFilter.Checkpoint) {
            return hasCheckpoint(wlStep);
          } else if (settings.filter === TrialWorkloadFilter.Validation) {
            return !!wlStep.validation;
          } else if (settings.filter === TrialWorkloadFilter.CheckpointOrValidation) {
            return !!wlStep.checkpoint || !!wlStep.validation;
          }
          return false;
        });
  }, [settings.filter, workloads]);

  const handleHasCheckpointOrValidationSelect = useCallback(
    (value: SelectValue): void => {
      const newFilter = value as TrialWorkloadFilter;
      const isValidFilter = Object.values(TrialWorkloadFilter).includes(newFilter);
      const filter = isValidFilter ? newFilter : undefined;
      updateSettings({ filter, tableOffset: 0 });
    },
    [updateSettings],
  );

  const handleTableChange = useCallback(
    (
      tablePagination: TablePaginationConfig,
      tableFilters: Record<string, FilterValue | null>,
      tableSorter: SorterResult<Step> | SorterResult<Step>[],
    ) => {
      if (Array.isArray(tableSorter)) return;

      const { columnKey, order } = tableSorter as SorterResult<Step>;
      if (!columnKey || !columns.find((column) => column.key === columnKey)) return;

      updateSettings({
        sortDesc: order === 'descend',
        sortKey: columnKey as string,
        tableLimit: tablePagination.pageSize,
        tableOffset: ((tablePagination.current ?? 1) - 1) * (tablePagination.pageSize ?? 0),
      });
    },
    [columns, updateSettings],
  );

  const options = (
    <ResponsiveFilters hasFiltersApplied={hasFiltersApplied}>
      <Select label="Show" value={settings.filter} onSelect={handleHasCheckpointOrValidationSelect}>
        {Object.values(TrialWorkloadFilter).map((key) => (
          <Option key={key} value={key}>
            {key}
          </Option>
        ))}
      </Select>
    </ResponsiveFilters>
  );

  useEffect(() => {
    return () => {
      stopPolling();
    };
  }, [stopPolling]);

  return (
    <>
      <Section options={options} title="Workloads">
        <ResponsiveTable<Step>
          columns={columns}
          dataSource={workloadSteps}
          loading={!trial}
          pagination={getFullPaginationConfig(
            {
              limit: settings.tableLimit,
              offset: settings.tableOffset,
            },
            workloadCount,
          )}
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
