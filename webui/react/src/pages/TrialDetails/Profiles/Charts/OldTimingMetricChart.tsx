import React, { useMemo } from 'react';

import Section from 'components/Section';
import UPlotChart from 'components/UPlot/UPlotChart';

import { OldChartProps } from '../types';
import { MetricType } from '../types';
import { useOldFetchProfilerMetrics } from '../useOldFetchProfilerMetrics';

export const TimingMetricChart: React.FC<OldChartProps> = ({ trial, getOptionsForMetrics }) => {
  const timingMetrics = useOldFetchProfilerMetrics(trial.id, trial.state, MetricType.Timing);

  const options = useMemo(
    () => getOptionsForMetrics('seconds', timingMetrics.names),
    [getOptionsForMetrics, timingMetrics.names],
  );

  return (
    <Section bodyBorder bodyNoPadding title="Timing Metrics">
      <UPlotChart data={timingMetrics.data} options={options} />
    </Section>
  );
};

export default TimingMetricChart;
