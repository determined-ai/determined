import React from 'react';

import { LineChart } from 'components/kit/LineChart';
import Section from 'components/Section';
import { ChartProps, MetricType } from 'pages/TrialDetails/Profiles/types';
import { useFetchProfilerMetrics } from 'pages/TrialDetails/Profiles/useFetchProfilerMetrics';
import {
  // getScientificNotationTickValues,
  getUnitForMetricName,
} from 'pages/TrialDetails/Profiles/utils';
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
        // experimentId={trial.id}
        handleError={handleError}
        series={throughputMetrics.data}
        xAxis={XAxisDomain.Time}
        xLabel="Time"
        yLabel={yLabel}
        // yTickValues={getScientificNotationTickValues}
      />
    </Section>
  );
};

export default ThroughputMetricChart;
