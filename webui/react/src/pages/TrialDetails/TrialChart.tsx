import { PlotData } from 'plotly.js/lib/core';
import React, { useCallback, useMemo, useState } from 'react';

import MetricChart from 'components/MetricChart';
import MetricSelectFilter from 'components/MetricSelectFilter';
import { MetricName, MetricType, Step } from 'types';

interface Props {
  id?: string;
  metricNames: MetricName[];
  steps?: Step[];
  validationMetric?: string;
}

const TrialChart: React.FC<Props> = ({ metricNames, validationMetric, ...props }: Props) => {
  const defaultMetric = metricNames.find(metricName => {
    return metricName.name === validationMetric && metricName.type === MetricType.Validation;
  });
  const fallbackMetric = metricNames && metricNames.length !== 0 ? metricNames[0] : undefined;
  const initMetric = defaultMetric || fallbackMetric;
  const [ metrics, setMetrics ] = useState<MetricName[]>(initMetric ? [ initMetric ] : []);

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

        const text = [
          `Batches: ${x}`,
          `Metric Value: ${y}`,
        ].join('<br>');

        if (text && x && y) {
          if (!dataMap[metric.name]) {
            dataMap[metric.name] = {
              hovertemplate: '%{text}<extra></extra>',
              mode: 'lines+markers',
              name: metric.name,
              text: [],
              type: 'scatter',
              x: [],
              y: [],
            };
          }
          (dataMap[metric.name].text as string[]).push(text);
          (dataMap[metric.name].x as number[]).push(x);
          (dataMap[metric.name].y as number[]).push(y);
        }
      });
    });

    return Object.values(dataMap).reduce((acc, value) => {
      acc.push(value);
      return acc;
    }, [] as Partial<PlotData>[]);
  }, [ metrics, props.steps ]);

  const handleMetricChange = useCallback((value: MetricName[]) => setMetrics(value), []);

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
