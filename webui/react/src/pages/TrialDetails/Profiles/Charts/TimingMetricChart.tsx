import React from 'react';

import { LineChart } from 'components/kit/LineChart';
import Section from 'components/Section';

import { ChartProps } from '../types';
import { MetricType } from '../types';
import { useFetchProfilerMetrics } from '../useFetchProfilerMetrics';

export const TimingMetricChart: React.FC<ChartProps> = ({ trial }) => {
  const timingMetrics = useFetchProfilerMetrics(trial.id, trial.state, MetricType.Timing);

  return (
    <Section bodyBorder bodyNoPadding title="Timing Metrics">
      <LineChart series={timingMetrics.data} xAxis="Time" />
    </Section>
  );
};

export default TimingMetricChart;
