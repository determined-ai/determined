import { Alert, Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import BadgeTag from 'components/BadgeTag';
import HumanReadableFloat from 'components/HumanReadableFloat';
import LearningCurveChart from 'components/LearningCurveChart';
import Link from 'components/Link';
import Message, { MessageType } from 'components/Message';
import MetricSelectFilter from 'components/MetricSelectFilter';
import ResponsiveFilters from 'components/ResponsiveFilters';
import ResponsiveTable from 'components/ResponsiveTable';
import Section from 'components/Section';
import SelectFilter from 'components/SelectFilter';
import Spinner from 'components/Spinner';
import { defaultRowClassName, getPaginationConfig, MINIMUM_PAGE_SIZE } from 'components/Table';
import { handlePath } from 'routes/utils';
import { V1TrialsSampleResponse } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { consumeStream } from 'services/utils';
import { ExperimentBase, MetricName, metricTypeParamMap, RunState } from 'types';
import { glasbeyColor } from 'utils/color';
import { alphanumericSorter, hpSorter, numericSorter } from 'utils/data';
import { terminalRunStates } from 'utils/types';

import css from './LearningCurve.module.scss';

const { Option } = Select;

interface Props {
  experiment: ExperimentBase;
  metrics: MetricName[];
  onMetricChange?: (metric: MetricName) => void;
  selectedMetric: MetricName
}

type HParams = Record<string, boolean | number | string>;

interface TrialHParams {
  hparams: HParams;
  id: number;
  metric: number | null;
  url: string;
}

const DEFAULT_MAX_TRIALS = 100;
const MAX_DATAPOINTS = 5000;
const MAX_ALLOWED_METRIC_VALUE = 100000;
const TOP_TRIALS_OPTIONS = [ 1, 10, 20, 50, 100 ];

const LearningCurve: React.FC<Props> = ({
  experiment,
  metrics,
  onMetricChange,
  selectedMetric,
}: Props) => {
  const [ trialIds, setTrialIds ] = useState<number[]>([]);
  const [ batches, setBatches ] = useState<number[]>([]);
  const [ chartData, setChartData ] = useState<(number | null)[][]>([]);
  const [ trialHps, setTrialHps ] = useState<TrialHParams[]>([]);
  const [ pageSize, setPageSize ] = useState(MINIMUM_PAGE_SIZE);
  const [ chartTrialId, setChartTrialId ] = useState<number>();
  const [ tableTrialId, setTableTrialId ] = useState<number>();
  const [ maxTrials, setMaxTrials ] = useState(DEFAULT_MAX_TRIALS);
  const [ hasLoaded, setHasLoaded ] = useState(false);
  const [ pageError, setPageError ] = useState<Error>();

  const hasTrials = trialHps.length !== 0;
  const isExperimentTerminal = terminalRunStates.has(experiment.state as RunState);

  const columns = useMemo(() => {
    const idRenderer = (_: string, record: TrialHParams) => {
      const index = trialIds.findIndex(trialId => trialId === record.id);
      const color = index !== -1 ? glasbeyColor(index) : 'rgba(0, 0, 0, 1.0)';
      return (
        <div className={css.idLayout}>
          <div className={css.colorLegend} style={{ backgroundColor: color }} />
          <div>{record.id}</div>
        </div>
      );
    };
    const idSorter = (a: TrialHParams, b: TrialHParams): number => alphanumericSorter(a.id, b.id);
    const idColumn = { key: 'id', render: idRenderer, sorter: idSorter, title: 'Trial ID' };

    const metricRenderer = (_: string, record: TrialHParams) => {
      return record.metric ? <HumanReadableFloat num={record.metric} /> : null;
    };
    const metricSorter = (recordA: TrialHParams, recordB: TrialHParams): number => {
      return numericSorter(recordA.metric || undefined, recordB.metric || undefined);
    };
    const metricColumn = {
      dataIndex: 'metric',
      key: 'metric',
      render: metricRenderer,
      sorter: metricSorter,
      title: <BadgeTag
        label={selectedMetric.name}
        tooltip={selectedMetric.type}>{selectedMetric.type.substr(0, 1).toUpperCase()}</BadgeTag>,
    };

    const hpRenderer = (key: string) => {
      return (_: string, record: TrialHParams) => {
        const value = record.hparams[key];
        const type = experiment.config.hyperparameters[key].type;
        if (typeof value === 'number' && [ 'const', 'double', 'float', 'log' ].includes(type)) {
          return <HumanReadableFloat num={value} />;
        }
        return record.hparams[key];
      };
    };
    const hpColumnSorter = (key: string) => {
      return (recordA: TrialHParams, recordB: TrialHParams): number => {
        const a = recordA.hparams[key];
        const b = recordB.hparams[key];
        return hpSorter(a, b);
      };
    };
    const hpColumns = Object.keys(experiment.config.hyperparameters || {}).map(key => ({
      key,
      render: hpRenderer(key),
      sorter: hpColumnSorter(key),
      title: key,
    }));

    return [ idColumn, metricColumn, ...hpColumns ];
  }, [ experiment.config.hyperparameters, selectedMetric, trialIds ]);

  const resetData = useCallback(() => {
    setChartData([]);
    setTrialHps([]);
    setTrialIds([]);
    setHasLoaded(false);
  }, []);

  const handleTopTrialsChange = useCallback((count: SelectValue) => {
    resetData();
    setMaxTrials(count as number);
  }, [ resetData ]);

  const handleMetricChange = useCallback((metric: MetricName) => {
    if (!onMetricChange) return;
    resetData();
    onMetricChange(metric);
  }, [ onMetricChange, resetData ]);

  const handleTrialClick = useCallback((event: React.MouseEvent, trialId: number) => {
    handlePath(event, { path: `/experiments/${experiment.id}/trials/${trialId}` });
  }, [ experiment.id ]);

  const handleTrialFocus = useCallback((trialId: number | null) => {
    setChartTrialId(trialId != null ? trialId : undefined);
  }, []);

  const handleTableChange = useCallback((tablePagination) => {
    setPageSize(tablePagination.pageSize);
  }, []);

  const handleTableRow = useCallback((record: TrialHParams) => ({
    onClick: (event: React.MouseEvent) => handlePath(event, { path: record.url }),
    onMouseEnter: () => setTableTrialId(record.id),
    onMouseLeave: () => setTableTrialId(undefined),
  }), []);

  const rowClassName = useCallback((record: TrialHParams) => {
    return defaultRowClassName({
      clickable: true,
      highlighted: record.id === chartTrialId,
    });
  }, [ chartTrialId ]);

  useEffect(() => {
    const canceler = new AbortController();
    const trialIdsMap: Record<number, number> = {};
    const trialDataMap: Record<number, number[]> = {};
    const trialHpMap: Record<number, TrialHParams> = {};
    const batchesMap: Record<number, number> = {};
    const metricsMap: Record<number, Record<number, number>> = {};
    const filterTrialMap: Record<number, boolean> = {};

    consumeStream<V1TrialsSampleResponse>(
      detApi.StreamingInternal.determinedTrialsSample(
        experiment.id,
        selectedMetric.name,
        metricTypeParamMap[selectedMetric.type],
        maxTrials,
        MAX_DATAPOINTS,
        undefined,
        undefined,
        undefined,
        { signal: canceler.signal },
      ),
      event => {
        if (!event || !event.trials || !Array.isArray(event.trials)) return;

        /*
         * Cache trial ids, hparams, batches and metric values into easily searchable
         * dictionaries, then construct the necessary data structures to render the
         * chart and the table.
         */

        (event.promotedTrials || []).forEach(trialId => trialIdsMap[trialId] = trialId);
        (event.demotedTrials || []).forEach(trialId => delete trialIdsMap[trialId]);

        (event.trials || []).forEach(trial => {
          const id = trial.trialId;
          const hasHParams = Object.keys(trial.hparams || {}).length !== 0;

          if (hasHParams && !trialHpMap[id]) {
            trialHpMap[id] = {
              hparams: trial.hparams,
              id,
              metric: null,
              url: `/experiments/${experiment.id}/trials/${id}`,
            };
          }

          trialDataMap[id] = trialDataMap[id] || [];
          metricsMap[id] = metricsMap[id] || {};
          filterTrialMap[id] = filterTrialMap[id] || false;

          trial.data.forEach(datapoint => {
            batchesMap[datapoint.batches] = datapoint.batches;
            metricsMap[id][datapoint.batches] = datapoint.value;
            trialHpMap[id].metric = datapoint.value;
            if (datapoint.value > MAX_ALLOWED_METRIC_VALUE) filterTrialMap[id] = true;
          });
        });

        const newTrialHps = Object.values(trialHpMap)
          .map(trialHp => trialHp.id)
          .sort(alphanumericSorter)
          .map(id => trialHpMap[id]);
        setTrialHps(newTrialHps);

        const newBatches = Object.values(batchesMap);
        setBatches(newBatches);

        const newTrialIds = Object.values(trialIdsMap).filter(trialId => !filterTrialMap[trialId]);
        setTrialIds(newTrialIds);

        const newChartData = newTrialIds.map(trialId => newBatches.map(batch => {
          const value = metricsMap[trialId][batch];
          return value != null ? value : null;
        }));
        setChartData(newChartData);

        // One successful event as come through.
        setHasLoaded(true);
      },
    ).catch(e => setPageError(e));

    return () => canceler.abort();
  }, [ experiment.id, maxTrials, selectedMetric ]);

  if (pageError?.message.includes('single-trial experiments are not supported')) {
    return <Alert
      description={<>
        Learn about &nbsp;
        <Link
          external
          path="/docs/reference/experiment-config.html#searcher"
          popout
          size="small">how to run a hyperparameter search</Link>.
      </>}
      message="Hyperparameter visualizations are not applicable for single trial experiments."
      type="warning" />;
  } else if (pageError) {
    return <Message title={pageError.message} />;
  } else if (!hasLoaded) {
    return <Spinner />;
  } else if (!hasTrials && hasLoaded) {
    return isExperimentTerminal ? (
      <Message title="No experiment visualization data to show." type={MessageType.Empty} />
    ) : (
      <div className={css.waiting}>
        <Alert
          description="Please wait until the experiment is further along."
          message="Not enough data points to show yet." />
        <Spinner />
      </div>
    );
  }

  return (
    <div className={css.base}>
      <Section
        options={<ResponsiveFilters>
          <SelectFilter
            enableSearchFilter={false}
            label="Top Trials"
            showSearch={false}
            style={{ width: 70 }}
            value={maxTrials}
            onChange={handleTopTrialsChange}>
            {TOP_TRIALS_OPTIONS.map(option => (
              <Option key={option} value={option}>{option}</Option>
            ))}
          </SelectFilter>
          <MetricSelectFilter
            defaultMetricNames={metrics}
            label="Metric"
            metricNames={metrics}
            multiple={false}
            value={selectedMetric}
            width={'100%'}
            onChange={handleMetricChange} />
        </ResponsiveFilters>}
        title="Learning Curve">
        <div className={css.chart}>
          <LearningCurveChart
            data={chartData}
            focusedTrialId={tableTrialId}
            selectedMetric={selectedMetric}
            trialIds={trialIds}
            xValues={batches}
            onTrialClick={handleTrialClick}
            onTrialFocus={handleTrialFocus} />
        </div>
      </Section>
      <Section title="Trial Hyperparameters">
        <ResponsiveTable<TrialHParams>
          columns={columns}
          dataSource={trialHps}
          pagination={getPaginationConfig(trialHps.length, pageSize)}
          rowClassName={rowClassName}
          rowKey="id"
          scroll={{ x: 1000 }}
          showSorterTooltip={false}
          size="small"
          onChange={handleTableChange}
          onRow={handleTableRow} />
      </Section>
    </div>
  );
};

export default LearningCurve;
