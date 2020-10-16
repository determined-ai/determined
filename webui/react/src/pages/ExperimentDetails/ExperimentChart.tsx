import { PlotData } from 'plotly.js/lib/core';
import React, { useMemo } from 'react';

import MetricChart from 'components/MetricChart';
import { ValidationHistory } from 'types';

interface Props {
  id?: string;
  startTime?: string;
  validationMetric?: string;
  validationHistory?: ValidationHistory[];
}

const ExperimentChart: React.FC<Props> = ({ validationMetric, ...props }: Props) => {
  const titleDetail = validationMetric ? ` (${validationMetric})` : '';
  const title = `Best Validation Metric${titleDetail}`;

  const data: Partial<PlotData>[] = useMemo(() => {
    if (!props.startTime || !props.validationHistory) return [];

    const startTime = new Date(props.startTime).getTime();
    const textData: string[] = [];
    const xData: number[] = [];
    const yData: number[] = [];

    props.validationHistory.forEach(validation => {
      const endTime = new Date(validation.endTime).getTime();
      const x = (endTime - startTime) / 1000;
      const y = validation.validationError;
      const text = [
        `Trial ${validation.trialId}`,
        `Elapsed Time: ${x} sec`,
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
      name: validationMetric,
      text: textData,
      type: 'scatter',
      x: xData,
      y: yData,
    } ];
  }, [ props.startTime, props.validationHistory, validationMetric ]);

  return <MetricChart
    data={data}
    id={props.id}
    title={title}
    xLabel="Time Elapsed (sec)"
    yLabel="Metric Value" />;
};

export default ExperimentChart;
