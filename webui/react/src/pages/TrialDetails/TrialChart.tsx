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
  const [ metric, setMetric ] = useState<MetricName | undefined>(defaultMetric || fallbackMetric);

  const data: Partial<PlotData>[] = useMemo(() => {
    const textData: string[] = [];
    const xData: number[] = [];
    const yData: number[] = [];

    (props.steps || []).forEach(step => {
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
        textData.push(text);
        xData.push(x);
        yData.push(y);
      }
    });

    return [ {
      hovermode: 'y unified',
      hovertemplate: '%{text}<extra></extra>',
      mode: 'lines+markers',
      text: textData,
      type: 'scatter',
      x: xData,
      y: yData,
    } ];
  }, [ metric, props.steps ]);

  const handleMetricChange = useCallback((value: MetricName) => setMetric(value), []);

  return <MetricChart
    data={data}
    id={props.id}
    options={<MetricSelectFilter
      metricNames={metricNames}
      value={metric}
      onChange={handleMetricChange} />}
    title="Metrics"
    xLabel="Batches"
    yLabel="Metric Value" />;
};

export default TrialChart;
