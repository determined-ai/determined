import React from 'react';

import { LineChart } from 'components/kit/LineChart';
import Section from 'components/Section';
import { ChartProps, MetricType } from 'pages/TrialDetails/Profiles/types';
import { useFetchProfilerMetrics } from 'pages/TrialDetails/Profiles/useFetchProfilerMetrics';
import { getUnitForMetricName } from 'pages/TrialDetails/Profiles/utils';
import { XAxisDomain } from 'types';
import handleError from 'utils/error';

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
        handleError={handleError}
        series={throughputMetrics.data}
        showLegend
        xAxis={XAxisDomain.Time}
        xLabel="Time"
        yLabel={yLabel}
      />
    </Section>
  );
};

export default ThroughputMetricChart;
