import { Button, Select, Tooltip } from 'antd';
import { SelectValue } from 'antd/es/select';
import { SorterResult } from 'antd/es/table/interface';
import React, { useCallback, useMemo, useState } from 'react';

import CheckpointModal from 'components/CheckpointModal';
import HumanReadableFloat from 'components/HumanReadableFloat';
import Icon from 'components/Icon';
import MetricBadgeTag from 'components/MetricBadgeTag';
import ResponsiveFilters from 'components/ResponsiveFilters';
import ResponsiveTable from 'components/ResponsiveTable';
import Section from 'components/Section';
import SelectFilter, { ALL_VALUE } from 'components/SelectFilter';
import { defaultRowClassName, getFullPaginationConfig } from 'components/Table';
import useStorage from 'hooks/useStorage';
import { ApiSorter } from 'services/types';
import {
  CheckpointDetail, CommandTask, ExperimentBase, MetricName,
  Pagination,
  Step, TrialDetails,
} from 'types';
import { isEqual } from 'utils/data';
import { numericSorter } from 'utils/sort';
import { extractMetricValue } from 'utils/trial';
import { hasCheckpoint, hasCheckpointStep, workloadsToSteps } from 'utils/workload';

import { TrialInfoFilter } from './TrialDetailsOverview';
import { columns as defaultColumns } from './TrialDetailsWorkloads.table';

const { Option } = Select;

const STORAGE_PATH = 'trial-detail';
const STORAGE_LIMIT_KEY = 'limit';
const STORAGE_CHECKPOINT_VALIDATION_KEY = 'checkpoint-validation';
const STORAGE_SORTER_KEY = 'sorter';

export interface Props {
  defaultMetrics: MetricName[];
  experiment: ExperimentBase;
  handleMetricChange: (value: MetricName[]) => void;
  metricNames: MetricName[];
  metrics: MetricName[];
  pagination: Pagination;
  setPagination: (pagination: Pagination) => void;
  setShowFilter: (showFilter: TrialInfoFilter) => void;
  setSorter: (sorter: ApiSorter) => void;
  showFilter: TrialInfoFilter;
  sorter: ApiSorter;
  trial: TrialDetails;
}

const TrialDetailsWorkloads: React.FC<Props> = (
  {
    defaultMetrics, experiment, pagination, setPagination,
    setShowFilter, setSorter, showFilter, sorter, trial, metrics,
  }: Props,
) => {
  const [ activeCheckpoint, setActiveCheckpoint ] = useState<CheckpointDetail>();
  const [ showCheckpoint, setShowCheckpoint ] = useState(false);
  const storage = useStorage(STORAGE_PATH);

  const hasFiltersApplied = useMemo(() => {
    const metricsApplied = !isEqual(metrics, defaultMetrics);
    const checkpointValidationFilterApplied = showFilter as string !== ALL_VALUE;
    return metricsApplied || checkpointValidationFilterApplied;
  }, [ defaultMetrics, showFilter, metrics ]);

  const columns = useMemo(() => {
    const checkpointRenderer = (_: string, record: Step) => {
      if (trial && record.checkpoint && hasCheckpointStep(record)) {
        const checkpoint = {
          ...record.checkpoint,
          batch: record.checkpoint.totalBatches,
          experimentId: trial?.experimentId,
          trialId: trial?.id,
        };
        return (
          <Tooltip title="View Checkpoint">
            <Button
              aria-label="View Checkpoint"
              icon={<Icon name="checkpoint" />}
              onClick={e => handleCheckpointShow(e, checkpoint)} />
          </Tooltip>
        );
      }
      return null;
    };

    const metricRenderer = (metricName: MetricName) => {
      const metricCol = (_: string, record: Step) => {
        const value = extractMetricValue(record, metricName);
        return value != null ? <HumanReadableFloat num={value} /> : undefined;
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
      if (column.key === sorter.key) column.sortOrder = sorter.descend ? 'descend' : 'ascend';
      return column;
    });
  }, [ experiment?.config, metrics, sorter, trial ]);

  const workloadSteps = useMemo(() => {
    const data = trial?.workloads || [];
    const workloadSteps = workloadsToSteps(data);
    return showFilter as string === ALL_VALUE ?
      workloadSteps : workloadSteps.filter(wlStep => {
        if (showFilter === TrialInfoFilter.Checkpoint) {
          return hasCheckpoint(wlStep);
        } else if (showFilter === TrialInfoFilter.Validation) {
          return !!wlStep.validation;
        } else if (showFilter === TrialInfoFilter.CheckpointOrValidation) {
          return !!wlStep.checkpoint || !!wlStep.validation;
        }
        return false;
      });
  }, [ showFilter, trial?.workloads ]);

  const handleCheckpointShow = (event: React.MouseEvent, checkpoint: CheckpointDetail) => {
    event.stopPropagation();
    setActiveCheckpoint(checkpoint);
    setShowCheckpoint(true);
  };
  const handleCheckpointDismiss = () => setShowCheckpoint(false);

  const handleHasCheckpointOrValidationSelect = useCallback((value: SelectValue): void => {
    const filter = value as unknown as TrialInfoFilter;
    if (value as string !== ALL_VALUE && !Object.values(TrialInfoFilter).includes(filter)) return;
    setShowFilter(filter);
    storage.set(STORAGE_CHECKPOINT_VALIDATION_KEY, filter);
  }, [ setShowFilter, storage ]);

  const handleTableChange = useCallback((tablePagination, tableFilters, sorter) => {
    if (Array.isArray(sorter)) return;

    const { columnKey, order } = sorter as SorterResult<CommandTask>;
    if (!columnKey || !columns.find(column => column.key === columnKey)) return;

    const updatedSorter = { descend: order === 'descend', key: columnKey as string };
    storage.set(STORAGE_SORTER_KEY, updatedSorter);
    setSorter(updatedSorter);

    storage.set(STORAGE_LIMIT_KEY, tablePagination.pageSize);
    setPagination(
      ({
        limit: tablePagination.pageSize,
        offset: (tablePagination.current - 1) * tablePagination.pageSize,
      }),
    );
  }, [ columns, setPagination, setSorter, storage ]);

  const options = (
    <ResponsiveFilters hasFiltersApplied={hasFiltersApplied}>
      <SelectFilter
        dropdownMatchSelectWidth={300}
        label="Show"
        value={showFilter}
        onSelect={handleHasCheckpointOrValidationSelect}>
        <Option key={ALL_VALUE} value={ALL_VALUE}>All</Option>
        {Object.values(TrialInfoFilter).map(key => <Option key={key} value={key}>{key}</Option>)}
      </SelectFilter>
    </ResponsiveFilters>
  );

  return (
    <>
      <Section options={options} title="Workloads">
        <ResponsiveTable<Step>
          columns={columns}
          dataSource={workloadSteps}
          pagination={getFullPaginationConfig(pagination, workloadSteps.length)}
          rowClassName={defaultRowClassName({ clickable: false })}
          rowKey="batchNum"
          scroll={{ x: 1000 }}
          showSorterTooltip={false}
          size="small"
          onChange={handleTableChange} />
      </Section>
      {activeCheckpoint && experiment?.config && (
        <CheckpointModal
          checkpoint={activeCheckpoint}
          config={experiment?.config}
          show={showCheckpoint}
          title={`Checkpoint for Batch ${activeCheckpoint.batch}`}
          onHide={handleCheckpointDismiss} />
      )}
    </>
  );
};

export default TrialDetailsWorkloads;
