import { Button, Select, Tooltip } from 'antd';
import { SelectValue } from 'antd/es/select';
import { SorterResult } from 'antd/es/table/interface';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import CheckpointModal from 'components/CheckpointModal';
import HumanReadableFloat from 'components/HumanReadableFloat';
import Icon from 'components/Icon';
import MetricBadgeTag from 'components/MetricBadgeTag';
import MetricSelectFilter from 'components/MetricSelectFilter';
import ResponsiveFilters from 'components/ResponsiveFilters';
import ResponsiveTable from 'components/ResponsiveTable';
import Section from 'components/Section';
import SelectFilter, { ALL_VALUE } from 'components/SelectFilter';
import { defaultRowClassName, getFullPaginationConfig, MINIMUM_PAGE_SIZE } from 'components/Table';
import useStorage from 'hooks/useStorage';
import { parseUrl } from 'routes/utils';
import { ApiSorter } from 'services/types';
import {
  CheckpointDetail, CommandTask, ExperimentBase, MetricName,
  MetricType, Pagination, Step, TrialDetails,
} from 'types';
import { isEqual } from 'utils/data';
import { numericSorter } from 'utils/sort';
import { hasCheckpoint, hasCheckpointStep, workloadsToSteps } from 'utils/step';
import { extractMetricNames, extractMetricValue } from 'utils/trial';

import { columns as defaultColumns } from './TrialDetailsWorkloads.table';

const { Option } = Select;

enum TrialInfoFilter {
  Checkpoint = 'Has Checkpoint',
  Validation = 'Has Validation',
  CheckpointOrValidation = 'Has Checkpoint or Validation',
}

const defaultSorter: ApiSorter = {
  descend: true,
  key: 'batches',
};

const URL_ALL = 'all';

const STORAGE_PATH = 'trial-detail';
const STORAGE_LIMIT_KEY = 'limit';
const STORAGE_CHECKPOINT_VALIDATION_KEY = 'checkpoint-validation';
const STORAGE_TABLE_METRICS_KEY = 'metrics/table';
const STORAGE_SORTER_KEY = 'sorter';

export interface Props {
  experiment: ExperimentBase;
  trial: TrialDetails;
}

