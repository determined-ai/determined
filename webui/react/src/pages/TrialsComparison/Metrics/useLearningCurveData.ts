import { useCallback, useEffect, useState } from 'react';

import { compareTrials } from 'services/api';
type ChartData = (number | null)[][]
interface SeriesData {
  batches: number[];
  metrics: Record<string, ChartData>
}
import {

  Metric,
} from 'types';
import { metricToKey } from 'utils/metric';

const BATCH_PADDING = 50;

const emptyChartData = (rows: number, columns: number): ChartData =>
  [ ...Array(rows) ].map(() => Array(columns).fill(null));

const seq = (n: number) => [ ...Array(n) ].map((_, i) => i + 1);

const useLearningCurve = (
  trialIds: number[],
  metrics: Metric[],
  maxBatch: number,
): SeriesData | undefined => {
  const [ seriesData, setSeriesData ] = useState<SeriesData>();
  const fetchSeriesData = useCallback(async () => {
    if (!trialIds || !metrics.length) return;

    // preparing the new data structure to store API response
    const metricKeys = metrics.map((metric: Metric) => metricToKey(metric));

    const metricValsMap: Record<string, ChartData> = metricKeys
      .map((metricKey) =>
        ({ [metricKey]: emptyChartData(trialIds.length, maxBatch + BATCH_PADDING) }))
      .reduce((a, b) => ({ ...a, ...b }), {});

    const newSeriesData: SeriesData = {
      batches: seq(maxBatch + BATCH_PADDING),
      metrics: metricValsMap,
    };

    // calling the API
    const trials = await compareTrials({
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
        if (!newSeriesData.metrics[metricKey]) return;
        metric.data.forEach(({ batches, value }) => {
          newSeriesData.metrics[metricKey][trialRowIndex][batches] = value;
        });
      });
    });
    setSeriesData(newSeriesData);
  }, [ trialIds, metrics, maxBatch ]);

  useEffect(() => {
    fetchSeriesData();
  }, [ fetchSeriesData ]);

  return seriesData;
};

export default useLearningCurve;
