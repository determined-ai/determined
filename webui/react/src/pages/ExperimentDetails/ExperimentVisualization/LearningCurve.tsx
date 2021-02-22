import { Alert, Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useEffect, useState } from 'react';

import LearningCurveChart from 'components/LearningCurveChart';
import Message, { MessageType } from 'components/Message';
import MetricSelectFilter from 'components/MetricSelectFilter';
import ResponsiveFilters from 'components/ResponsiveFilters';
import Section from 'components/Section';
import SelectFilter from 'components/SelectFilter';
import Spinner from 'components/Spinner';
import { handlePath, paths } from 'routes/utils';
import { V1TrialsSampleResponse } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { consumeStream } from 'services/utils';
import { ExperimentBase, MetricName, metricTypeParamMap, RunState } from 'types';
import { terminalRunStates } from 'utils/types';

import HpTrialTable, { TrialHParams } from './HpTrialTable';
import css from './LearningCurve.module.scss';

const { Option } = Select;

interface Props {
  experiment: ExperimentBase;
  metrics: MetricName[];
  onMetricChange?: (metric: MetricName) => void;
  selectedMetric: MetricName
}

const DEFAULT_MAX_TRIALS = 100;
const MAX_DATAPOINTS = 5000;
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
  const [ chartTrialId, setChartTrialId ] = useState<number>();
  const [ tableTrialId, setTableTrialId ] = useState<number>();
  const [ maxTrials, setMaxTrials ] = useState(DEFAULT_MAX_TRIALS);
  const [ hasLoaded, setHasLoaded ] = useState(false);
  const [ pageError, setPageError ] = useState<Error>();

  const hasTrials = trialHps.length !== 0;
  const isExperimentTerminal = terminalRunStates.has(experiment.state as RunState);

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
    handlePath(event, { path: paths.trialDetails(trialId, experiment.id) });
  }, [ experiment.id ]);

  const handleTrialFocus = useCallback((trialId: number | null) => {
    setChartTrialId(trialId != null ? trialId : undefined);
  }, []);

  const handleTableMouseEnter = useCallback((event: React.MouseEvent, record: TrialHParams) => {
    if (record.id) setTableTrialId(record.id);
  }, []);

  const handleTableMouseLeave = useCallback(() => {
    setTableTrialId(undefined);
  }, []);

  useEffect(() => {
    const canceler = new AbortController();
    const trialIdsMap: Record<number, number> = {};
    const trialDataMap: Record<number, number[]> = {};
    const trialHpMap: Record<number, TrialHParams> = {};
    const batchesMap: Record<number, number> = {};
    const metricsMap: Record<number, Record<number, number>> = {};

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
        const newTrialIds = Object.values(trialIdsMap);
        setTrialIds(newTrialIds);

        (event.trials || []).forEach(trial => {
          const id = trial.trialId;
          const hasHParams = Object.keys(trial.hparams || {}).length !== 0;

          if (hasHParams && !trialHpMap[id]) {
            trialHpMap[id] = { hparams: trial.hparams, id, metric: null };
          }

          trialDataMap[id] = trialDataMap[id] || [];
          metricsMap[id] = metricsMap[id] || {};

          trial.data.forEach(datapoint => {
            batchesMap[datapoint.batches] = datapoint.batches;
            metricsMap[id][datapoint.batches] = datapoint.value;
            trialHpMap[id].metric = datapoint.value;
          });
        });

        const newTrialHps = newTrialIds.map(id => trialHpMap[id]);
        setTrialHps(newTrialHps);

        const newBatches = Object.values(batchesMap);
        setBatches(newBatches);

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

  if (pageError) {
    return <Message title={pageError.message} />;
  } else if (hasLoaded && !hasTrials) {
    return isExperimentTerminal ? (
      <Message title="No learning curve data to show." type={MessageType.Empty} />
    ) : (
      <div className={css.waiting}>
        <Alert
          description="Please wait until the experiment is further along."
          message="Not enough data points to plot." />
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
        <div className={css.container}>
          {!hasLoaded ? <Spinner /> : (
            <>
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
              <div className={css.table}>
                <HpTrialTable
                  experimentId={experiment.id}
                  highlightedTrialId={chartTrialId}
                  hyperparameters={experiment.config.hyperparameters || {}}
                  metric={selectedMetric}
                  trialHps={trialHps}
                  trialIds={trialIds}
                  onMouseEnter={handleTableMouseEnter}
                  onMouseLeave={handleTableMouseLeave}
                />
              </div>
            </>
          )}
        </div>
      </Section>
    </div>
  );
};

export default LearningCurve;
