import { useCallback, useEffect, useState } from 'react';

// eslint-disable-next-line import/order
import { timeSeries } from 'services/api';
type ChartData = (number | null)[][];
type MetricKey = string;

type MetricInfo = {
  chartData: ChartData;
  nonEmptyTrials: Set<number>;
};
interface LearningCurveData {
  batches: number[];
  infoForMetrics: Record<MetricKey, MetricInfo>;
}
import { Metric } from 'types';
import { metricToKey } from 'utils/metric';

const BATCH_PADDING = 10;

const emptyChartData = (rows: number, columns: number): ChartData =>
  [...Array(rows)].map(() => Array(columns).fill(null));

const seq = (n: number) => [...Array(n)].map((_, i) => i + 1);

const useLearningCurve = (
  trialIds: number[],
  metrics: Metric[],
  maxBatch: number,
): LearningCurveData | undefined => {
  const [learningCurveData, setLearningCurveData] = useState<LearningCurveData>();
  const fetchSeriesData = useCallback(async () => {
    if (!trialIds || !metrics.length) return;

    // preparing the new data structure to store API response
    const metricKeys = metrics.map((metric: Metric) => metricToKey(metric));

    const infoForMetrics: Record<MetricKey, MetricInfo> = metricKeys
      .map((metricKey) => ({
        [metricKey]: {
          chartData: emptyChartData(trialIds.length, maxBatch + BATCH_PADDING),
          nonEmptyTrials: new Set<number>(),
        },
      }))
      .reduce((a, b) => ({ ...a, ...b }), {});

    const newLearningCurveData: LearningCurveData = {
      batches: seq(maxBatch + BATCH_PADDING),
      infoForMetrics,
    };

    // calling the API
    const trials = await timeSeries({
      maxDatapoints: 1000,
      metricNames: metrics,
      trialIds: trialIds,
    });

    // populating the data structure with the API results

    trials.forEach((trial) => {
      const trialRowIndex = trialIds.indexOf(trial.id);
      if (trialRowIndex === -1) return;
      trial.metrics.forEach((metric) => {
        const metricKey = metricToKey(metric);
        const metricInfo = newLearningCurveData.infoForMetrics[metricKey];
        if (!metricInfo) return;
        metric.data.forEach(({ batches, value }) => {
          const batchColumnIndex = batches - 1;
          if (batchColumnIndex >= 0 && batches <= maxBatch) {
            metricInfo.nonEmptyTrials.add(trial.id);
            const chartData = metricInfo.chartData;
            chartData[trialRowIndex][batchColumnIndex] = value;
          }
        });
      });
    });

    setLearningCurveData(newLearningCurveData);
  }, [trialIds, metrics, maxBatch]);

  useEffect(() => {
    fetchSeriesData();
  }, [fetchSeriesData]);

  return learningCurveData;
};

export default useLearningCurve;
