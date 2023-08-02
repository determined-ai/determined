import React from 'react';

import { LineChart } from 'components/kit/LineChart';
import { XAxisDomain } from 'components/kit/LineChart/XAxisFilter';
import Section from 'components/Section';
import handleError from 'utils/error';

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
        experimentId={trial.id}
        handleError={handleError}
        series={throughputMetrics.data}
        xAxis={XAxisDomain.Time}
        xLabel="Time"
        yLabel={yLabel}
        yTickValues={getScientificNotationTickValues}
      />
    </Section>
  );
};

export default ThroughputMetricChart;
