import React, { useMemo } from 'react';

import Section from 'components/Section';
import UPlotChart from 'components/UPlot/UPlotChart';

import { OldChartProps } from '../types';
import { MetricType } from '../types';
import { useOldFetchProfilerMetrics } from '../useOldFetchProfilerMetrics';

const ThroughputMetricChart: React.FC<OldChartProps> = ({ getOptionsForMetrics, trial }) => {
  const throughputMetrics = useOldFetchProfilerMetrics(
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
      <UPlotChart data={throughputMetrics.data} options={options} />
    </Section>
  );
};

export default ThroughputMetricChart;
