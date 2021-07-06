import { Button, Tooltip } from 'antd';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import CheckpointModal from 'components/CheckpointModal';
import HumanReadableFloat from 'components/HumanReadableFloat';
import Icon from 'components/Icon';
import MetricBadgeTag from 'components/MetricBadgeTag';
import MetricSelectFilter from 'components/MetricSelectFilter';
import ResponsiveFilters from 'components/ResponsiveFilters';
import ResponsiveTable from 'components/ResponsiveTable';
import Section from 'components/Section';
import { defaultRowClassName, getPaginationConfig, MINIMUM_PAGE_SIZE } from 'components/Table';
import useStorage from 'hooks/useStorage';
import {
  CheckpointDetail, CheckpointState, ExperimentBase, MetricName, MetricType, Step, TrialDetails,
} from 'types';
import { isEqual } from 'utils/data';
import { numericSorter } from 'utils/sort';
import { hasCheckpointStep, workloadsToSteps } from 'utils/step';
import { extractMetricNames, extractMetricValue } from 'utils/trial';

import { columns as defaultColumns } from './TrialDetailsWorkloads.table';

const STORAGE_PATH = 'trial-detail';
const STORAGE_LIMIT_KEY = 'limit';
const STORAGE_TABLE_METRICS_KEY = 'metrics/table';

export interface Props {
  experiment: ExperimentBase;
  trial: TrialDetails;
}

const TrialDetailsWorkloads: React.FC<Props> = ({ experiment, trial }: Props) => {
  const [ activeCheckpoint, setActiveCheckpoint ] = useState<CheckpointDetail>();
  const [ defaultMetrics, setDefaultMetrics ] = useState<MetricName[]>([]);
  const [ metrics, setMetrics ] = useState<MetricName[]>([]);
  const [ showCheckpoint, setShowCheckpoint ] = useState(false);
  const storage = useStorage(STORAGE_PATH);
  const storageMetricsPath = experiment ? `experiments/${experiment.id}` : undefined;

  const initLimit = storage.getWithDefault(STORAGE_LIMIT_KEY, MINIMUM_PAGE_SIZE);
  const storageTableMetricsKey =
    storageMetricsPath && `${storageMetricsPath}/${STORAGE_TABLE_METRICS_KEY}`;

  const [ pageSize, setPageSize ] = useState(initLimit);

  const hasFiltersApplied = useMemo(() => {
    const metricsApplied = !isEqual(metrics, defaultMetrics);
    return metricsApplied;
  }, [ metrics, defaultMetrics ]);

  const metricNames = useMemo(() => extractMetricNames(
    trial?.workloads || [],
  ), [ trial?.workloads ]);

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
      if (column.key === 'checkpoint') {
        column.render = checkpointRenderer;
        column.filters = [ { text: 'Has checkpoint', value: 'checkpoint' } ];
        column.onFilter = (value, record) => record.checkpoint?.state === CheckpointState.Completed;
      }
      return column;
    });

    metrics.forEach(metricName => {
      const stateIndex = newColumns.findIndex(column => column.key === 'state');
      newColumns.splice(stateIndex, 0, {
        defaultSortOrder: metric && metric === metricName.name ?
          (smallerIsBetter ? 'ascend' : 'descend') : undefined,
        filters: [ { text: `Has ${metricName.name}`, value: metricName.name } ],
        onFilter: (value, record) => !!record.validation?.metrics?.[String(value)],
        render: metricRenderer(metricName),
        sorter: (a, b) => numericSorter(
          extractMetricValue(a, metricName),
          extractMetricValue(b, metricName),
        ),
        title: <MetricBadgeTag metric={metricName} />,
      });
    });

    return newColumns;
  }, [ experiment?.config, metrics, trial ]);

  const workloadSteps = useMemo(() => {
    const data = trial?.workloads || [];
    const workloadSteps = workloadsToSteps(data);
    return workloadSteps;
  }, [ trial?.workloads ]);

  // Default to selecting config search metric only.
  useEffect(() => {
    const searcherName = experiment.config?.searcher?.metric;
    const defaultMetric = metricNames.find(metricName => {
      return metricName.name === searcherName && metricName.type === MetricType.Validation;
    });
    const defaultMetrics = defaultMetric ? [ defaultMetric ] : [];
    setDefaultMetrics(defaultMetrics);
    const initMetrics = storage.getWithDefault(storageTableMetricsKey || '', defaultMetrics);
    setMetrics(initMetrics);
  }, [ experiment, metricNames, storage, storageTableMetricsKey ]);

  const handleCheckpointShow = (event: React.MouseEvent, checkpoint: CheckpointDetail) => {
    event.stopPropagation();
    setActiveCheckpoint(checkpoint);
    setShowCheckpoint(true);
  };
  const handleCheckpointDismiss = () => setShowCheckpoint(false);

  const handleMetricChange = useCallback((value: MetricName[]) => {
    setMetrics(value);
    if (storageTableMetricsKey) storage.set(storageTableMetricsKey, value);
  }, [ storage, storageTableMetricsKey ]);

  const handleTableChange = useCallback((tablePagination) => {
    storage.set(STORAGE_LIMIT_KEY, tablePagination.pageSize);
    setPageSize(tablePagination.pageSize);
  }, [ storage ]);

  const options = (
    <ResponsiveFilters hasFiltersApplied={hasFiltersApplied}>
      {metrics && <MetricSelectFilter
        defaultMetricNames={defaultMetrics}
        metricNames={metricNames}
        multiple
        value={metrics}
        onChange={handleMetricChange} />}
    </ResponsiveFilters>
  );

  return (
    <>
      <Section options={options}>
        <ResponsiveTable<Step>
          columns={columns}
          dataSource={workloadSteps}
          pagination={getPaginationConfig(workloadSteps.length, pageSize)}
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