const TrialDetailsWorkloads: React.FC<Props> = ({ experiment, trial }: Props) => {
  const [ activeCheckpoint, setActiveCheckpoint ] = useState<CheckpointDetail>();
  const [ defaultMetrics, setDefaultMetrics ] = useState<MetricName[]>([]);
  const [ metrics, setMetrics ] = useState<MetricName[]>([]);
  const [ showCheckpoint, setShowCheckpoint ] = useState(false);
  const [ isUrlParsed, setIsUrlParsed ] = useState(false);
  const storage = useStorage(STORAGE_PATH);
  const storageMetricsPath = experiment ? `experiments/${experiment.id}` : undefined;

  const initFilter = storage.getWithDefault(
    STORAGE_CHECKPOINT_VALIDATION_KEY,
    TrialInfoFilter.CheckpointOrValidation,
  );
  const initLimit = storage.getWithDefault(STORAGE_LIMIT_KEY, MINIMUM_PAGE_SIZE);
  const initSorter = storage.getWithDefault(STORAGE_SORTER_KEY, { ...defaultSorter });
  const [ sorter, setSorter ] = useState<ApiSorter>(initSorter);
  const storageTableMetricsKey =
    storageMetricsPath && `${storageMetricsPath}/${STORAGE_TABLE_METRICS_KEY}`;
  const [ pagination, setPagination ] = useState<Pagination>(
    { limit: initLimit, offset: 0 },
  );
  const [ showFilter, setShowFilter ] = useState(initFilter);

  /*
   * When filters changes update the page URL.
   */
  useEffect(() => {
    if (!isUrlParsed) return;

    const searchParams = new URLSearchParams;
    const url = parseUrl(window.location.href);

    // limit
    searchParams.append('limit', pagination.limit.toString());

    // offset
    searchParams.append('offset', pagination.offset.toString());

    // sortDesc
    searchParams.append('sortDesc', sorter.descend ? '1' : '0');

    // sortKey
    searchParams.append('sortKey', sorter.key || '');

    // filters
    searchParams.append('filter', showFilter ? showFilter : URL_ALL);

    // metrics
    if (metrics && metrics.length > 0) {
      metrics.forEach(metric => searchParams.append('metric', metric.name + '|' + metric.type));
    } else {
      searchParams.append('state', URL_ALL);
    }

    window.history.pushState(
      {},
      '',
      url.origin + url.pathname + '?' + searchParams.toString(),
    );
  }, [ isUrlParsed, metrics, pagination, showFilter, sorter ]);

  /*
   * On first load: if filters are specified in URL, override default.
   */
  useEffect(() => {
    if (isUrlParsed) return;

    // If search params are not set, we default to user preferences
    const url = parseUrl(window.location.href);
    if (url.search === '') {
      setIsUrlParsed(true);
      return;
    }

    const urlSearchParams = url.searchParams;

    // limit
    const limit = urlSearchParams.get('limit');
    if (limit != null && !isNaN(parseInt(limit))) {
      pagination.limit = parseInt(limit);
    }

    // offset
    const offset = urlSearchParams.get('offset');
    if (offset != null && !isNaN(parseInt(offset))) {
      pagination.offset = parseInt(offset);
    }

    // sortDesc
    const sortDesc = urlSearchParams.get('sortDesc');
    if (sortDesc != null) {
      sorter.descend = (sortDesc === '1');
    }

    // filter
    const filter = urlSearchParams.get('filter');
    if (filter != null) {
      setShowFilter(filter as TrialInfoFilter);
    }

    // metrics
    const visibleMetrics = urlSearchParams.getAll('metric');
    let metrics = defaultMetrics;
    if (visibleMetrics != null) {
      metrics = (visibleMetrics.map(metric => {
        const splitMetric = metric.split('|');
        return { name: splitMetric[0], type: splitMetric[1] as MetricType };
      }));
    }

    // sortKey
    const sortKey = urlSearchParams.get('sortKey');
    if (sortKey != null &&
    [ 'batches', 'state', ...metrics.map(metric => metric.name) ].includes(sortKey)) {
      sorter.key = sortKey;
    }

    setIsUrlParsed(true);
    setPagination(pagination);
    setMetrics(metrics);
    setSorter(sorter);
  }, [ defaultMetrics, isUrlParsed, metrics, pagination, showFilter, sorter ]);

  const hasFiltersApplied = useMemo(() => {
    const metricsApplied = !isEqual(metrics, defaultMetrics);
    const checkpointValidationFilterApplied = showFilter as string !== ALL_VALUE;
    return metricsApplied || checkpointValidationFilterApplied;
  }, [ showFilter, metrics, defaultMetrics ]);

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

  const handleHasCheckpointOrValidationSelect = useCallback((value: SelectValue): void => {
    const filter = value as unknown as TrialInfoFilter;
    if (value as string !== ALL_VALUE && !Object.values(TrialInfoFilter).includes(filter)) return;
    setShowFilter(filter);
    storage.set(STORAGE_CHECKPOINT_VALIDATION_KEY, filter);
  }, [ setShowFilter, storage ]);

  const handleMetricChange = useCallback((value: MetricName[]) => {
    setMetrics(value);
    if (storageTableMetricsKey) storage.set(storageTableMetricsKey, value);
  }, [ storage, storageTableMetricsKey ]);

  const handleTableChange = useCallback((tablePagination, tableFilters, sorter) => {
    if (Array.isArray(sorter)) return;

    const { columnKey, order } = sorter as SorterResult<CommandTask>;
    if (!columnKey || !columns.find(column => column.key === columnKey)) return;

    const updatedSorter = { descend: order === 'descend', key: columnKey as string };
    storage.set(STORAGE_SORTER_KEY, updatedSorter);
    setSorter(updatedSorter);

    storage.set(STORAGE_LIMIT_KEY, tablePagination.pageSize);
    setPagination(pagination => {
      return {
        ...pagination,
        limit: tablePagination.pageSize,
        offset: (tablePagination.current - 1) * tablePagination.pageSize,
      };
    });
  }, [ columns, storage ]);

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
