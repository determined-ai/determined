import React from 'react';

import { LineChart } from 'components/kit/LineChart';
import Section from 'components/Section';

import { ChartProps } from '../types';
import { MetricType } from '../types';
import { useFetchProfilerMetrics } from '../useFetchProfilerMetrics';

const ThroughputMetricChart: React.FC<ChartProps> = ({ trial }) => {
  const throughputMetrics = useFetchProfilerMetrics(
    trial.id,
    trial.state,
    MetricType.Throughput,
    'samples_per_second',
    undefined,
    undefined,
  );

  return (
    <Section bodyBorder bodyNoPadding title="Throughput">
      <LineChart series={throughputMetrics.data} xAxis="Time" />
    </Section>
  );
};

export default ThroughputMetricChart;
