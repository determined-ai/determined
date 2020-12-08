import { PlotData } from 'plotly.js/lib/core';
import React, { useCallback, useMemo, useState } from 'react';

import MetricChart from 'components/MetricChart';
import MetricSelectFilter from 'components/MetricSelectFilter';
import useStorage from 'hooks/useStorage';
import { MetricName, MetricType, RunState, WorkloadWrapper } from 'types';

interface Props {
  id?: string;
  metricNames: MetricName[];
  defaultMetricNames: MetricName[];
  workloads?: WorkloadWrapper[];
  storageKey?: string;
  validationMetric?: string;
}

const STORAGE_PATH = 'trial-detail';

// Plotly.js colors imported from node_modules/plotly.js/src/components/color/attributes.js
const chartColorList = [
  '#1f77b4',  // muted blue
  '#ff7f0e',  // safety orange
  '#2ca02c',  // cooked asparagus green
  '#d62728',  // brick red
  '#9467bd',  // muted purple
  '#8c564b',  // chestnut brown
  '#e377c2',  // raspberry yogurt pink
  '#7f7f7f',  // middle gray
  '#bcbd22',  // curry yellow-green
  '#17becf',  // blue-teal
];

const metricColorByKey: { [metricKey: string]: string } = {};

const getMetricColorByKey = (metricKey: string): string => {
  if (!metricColorByKey[metricKey]) {
    const index = Object.keys(metricColorByKey).length % chartColorList.length;
    metricColorByKey[metricKey] = chartColorList[index];
  }
  return metricColorByKey[metricKey];
};

const TrialChart: React.FC<Props> = ({
  metricNames,
  storageKey,
  validationMetric,
  defaultMetricNames,
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

  const metricsSelected = JSON.stringify([ initMetric ]) !== JSON.stringify(metrics);

  const data: Partial<PlotData>[] = useMemo(() => {
    const dataMap: Record<string, Partial<PlotData>> = {};

    (props.workloads || []).forEach(wlWrapper => {
      metrics.forEach(metric => {
        if (!metric) return;
        const metricsWl = metric.type === MetricType.Training ?
          wlWrapper.training : wlWrapper.validation;
        if (!metricsWl || metricsWl.state !== RunState.Completed) return;

        const source = metricsWl.metrics || {};
        const x = metricsWl.numBatches + metricsWl.priorBatchesProcessed;
        const y = source[metric.name];

        const metricKey = `${metric.type}_${metric.name}`;

        const text = [
          `Batches: ${x}`,
          `Metric Value: ${y}`,
        ].join('<br>');

        if (text && x && y) {
          if (!dataMap[metricKey]) {
            dataMap[metricKey] = {
              hovertemplate: '%{text}<extra></extra>',
              line: { color: getMetricColorByKey(metricKey) },
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
  }, [ metrics, props.workloads ]);

  const handleMetricChange = useCallback((value: MetricName[]) => {
    setMetrics(value);

    if (storageKey) storage.set(storageKey, value);
  }, [ storage, storageKey ]);

  return <MetricChart
    data={data}
    id={props.id}
    metricsSelected={metricsSelected}
    options={<MetricSelectFilter
      defaultMetricNames={defaultMetricNames}
      metricNames={metricNames}
      multiple
      value={metrics}
      onChange={handleMetricChange} />}
    title="Metrics"
    xLabel="Batches"
    yLabel="Metric Value" />;
};

export default TrialChart;
