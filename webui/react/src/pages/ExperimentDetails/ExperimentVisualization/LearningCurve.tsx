import { Alert, Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import HumanReadableFloat from 'components/HumanReadableFloat';
import LearningCurveChart from 'components/LearningCurveChart';
import Message from 'components/Message';
import MetricSelectFilter from 'components/MetricSelectFilter';
import ResponsiveFilters from 'components/ResponsiveFilters';
import ResponsiveTable from 'components/ResponsiveTable';
import Section from 'components/Section';
import SelectFilter from 'components/SelectFilter';
import Spinner from 'components/Spinner';
import { defaultRowClassName, getPaginationConfig, MINIMUM_PAGE_SIZE } from 'components/Table';
import { handlePath } from 'routes/utils';
import { V1TrialsSampleResponse, V1TrialsSampleResponseTrial } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { consumeStream } from 'services/utils';
import { ExperimentDetails, MetricName, metricTypeParamMap } from 'types';
import { alphanumericSorter, hpSorter } from 'utils/data';

import css from './LearningCurve.module.scss';

const { Option } = Select;

interface Props {
  experiment: ExperimentDetails;
  metrics: MetricName[];
  onMetricChange?: (metric: MetricName) => void;
  selectedMetric: MetricName
}

type HParams = Record<string, boolean | number | string>;

interface TrialHParams {
  hparams: HParams;
  id: number;
  url: string;
}

const DEFAULT_MAX_TRIALS = 100;
const MAX_DATAPOINTS = 5000;
const TOP_TRIALS_OPTIONS = [ 1, 10, 20, 50, 100, 200, 500 ];

const LearningCurve: React.FC<Props> = ({
  experiment,
  metrics,
  onMetricChange,
  selectedMetric,
}: Props) => {
  const [ trialIds, setTrialIds ] = useState<number[]>([]);
  const [ batches, setBatches ] = useState<number[]>([]);
  const [ chartData, setChartData ] = useState<(number | null)[][]>([]);
  const [ trialHpMap, setTrialHpMap ] = useState<Record<number, HParams>>({});
  const [ trialList, setTrialList ] = useState<Array<V1TrialsSampleResponseTrial>>([]);
  const [ pageSize, setPageSize ] = useState(MINIMUM_PAGE_SIZE);
  const [ chartTrialId, setChartTrialId ] = useState<number>();
  const [ tableTrialId, setTableTrialId ] = useState<number>();
  const [ maxTrials, setMaxTrials ] = useState(DEFAULT_MAX_TRIALS);
  const [ hasLoaded, setHasLoaded ] = useState(false);
  const [ pageError, setPageError ] = useState<Error>();

  const hasTrials = Object.keys(trialHpMap).length !== 0;

  const trialHParams: TrialHParams[] = useMemo(() => {
    if (!trialHpMap) return [];
    return trialIds.map(trialId => ({
      hparams: trialHpMap[trialId],
      id: trialId,
      url: `/trials/${trialId}`,
    }));
  }, [ trialHpMap, trialIds ]);

  const columns = useMemo(() => {
    const idSorter = (a: TrialHParams, b: TrialHParams): number => alphanumericSorter(a.id, b.id);
    const idColumn = { dataIndex: 'id', key: 'id', sorter: idSorter, title: 'Trial ID' };

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

    return [ idColumn, ...hpColumns ];
  }, [ experiment.config.hyperparameters ]);

  const resetData = useCallback(() => {
    setChartData([]);
    setTrialHpMap({});
    setTrialIds([]);
    setTrialList([]);
  }, []);

  const handleTopTrialsChange = useCallback((count: SelectValue) => {
    resetData();
    setMaxTrials(count as number);
  }, [ resetData ]);

  const handleMetricChange = useCallback((metric: MetricName) => {
    if (onMetricChange) onMetricChange(metric);
  }, [ onMetricChange ]);

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

        // Figure out if we need to update the list of trial ids.
        const hasDemotedTrials = event.demotedTrials && event.demotedTrials.length !== 0;
        const hasPromotedTrials = event.promotedTrials && event.promotedTrials.length !== 0;
        if (hasDemotedTrials || hasPromotedTrials) {
          // Update the trial ids based on the list of promotions and demotions.
          const trialIdsSeen = trialIds.reduce((acc, trialId) => {
            acc[trialId] = true;
            return acc;
          }, {} as Record<number, boolean>);
          (event.demotedTrials || []).forEach(trialId => delete trialIdsSeen[trialId]);
          (event.promotedTrials || []).forEach(trialId => trialIdsSeen[trialId] = true);

          // Update trial ids after promotion and demotion applied.
          setTrialIds(Object.keys(trialIdsSeen).map(id => parseInt(id)).sort(alphanumericSorter));
        }

        // Save the trials sample data for post processing.
        setTrialList(event.trials || []);

        // One successful event as come through.
        setHasLoaded(true);
      },
    ).catch(e => setPageError(e));

    return () => canceler.abort();
    /* eslint-disable-next-line react-hooks/exhaustive-deps */
  }, [ experiment.id, maxTrials, selectedMetric ]);

  useEffect(() => {
    const newTrialHpMap: Record<number, HParams> = {};
    const batchesSeen: Record<number, boolean> = {};
    const metricsSeen: Record<number, Record<number, number | null>> = {};

    trialList.forEach(trialData => {
      const id = trialData.trialId;
      if (!id) return;

      const hasHParams = Object.keys(trialData.hparams || {}).length !== 0;
      if (hasHParams && !trialHpMap[id]) newTrialHpMap[id] = trialData.hparams;

      metricsSeen[id] = metricsSeen[id] || {};
      (trialData.data || []).forEach(batchMetric => {
        batchesSeen[batchMetric.batches] = true;
        metricsSeen[id][batchMetric.batches] = batchMetric.value;
      });
    });

    // Update batches with every step batches encountered.
    const newBatches = Object.keys(batchesSeen)
      .map(batch => parseInt(batch))
      .sort(alphanumericSorter);
    setBatches(newBatches);

    // Update the hyperparameters for all of the newly encountered trials.
    if (Object.keys(newTrialHpMap).length !== 0) {
      setTrialHpMap({ ...trialHpMap, ...newTrialHpMap });
    }

    // Construct the data to feed to the chart.
    const newChartData = trialIds.map(trialId => {
      return newBatches.map(batch => {
        const value = metricsSeen[trialId][batch];
        return value != null ? value : null;
      });
    });
    setChartData(newChartData);
    /* eslint-disable-next-line react-hooks/exhaustive-deps */
  }, [ trialIds, trialList ]);

  if (pageError) {
    return <Message title={pageError.message} />;
  } else if (!hasLoaded) {
    return <Spinner />;
  } else if (!hasTrials && hasLoaded) {
    return (
      <>
        <Alert
          description="Please wait until the experiment is further along."
          message="Not enough data points to show yet." />
        <Spinner />
      </>
    );
  }

  return (
    <>
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
        <div className={css.base}>
          <LearningCurveChart
            data={chartData}
            focusedTrialId={tableTrialId}
            trialIds={trialIds}
            xValues={batches}
            onTrialFocus={handleTrialFocus} />
        </div>
      </Section>
      <Section title="Trial Hyperparameters">
        <ResponsiveTable<TrialHParams>
          columns={columns}
          dataSource={trialHParams}
          pagination={getPaginationConfig(trialHParams.length, pageSize)}
          rowClassName={rowClassName}
          rowKey="id"
          scroll={{ x: 1000 }}
          showSorterTooltip={false}
          size="small"
          onChange={handleTableChange}
          onRow={handleTableRow} />
      </Section>
    </>
  );
};

export default LearningCurve;
