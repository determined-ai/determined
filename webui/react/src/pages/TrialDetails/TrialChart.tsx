import { PlotData } from 'plotly.js/lib/core';
import React, { useCallback, useMemo, useState } from 'react';

import MetricChart from 'components/MetricChart';
import MetricSelectFilter from 'components/MetricSelectFilter';
import useStorage from 'hooks/useStorage';
import { MetricName, MetricType, Step } from 'types';

interface Props {
  id?: string;
  metricNames: MetricName[];
  steps?: Step[];
  storageKey?: string;
  validationMetric?: string;
}

const STORAGE_PATH = 'trial-detail';

const TrialChart: React.FC<Props> = ({
  metricNames,
  storageKey,
  validationMetric,
  ...props
}: Props) => {
  const storage = useStorage(STORAGE_PATH);
  const defaultMetric = metricNames.find(metricName => {
    return metricName.name === validationMetric && metricName.type === MetricType.Validation;
  });
  const fallbackMetric = metricNames && metricNames.length !== 0 ? metricNames[0] : undefined;
  const initMetric = defaultMetric || fallbackMetric;
  const initMetrics = storage.getWithDefault(storageKey || '', initMetric ? [ initMetric ] : []);
  const [ metrics, setMetrics ] = useState<MetricName[]>(initMetrics);

  const data: Partial<PlotData>[] = useMemo(() => {
    const dataMap: Record<string, Partial<PlotData>> = {};

    (props.steps || []).forEach(step => {
      metrics.forEach(metric => {
        if (!metric) return;

        const trainingSource = step.avgMetrics || {};
        const validationSource = step.validation?.metrics?.validationMetrics || {};
        const x = step.numBatches + step.priorBatchesProcessed;
        const y = metric.type === MetricType.Validation ?
          validationSource[metric.name] : trainingSource[metric.name];

        const metricKey = `${metric.type}_${metric.name}`;

        const text = [
          `Batches: ${x}`,
          `Metric Value: ${y}`,
        ].join('<br>');

        if (text && x && y) {
          if (!dataMap[metricKey]) {
            dataMap[metricKey] = {
              hovertemplate: '%{text}<extra></extra>',
              mode: 'lines+markers',
              name: `${metric.name} (${metric.type})`,
              text: [],
              type: 'scatter',
              x: [],
              y: [],
            };
          }
          (dataMap[metricKey].text as string[]).push(text);
          (dataMap[metricKey].x as number[]).push(x);
          (dataMap[metricKey].y as number[]).push(y);
        }
      });
    });

    return Object.values(dataMap).reduce((acc, value) => {
      acc.push(value);
      return acc;
    }, [] as Partial<PlotData>[]);
  }, [ metrics, props.steps ]);

  const handleMetricChange = useCallback((value: MetricName[]) => {
    setMetrics(value);

    if (storageKey) storage.set(storageKey, value);
  }, [ storage, storageKey ]);

  return <MetricChart
    data={data}
    id={props.id}
    options={<MetricSelectFilter
      metricNames={metricNames}
      multiple
      value={metrics}
      onChange={handleMetricChange} />}
    title="Metrics"
    xLabel="Batches"
    yLabel="Metric Value" />;
};

export default TrialChart;
