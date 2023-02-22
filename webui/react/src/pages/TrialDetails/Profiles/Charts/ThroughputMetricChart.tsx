import React from 'react';

import { LineChart } from 'components/kit/LineChart';
import { XAxisDomain } from 'components/kit/LineChart/XAxisFilter';
import Section from 'components/Section';
import { getTimeTickValues } from 'utils/chart';

import { ChartProps } from '../types';
import { MetricType } from '../types';
import { useFetchProfilerMetrics } from '../useFetchProfilerMetrics';
import { getScientificNotationTickValues, getUnitForMetricName } from '../utils';

const ThroughputMetricChart: React.FC<ChartProps> = ({ trial }) => {
  const throughputMetrics = useFetchProfilerMetrics(
    trial.id,
    trial.state,
    MetricType.Throughput,
    'samples_per_second',
    undefined,
    undefined,
  );

  const yLabel = getUnitForMetricName('samples_per_second');

  return (
    <Section bodyBorder bodyNoPadding title="Throughput">
      <LineChart
        series={throughputMetrics.data}
        xAxis={XAxisDomain.Time}
        xLabel="Time"
        xTickValues={getTimeTickValues}
        yLabel={yLabel}
        yTickValues={getScientificNotationTickValues}
      />
    </Section>
  );
};

export default ThroughputMetricChart;
