import React, { useMemo } from 'react';

import Section from 'components/Section';
import UPlotChart from 'components/UPlot/UPlotChart';

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
      <UPlotChart
        data={timingMetrics.data}
        noDataMessage="No data found. Timing metrics may not be available for your framework."
        options={options}
      />
    </Section>
  );
};

export default TimingMetricChart;
