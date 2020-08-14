import { Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import { PlotData } from 'plotly.js/lib/core';
import React, { useCallback, useMemo, useState } from 'react';

import MetricChart from 'components/MetricChart';
import SelectFilter from 'components/SelectFilter';
import { MetricName, MetricType, Step } from 'types';
import { metricNameToValue, valueToMetricName } from 'utils/trial';

const { OptGroup, Option } = Select;

interface Props {
  id?: string;
  metricNames: MetricName[];
  steps?: Step[];
  validationMetric?: string;
}

const TrialChart: React.FC<Props> = ({ metricNames, validationMetric, ...props }: Props) => {
  const titleDetail = validationMetric ? ` (${validationMetric})` : '';
  const title = `Training Metric${titleDetail}`;
  const [ metric, setMetric ] = useState<MetricName | undefined>(
    validationMetric ? { name: validationMetric, type: MetricType.Validation } : undefined,
  );

  const trainingMetricNames = useMemo(() => {
    return metricNames.filter(metric => metric.type === MetricType.Training);
  }, [ metricNames ]);

  const validationMetricNames = useMemo(() => {
    return metricNames.filter(metric => metric.type === MetricType.Validation);
  }, [ metricNames ]);

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

  const handleMetricSelect = useCallback((value: SelectValue) => {
    setMetric(valueToMetricName(value as string));
  }, []);

  const options = (
    <SelectFilter
      enableSearchFilter={false}
      label="Metric"
      showSearch={false}
      value={metric && metricNameToValue(metric)}
      onSelect={handleMetricSelect}>
      {validationMetricNames.length > 0 && <OptGroup label="Validation Metrics">
        {validationMetricNames.map(key => {
          const value = metricNameToValue(key);
          return <Option key={value} value={value}>{key.name} [{key.type}]</Option>;
        })}
      </OptGroup>}
      {trainingMetricNames.length > 0 && <OptGroup label="Training Metrics">
        {trainingMetricNames.map(key => {
          const value = metricNameToValue(key);
          return <Option key={value} value={value}>{key.name} [{key.type}]</Option>;
        })}
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
