import React, { useMemo } from 'react';

import { LineChart } from 'components/kit/LineChart';
import { XAxisDomain } from 'components/kit/LineChart/XAxisFilter';
import Section from 'components/Section';

import { ChartProps } from '../types';
import { MetricType } from '../types';
import { useFetchProfilerMetrics } from '../useFetchProfilerMetrics';

export const TimingMetricChart: React.FC<ChartProps> = ({ trial, getOptionsForMetrics }) => {
  const timingMetrics = useFetchProfilerMetrics(trial.id, trial.state, MetricType.Timing);

  const options = useMemo(
    () => getOptionsForMetrics('seconds', timingMetrics.names),
    [getOptionsForMetrics, timingMetrics.names],
  );

  return (
    <Section bodyBorder bodyNoPadding title="Timing Metrics">
      <LineChart
        series={timingMetrics.data}
        xAxis={XAxisDomain.Time}
        xLabel="Time"
        yLabel={options.axes?.[1].label ?? ''}
      />
    </Section>
  );
};

export default TimingMetricChart;
