import { Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import { PlotData } from 'plotly.js/lib/core';
import React, { useCallback, useMemo, useState } from 'react';

import MetricChart from 'components/MetricChart';
import SelectFilter from 'components/SelectFilter';
import { MetricNames, Step } from 'types';

const { OptGroup, Option } = Select;

interface Props {
  id?: string;
  metricNames: MetricNames;
  steps?: Step[];
  validationMetric?: string;
}

const TrialChart: React.FC<Props> = ({ metricNames, validationMetric, ...props }: Props) => {
  const titleDetail = validationMetric ? ` (${validationMetric})` : '';
  const title = `Training Metric${titleDetail}`;
  const [ metric, setMetric ] = useState(validationMetric);

  const data: Partial<PlotData>[] = useMemo(() => {
    const textData: string[] = [];
    const xData: number[] = [];
    const yData: number[] = [];

    (props.steps || []).forEach(step => {
      if (!metric) return;

      const metricSources = [
        step.avgMetrics || {},
        step.validation?.metrics?.validationMetrics || {},
      ];
      const x = step.numBatches + step.priorBatchesProcessed;
      const y = (metricSources.find(source => source[metric] != null) || {})[metric];

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

  const handleMetricSelect = useCallback((newValue: SelectValue) => {
    setMetric(newValue as string);
  }, []);

  const options = (
    <SelectFilter
      enableSearchFilter={false}
      label="Metric"
      showSearch={false}
      value={metric}
      onSelect={handleMetricSelect}>
      {metricNames.validation.length > 0 && <OptGroup label="Validation Metrics">
        {metricNames.validation.map(key => <Option key={key} value={key}>{key}</Option>)}
      </OptGroup>}
      {metricNames.training.length > 0 && <OptGroup label="Training Metrics">
        {metricNames.training.map(key => <Option key={key} value={key}>{key}</Option>)}
      </OptGroup>}
    </SelectFilter>
  );

  return <MetricChart
    data={data}
    id={props.id}
    options={options}
    title={title}
    xLabel="Batches"
    yLabel="Metric Value" />;
};

export default TrialChart;
