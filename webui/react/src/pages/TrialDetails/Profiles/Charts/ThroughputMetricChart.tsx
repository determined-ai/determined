import React, { useMemo } from 'react';

import Section from 'components/Section';
import UPlotChart from 'components/UPlot/UPlotChart';

import { ChartProps } from '../types';
import { MetricType } from '../types';
import { useFetchMetrics } from '../useFetchMetrics';

const ThroughputMetricChart: React.FC<ChartProps> = ({ getOptionsForMetrics, trial }) => {
  const throughputMetrics = useFetchMetrics(
    trial.id,
    trial.state,
    MetricType.Throughput,
    'samples_per_second',
    undefined,
    undefined,
  );

  const options = useMemo(
    () => getOptionsForMetrics('samples_per_second', throughputMetrics.names),
    [ getOptionsForMetrics, throughputMetrics.names ],
  );

  return (
    <Section bodyBorder bodyNoPadding title="Throughput">
      <UPlotChart data={throughputMetrics.data} options={options} title="Throughput" />
    </Section>
  );
};

export default ThroughputMetricChart;
