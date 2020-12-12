import React, { useCallback, useEffect, useState } from 'react';

import LearningCurveChart from 'components/LearningCurveChart';
import MetricSelectFilter from 'components/MetricSelectFilter';
import Section from 'components/Section';
import { V1TrialsSampleResponse, V1TrialsSampleResponseTrial } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { consumeStream } from 'services/utils';
import { ExperimentDetails, MetricName, metricTypeParamMap } from 'types';
import { alphanumericSorter } from 'utils/data';

import css from './LearningCurve.module.scss';

interface Props {
  experiment: ExperimentDetails;
  metrics: MetricName[];
  onMetricChange?: (metric: MetricName) => void;
  selectedMetric: MetricName
}

type HParams = Record<string, boolean | number | string>;

const MAX_TRIALS = 100;
const MAX_DATAPOINTS = 5000;

const LearningCurve: React.FC<Props> = ({
  experiment,
  metrics,
  onMetricChange,
  selectedMetric,
}: Props) => {
  const [ trialIds, setTrialIds ] = useState<number[]>([]);
  const [ batches, setBatches ] = useState<number[]>([]);
  const [ chartData, setChartData ] = useState<(number | null)[][]>([]);
  const [ trialHParams, setTrialHParams ] = useState<Record<number, HParams>>({});
  const [ trialList, setTrialList ] = useState<Array<V1TrialsSampleResponseTrial>>([]);

  const handleMetricChange = useCallback((metric: MetricName) => {
    if (onMetricChange) onMetricChange(metric);
  }, [ onMetricChange ]);

  useEffect(() => {
    const canceler = new AbortController();

    consumeStream<V1TrialsSampleResponse>(
      detApi.StreamingInternal.determinedTrialsSample(
        experiment.id,
        selectedMetric.name,
        metricTypeParamMap[selectedMetric.type],
        MAX_TRIALS,
        MAX_DATAPOINTS,
        undefined,
        undefined,
        undefined,
        { signal: canceler.signal },
      ),
      event => {
        if (!event.trials || !Array.isArray(event.trials)) return;

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
      },
    );

    return () => canceler.abort();
    /* eslint-disable-next-line react-hooks/exhaustive-deps */
  }, [ experiment.id, selectedMetric ]);

  useEffect(() => {
    const newTrialHParams: Record<number, HParams> = {};
    const batchesSeen: Record<number, boolean> = {};
    const metricsSeen: Record<number, Record<number, number | null>> = {};

    trialList.forEach(trialData => {
      const id = trialData.trialId;
      if (!id) return;

      const hasHParams = Object.keys(trialData.hparams || {}).length !== 0;
      if (hasHParams && !trialHParams[id]) newTrialHParams[id] = trialData.hparams;

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
    if (Object.keys(newTrialHParams).length !== 0) {
      setTrialHParams({ ...trialHParams, ...newTrialHParams });
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

  return (
    <Section
      options={<MetricSelectFilter
        defaultMetricNames={metrics}
        label="Metric"
        metricNames={metrics}
        multiple={false}
        value={selectedMetric}
        width={'100%'}
        onChange={handleMetricChange} />}
      title="Learning Curve">
      <div className={css.base}>
        <LearningCurveChart data={chartData} trialIds={trialIds} xValues={batches} />
      </div>
    </Section>
  );
};

export default LearningCurve;
