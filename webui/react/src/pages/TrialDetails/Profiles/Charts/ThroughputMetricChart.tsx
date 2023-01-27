import React, { useMemo } from 'react';

import { LineChart } from 'components/kit/LineChart';
import { XAxisDomain } from 'components/kit/LineChart/XAxisFilter';
import Section from 'components/Section';

import { ChartProps } from '../types';
import { MetricType } from '../types';
import { useFetchProfilerMetrics } from '../useFetchProfilerMetrics';

const ThroughputMetricChart: React.FC<ChartProps> = ({ getOptionsForMetrics, trial }) => {
  const throughputMetrics = useFetchProfilerMetrics(
    trial.id,
    trial.state,
    MetricType.Throughput,
    'samples_per_second',
    undefined,
    undefined,
  );

  const options = useMemo(
    () => getOptionsForMetrics('samples_per_second', throughputMetrics.names),
    [getOptionsForMetrics, throughputMetrics.names],
  );

  return (
    <Section bodyBorder bodyNoPadding title="Throughput">
      <LineChart
        series={throughputMetrics.data}
        xAxis={XAxisDomain.Time}
        xLabel="Time"
        yLabel={options.axes?.[1].label ?? ''}
      />
    </Section>
  );
};

export default ThroughputMetricChart;
